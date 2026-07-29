package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
	"golang.org/x/crypto/ssh"
)

type sftpConfig struct {
	host               string
	port               int
	username           string
	password           string
	privateKey         string
	passphrase         string
	rootPath           string
	hostKeyFingerprint string
	publicBaseURL      string
}

type sftpBackend struct{ config sftpConfig }

func newSFTPStorageProvider() *pluginsdk.StorageProvider {
	return pluginsdk.NewStorageProvider("storage.sftp", func() (pluginsdk.StorageBackend, error) {
		config, err := sftpConfigFromEnv()
		if err != nil {
			return nil, err
		}
		return &sftpBackend{config: config}, nil
	})
}

func sftpConfigFromEnv() (sftpConfig, error) {
	config := sftpConfig{
		host:               strings.TrimSpace(os.Getenv("SFORUM_SETTING_HOST")),
		username:           strings.TrimSpace(os.Getenv("SFORUM_SETTING_USERNAME")),
		password:           os.Getenv("SFORUM_SETTING_PASSWORD"),
		privateKey:         os.Getenv("SFORUM_SETTING_PRIVATE_KEY"),
		passphrase:         os.Getenv("SFORUM_SETTING_PASSPHRASE"),
		rootPath:           strings.TrimSpace(os.Getenv("SFORUM_SETTING_ROOT_PATH")),
		hostKeyFingerprint: strings.TrimSpace(os.Getenv("SFORUM_SETTING_HOST_KEY_FINGERPRINT")),
		publicBaseURL:      strings.TrimSpace(os.Getenv("SFORUM_SETTING_PUBLIC_BASE_URL")),
		port:               settingPort("SFORUM_SETTING_PORT", 22),
	}
	if config.rootPath == "" {
		config.rootPath = "/"
	}
	if config.host == "" || config.username == "" {
		return sftpConfig{}, fmt.Errorf("host and username are required")
	}
	if strings.TrimSpace(config.password) == "" && strings.TrimSpace(config.privateKey) == "" {
		return sftpConfig{}, fmt.Errorf("password or private key is required")
	}
	if normalizeFingerprint(config.hostKeyFingerprint) == "" {
		return sftpConfig{}, fmt.Errorf("SSH host key fingerprint is required")
	}
	return config, nil
}

func (b *sftpBackend) Probe() error {
	client, sshClient, err := b.connect()
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()
	_, err = client.Stat(b.config.rootPath)
	return err
}

func (b *sftpBackend) Put(key, contentType string, reader io.Reader) error {
	_ = contentType
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return err
	}
	client, sshClient, err := b.connect()
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()
	if err := client.MkdirAll(path.Dir(remotePath)); err != nil {
		return err
	}
	file, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		_ = client.Remove(remotePath)
		return err
	}
	return file.Close()
}

func (b *sftpBackend) Open(key string) (io.ReadCloser, error) {
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return nil, err
	}
	client, sshClient, err := b.connect()
	if err != nil {
		return nil, err
	}
	file, err := client.Open(remotePath)
	if err != nil {
		_ = client.Close()
		_ = sshClient.Close()
		return nil, err
	}
	return &sftpReadCloser{File: file, client: client, sshClient: sshClient}, nil
}

func (b *sftpBackend) Delete(key string) error {
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return err
	}
	client, sshClient, err := b.connect()
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()
	return client.Remove(remotePath)
}

func (b *sftpBackend) Stat(key string) (pluginsdk.StorageObjectInfo, error) {
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return pluginsdk.StorageObjectInfo{}, err
	}
	client, sshClient, err := b.connect()
	if err != nil {
		return pluginsdk.StorageObjectInfo{}, err
	}
	defer sshClient.Close()
	defer client.Close()
	info, err := client.Stat(remotePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return pluginsdk.StorageObjectInfo{Exists: false}, nil
		}
		return pluginsdk.StorageObjectInfo{}, err
	}
	return pluginsdk.StorageObjectInfo{Exists: true, Size: info.Size(), ModifiedUnix: info.ModTime().UTC().Unix()}, nil
}

func (b *sftpBackend) Exists(key string) (bool, error) {
	info, err := b.Stat(key)
	return info.Exists, err
}

func (b *sftpBackend) PublicURL(key string) string {
	return pluginsdk.JoinStoragePublicURL(b.config.publicBaseURL, key)
}

func (b *sftpBackend) SignedURL(key string, ttl time.Duration) (string, error) {
	_ = ttl
	return b.PublicURL(key), nil
}

func (b *sftpBackend) connect() (*sftp.Client, *ssh.Client, error) {
	auth, err := b.authMethods()
	if err != nil {
		return nil, nil, err
	}
	config := &ssh.ClientConfig{User: b.config.username, Auth: auth, HostKeyCallback: b.hostKeyCallback(), Timeout: 10 * time.Second}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(context.Background(), "tcp", fmt.Sprintf("%s:%d", b.config.host, b.config.port))
	if err != nil {
		return nil, nil, err
	}
	sshConn, channels, requests, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:%d", b.config.host, b.config.port), config)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	sshClient := ssh.NewClient(sshConn, channels, requests)
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, err
	}
	return client, sshClient, nil
}

func (b *sftpBackend) authMethods() ([]ssh.AuthMethod, error) {
	methods := []ssh.AuthMethod{}
	if strings.TrimSpace(b.config.privateKey) != "" {
		var signer ssh.Signer
		var err error
		if strings.TrimSpace(b.config.passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(b.config.privateKey), []byte(b.config.passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(b.config.privateKey))
		}
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if strings.TrimSpace(b.config.password) != "" {
		methods = append(methods, ssh.Password(b.config.password))
	}
	return methods, nil
}

func (b *sftpBackend) hostKeyCallback() ssh.HostKeyCallback {
	expected := normalizeFingerprint(b.config.hostKeyFingerprint)
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		actual := normalizeFingerprint(ssh.FingerprintSHA256(key))
		if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
			return fmt.Errorf("SFTP host key fingerprint mismatch for %s (%s)", hostname, remote.String())
		}
		return nil
	}
}

func normalizeFingerprint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "SHA256:") {
		return value
	}
	decoded, err := hex.DecodeString(strings.ReplaceAll(value, ":", ""))
	if err == nil {
		sum := sha256.Sum256(decoded)
		return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
	}
	return value
}

type sftpReadCloser struct {
	*sftp.File
	client    *sftp.Client
	sshClient *ssh.Client
}

func (r *sftpReadCloser) Close() error {
	err := r.File.Close()
	if clientErr := r.client.Close(); err == nil {
		err = clientErr
	}
	if sshErr := r.sshClient.Close(); err == nil {
		err = sshErr
	}
	return err
}

func settingPort(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fallback
	}
	return port
}
