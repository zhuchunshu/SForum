package storage

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPAdapter struct {
	config SFTPConfig
}

func NewSFTPAdapter(config SFTPConfig) (*SFTPAdapter, error) {
	if strings.TrimSpace(config.Host) == "" || strings.TrimSpace(config.Username) == "" {
		return nil, ErrInvalidConfig
	}
	if strings.TrimSpace(config.Password) == "" && strings.TrimSpace(config.PrivateKey) == "" {
		return nil, ErrInvalidConfig
	}
	if config.Port <= 0 {
		config.Port = 22
	}
	return &SFTPAdapter{config: config}, nil
}

func (a *SFTPAdapter) Put(ctx context.Context, key string, input PutInput) error {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return err
	}
	client, sshClient, err := a.connect(ctx)
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
	if _, err := io.Copy(file, input.Reader); err != nil {
		_ = file.Close()
		_ = client.Remove(remotePath)
		return err
	}
	return file.Close()
}

func (a *SFTPAdapter) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return nil, err
	}
	client, sshClient, err := a.connect(ctx)
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

func (a *SFTPAdapter) Delete(ctx context.Context, key string) error {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return err
	}
	client, sshClient, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()
	return client.Remove(remotePath)
}

func (a *SFTPAdapter) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return ObjectInfo{}, err
	}
	client, sshClient, err := a.connect(ctx)
	if err != nil {
		return ObjectInfo{}, err
	}
	defer sshClient.Close()
	defer client.Close()
	info, err := client.Stat(remotePath)
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, Size: info.Size(), ModifiedAt: info.ModTime()}, nil
}

func (a *SFTPAdapter) Exists(ctx context.Context, key string) (bool, error) {
	_, err := a.Stat(ctx, key)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (a *SFTPAdapter) PublicURL(key string) string {
	return joinPublicURL(a.config.PublicBaseURL, key)
}

func (a *SFTPAdapter) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return a.PublicURL(key), nil
}

func (a *SFTPAdapter) Probe(ctx context.Context) error {
	client, sshClient, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()
	_, err = client.Stat(firstNonBlank(a.config.RootPath, "."))
	return err
}

func (a *SFTPAdapter) connect(ctx context.Context) (*sftp.Client, *ssh.Client, error) {
	auth, err := a.authMethods()
	if err != nil {
		return nil, nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User:            a.config.Username,
		Auth:            auth,
		HostKeyCallback: a.hostKeyCallback(),
		Timeout:         10 * time.Second,
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", a.config.Host, a.config.Port))
	if err != nil {
		return nil, nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:%d", a.config.Host, a.config.Port), sshConfig)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, err
	}
	return client, sshClient, nil
}

func (a *SFTPAdapter) authMethods() ([]ssh.AuthMethod, error) {
	methods := []ssh.AuthMethod{}
	if strings.TrimSpace(a.config.PrivateKey) != "" {
		key := []byte(a.config.PrivateKey)
		var signer ssh.Signer
		var err error
		if strings.TrimSpace(a.config.Passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(a.config.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if strings.TrimSpace(a.config.Password) != "" {
		methods = append(methods, ssh.Password(a.config.Password))
	}
	return methods, nil
}

func (a *SFTPAdapter) hostKeyCallback() ssh.HostKeyCallback {
	expected := normalizeFingerprint(a.config.HostKeyFingerprint)
	if expected == "" {
		return ssh.InsecureIgnoreHostKey()
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		actual := normalizeFingerprint(ssh.FingerprintSHA256(key))
		if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
			return fmt.Errorf("sftp host key fingerprint mismatch for %s (%s)", hostname, remote.String())
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
