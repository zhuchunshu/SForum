package storage

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type AliyunOSSAdapter struct {
	client        *oss.Client
	bucket        *oss.Bucket
	bucketName    string
	endpoint      string
	publicBaseURL string
}

func NewAliyunOSSAdapter(config AliyunOSSConfig, publicBaseURL string) (*AliyunOSSAdapter, error) {
	endpoint := strings.TrimSpace(config.Endpoint)
	bucketName := strings.TrimSpace(config.Bucket)
	if endpoint == "" || bucketName == "" || strings.TrimSpace(config.AccessKeyID) == "" || strings.TrimSpace(config.AccessKeySecret) == "" {
		return nil, ErrInvalidConfig
	}
	client, err := oss.New(endpoint, strings.TrimSpace(config.AccessKeyID), strings.TrimSpace(config.AccessKeySecret))
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	return &AliyunOSSAdapter{
		client:        client,
		bucket:        bucket,
		bucketName:    bucketName,
		endpoint:      endpoint,
		publicBaseURL: firstNonBlank(publicBaseURL, deriveAliyunPublicURL(bucketName, endpoint)),
	}, nil
}

func (a *AliyunOSSAdapter) Put(_ context.Context, key string, input PutInput) error {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return err
	}
	options := []oss.Option{}
	if strings.TrimSpace(input.ContentType) != "" {
		options = append(options, oss.ContentType(input.ContentType))
	}
	return a.bucket.PutObject(key, input.Reader, options...)
}

func (a *AliyunOSSAdapter) Open(_ context.Context, key string) (io.ReadCloser, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return nil, err
	}
	return a.bucket.GetObject(key)
}

func (a *AliyunOSSAdapter) Delete(_ context.Context, key string) error {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return err
	}
	return a.bucket.DeleteObject(key)
}

func (a *AliyunOSSAdapter) Stat(_ context.Context, key string) (ObjectInfo, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	header, err := a.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	size, _ := strconv.ParseInt(header.Get("Content-Length"), 10, 64)
	modified, _ := http.ParseTime(header.Get("Last-Modified"))
	return ObjectInfo{
		Key:         key,
		Size:        size,
		ContentType: header.Get("Content-Type"),
		ModifiedAt:  modified,
	}, nil
}

func (a *AliyunOSSAdapter) Exists(_ context.Context, key string) (bool, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return false, err
	}
	return a.bucket.IsObjectExist(key)
}

func (a *AliyunOSSAdapter) PublicURL(key string) string {
	return joinPublicURL(a.publicBaseURL, key)
}

func (a *AliyunOSSAdapter) SignedURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return "", err
	}
	seconds := int64(ttl.Seconds())
	if seconds <= 0 {
		seconds = 300
	}
	return a.bucket.SignURL(key, oss.HTTPGet, seconds)
}

func (a *AliyunOSSAdapter) Probe(_ context.Context) error {
	_, err := a.client.GetBucketInfo(a.bucketName)
	return err
}

func deriveAliyunPublicURL(bucket string, endpoint string) string {
	endpoint = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(endpoint), "https://"), "http://")
	if endpoint == "" || bucket == "" {
		return ""
	}
	return "https://" + bucket + "." + endpoint
}
