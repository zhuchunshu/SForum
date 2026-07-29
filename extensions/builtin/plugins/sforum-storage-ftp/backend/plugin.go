package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

type ftpConfig struct {
	host          string
	port          int
	username      string
	password      string
	rootPath      string
	passive       bool
	explicitTLS   bool
	publicBaseURL string
}

type ftpBackend struct{ config ftpConfig }

func newFTPStorageProvider() *pluginsdk.StorageProvider {
	return pluginsdk.NewStorageProvider("storage.ftp", func() (pluginsdk.StorageBackend, error) {
		config, err := ftpConfigFromEnv()
		if err != nil {
			return nil, err
		}
		return &ftpBackend{config: config}, nil
	})
}

func ftpConfigFromEnv() (ftpConfig, error) {
	config := ftpConfig{
		host:          strings.TrimSpace(os.Getenv("SFORUM_SETTING_HOST")),
		username:      strings.TrimSpace(os.Getenv("SFORUM_SETTING_USERNAME")),
		password:      os.Getenv("SFORUM_SETTING_PASSWORD"),
		rootPath:      strings.TrimSpace(os.Getenv("SFORUM_SETTING_ROOT_PATH")),
		passive:       settingBool("SFORUM_SETTING_PASSIVE", true),
		explicitTLS:   settingBool("SFORUM_SETTING_EXPLICIT_TLS", false),
		publicBaseURL: strings.TrimSpace(os.Getenv("SFORUM_SETTING_PUBLIC_BASE_URL")),
	}
	if config.rootPath == "" {
		config.rootPath = "/"
	}
	config.port = settingPort("SFORUM_SETTING_PORT", 21)
	if config.host == "" || config.username == "" || strings.TrimSpace(config.password) == "" {
		return ftpConfig{}, fmt.Errorf("host, username, and password are required")
	}
	return config, nil
}

func (b *ftpBackend) Probe() error {
	conn, err := b.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.NoOp()
}

func (b *ftpBackend) Put(key, contentType string, reader io.Reader) error {
	_ = contentType
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return err
	}
	conn, err := b.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()
	if err := makeFTPDirs(conn, path.Dir(remotePath)); err != nil {
		return err
	}
	return conn.Stor(remotePath, reader)
}

func (b *ftpBackend) Open(key string) (io.ReadCloser, error) {
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return nil, err
	}
	conn, err := b.connect()
	if err != nil {
		return nil, err
	}
	response, err := conn.Retr(remotePath)
	if err != nil {
		_ = conn.Quit()
		return nil, err
	}
	return &ftpReadCloser{Response: response, conn: conn}, nil
}

func (b *ftpBackend) Delete(key string) error {
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return err
	}
	conn, err := b.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.Delete(remotePath)
}

func (b *ftpBackend) Stat(key string) (pluginsdk.StorageObjectInfo, error) {
	remotePath, err := pluginsdk.JoinStorageRemotePath(b.config.rootPath, key)
	if err != nil {
		return pluginsdk.StorageObjectInfo{}, err
	}
	conn, err := b.connect()
	if err != nil {
		return pluginsdk.StorageObjectInfo{}, err
	}
	defer conn.Quit()
	size, err := conn.FileSize(remotePath)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "550") {
			return pluginsdk.StorageObjectInfo{Exists: false}, nil
		}
		return pluginsdk.StorageObjectInfo{}, err
	}
	return pluginsdk.StorageObjectInfo{Exists: true, Size: size}, nil
}

func (b *ftpBackend) Exists(key string) (bool, error) {
	info, err := b.Stat(key)
	return info.Exists, err
}

func (b *ftpBackend) PublicURL(key string) string {
	return pluginsdk.JoinStoragePublicURL(b.config.publicBaseURL, key)
}

func (b *ftpBackend) SignedURL(key string, ttl time.Duration) (string, error) {
	_ = ttl
	return b.PublicURL(key), nil
}

func (b *ftpBackend) connect() (*ftp.ServerConn, error) {
	options := []ftp.DialOption{
		ftp.DialWithContext(context.Background()),
		ftp.DialWithTimeout(10 * time.Second),
		ftp.DialWithDisabledEPSV(!b.config.passive),
	}
	if b.config.explicitTLS {
		options = append(options, ftp.DialWithExplicitTLS(&tls.Config{ServerName: b.config.host, MinVersion: tls.VersionTLS12}))
	}
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", b.config.host, b.config.port), options...)
	if err != nil {
		return nil, err
	}
	if err := conn.Login(b.config.username, b.config.password); err != nil {
		_ = conn.Quit()
		return nil, err
	}
	return conn, nil
}

type ftpReadCloser struct {
	*ftp.Response
	conn *ftp.ServerConn
}

func (r *ftpReadCloser) Close() error {
	err := r.Response.Close()
	if quitErr := r.conn.Quit(); err == nil {
		err = quitErr
	}
	return err
}

func makeFTPDirs(conn *ftp.ServerConn, directory string) error {
	directory = strings.Trim(directory, "/")
	if directory == "" || directory == "." {
		return nil
	}
	current := ""
	for _, segment := range strings.Split(directory, "/") {
		current = path.Join(current, segment)
		if err := conn.MakeDir(current); err != nil {
			continue
		}
	}
	return nil
}

func settingBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func settingPort(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fallback
	}
	return port
}
