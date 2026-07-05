package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

type FTPAdapter struct {
	config FTPConfig
}

func NewFTPAdapter(config FTPConfig) (*FTPAdapter, error) {
	if strings.TrimSpace(config.Host) == "" || strings.TrimSpace(config.Username) == "" {
		return nil, ErrInvalidConfig
	}
	if config.Port <= 0 {
		config.Port = 21
	}
	return &FTPAdapter{config: config}, nil
}

func (a *FTPAdapter) Put(ctx context.Context, key string, input PutInput) error {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return err
	}
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()
	if err := makeFTPDirs(conn, path.Dir(remotePath)); err != nil {
		return err
	}
	return conn.Stor(remotePath, input.Reader)
}

func (a *FTPAdapter) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return nil, err
	}
	conn, err := a.connect(ctx)
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

func (a *FTPAdapter) Delete(ctx context.Context, key string) error {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return err
	}
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.Delete(remotePath)
}

func (a *FTPAdapter) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	remotePath, err := joinRemotePath(a.config.RootPath, key)
	if err != nil {
		return ObjectInfo{}, err
	}
	conn, err := a.connect(ctx)
	if err != nil {
		return ObjectInfo{}, err
	}
	defer conn.Quit()
	size, err := conn.FileSize(remotePath)
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, Size: size}, nil
}

func (a *FTPAdapter) Exists(ctx context.Context, key string) (bool, error) {
	_, err := a.Stat(ctx, key)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (a *FTPAdapter) PublicURL(key string) string {
	return joinPublicURL(a.config.PublicBaseURL, key)
}

func (a *FTPAdapter) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return a.PublicURL(key), nil
}

func (a *FTPAdapter) Probe(ctx context.Context) error {
	conn, err := a.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Quit()
	return conn.NoOp()
}

func (a *FTPAdapter) connect(ctx context.Context) (*ftp.ServerConn, error) {
	options := []ftp.DialOption{
		ftp.DialWithContext(ctx),
		ftp.DialWithTimeout(10 * time.Second),
		ftp.DialWithDisabledEPSV(!a.config.Passive),
	}
	if a.config.ExplicitTLS {
		options = append(options, ftp.DialWithExplicitTLS(&tls.Config{ServerName: a.config.Host, MinVersion: tls.VersionTLS12}))
	}
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", a.config.Host, a.config.Port), options...)
	if err != nil {
		return nil, err
	}
	if err := conn.Login(a.config.Username, a.config.Password); err != nil {
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

func makeFTPDirs(conn *ftp.ServerConn, dir string) error {
	dir = strings.Trim(dir, "/")
	if dir == "" || dir == "." {
		return nil
	}
	current := ""
	for _, part := range strings.Split(dir, "/") {
		current = path.Join(current, part)
		if err := conn.MakeDir(current); err != nil {
			continue
		}
	}
	return nil
}
