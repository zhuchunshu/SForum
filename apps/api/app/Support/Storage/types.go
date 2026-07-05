package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"
)

const (
	ProviderLocal      = "local"
	ProviderAliyunOSS  = "aliyun_oss"
	ProviderTencentCOS = "tencent_cos"
	ProviderFTP        = "ftp"
	ProviderSFTP       = "sftp"
)

var (
	ErrInvalidConfig = errors.New("storage: invalid config")
	ErrInvalidKey    = errors.New("storage: invalid object key")
)

type PutInput struct {
	Reader      io.Reader
	Size        int64
	ContentType string
}

type ObjectInfo struct {
	Key         string
	Size        int64
	ContentType string
	ModifiedAt  time.Time
}

type Adapter interface {
	Put(ctx context.Context, key string, input PutInput) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Exists(ctx context.Context, key string) (bool, error)
	PublicURL(key string) string
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Probe(ctx context.Context) error
}

type Config struct {
	Provider      string
	LocalRoot     string
	PublicBaseURL string

	Local LocalConfig
	OSS   AliyunOSSConfig
	COS   TencentCOSConfig
	FTP   FTPConfig
	SFTP  SFTPConfig
}

type LocalConfig struct {
	PublicPrefix string
}

type AliyunOSSConfig struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	AccessKeySecret string
}

type TencentCOSConfig struct {
	Region    string
	Bucket    string
	SecretID  string
	SecretKey string
	CDNDomain string
}

type FTPConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	RootPath      string
	Passive       bool
	ExplicitTLS   bool
	PublicBaseURL string
}

type SFTPConfig struct {
	Host               string
	Port               int
	Username           string
	Password           string
	PrivateKey         string
	Passphrase         string
	RootPath           string
	HostKeyFingerprint string
	PublicBaseURL      string
}

func NewAdapter(config Config) (Adapter, error) {
	switch strings.TrimSpace(config.Provider) {
	case "", ProviderLocal:
		return NewLocalAdapter(config.LocalRoot, firstNonBlank(config.Local.PublicPrefix, config.PublicBaseURL))
	case ProviderAliyunOSS:
		return NewAliyunOSSAdapter(config.OSS, config.PublicBaseURL)
	case ProviderTencentCOS:
		return NewTencentCOSAdapter(config.COS, config.PublicBaseURL)
	case ProviderFTP:
		return NewFTPAdapter(config.FTP)
	case ProviderSFTP:
		return NewSFTPAdapter(config.SFTP)
	default:
		return nil, ErrInvalidConfig
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
