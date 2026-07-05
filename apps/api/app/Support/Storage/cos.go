package storage

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type TencentCOSAdapter struct {
	client        *cos.Client
	publicBaseURL string
	secretID      string
	secretKey     string
}

func NewTencentCOSAdapter(config TencentCOSConfig, publicBaseURL string) (*TencentCOSAdapter, error) {
	bucketURL := deriveTencentBucketURL(config.Bucket, config.Region)
	if bucketURL == "" || strings.TrimSpace(config.SecretID) == "" || strings.TrimSpace(config.SecretKey) == "" {
		return nil, ErrInvalidConfig
	}
	parsed, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: parsed}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  strings.TrimSpace(config.SecretID),
			SecretKey: strings.TrimSpace(config.SecretKey),
		},
	})
	return &TencentCOSAdapter{
		client:        client,
		publicBaseURL: firstNonBlank(config.CDNDomain, publicBaseURL, bucketURL),
		secretID:      strings.TrimSpace(config.SecretID),
		secretKey:     strings.TrimSpace(config.SecretKey),
	}, nil
}

func (a *TencentCOSAdapter) Put(ctx context.Context, key string, input PutInput) error {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return err
	}
	options := &cos.ObjectPutOptions{}
	if strings.TrimSpace(input.ContentType) != "" {
		options.ObjectPutHeaderOptions = &cos.ObjectPutHeaderOptions{ContentType: input.ContentType}
	}
	_, err = a.client.Object.Put(ctx, key, input.Reader, options)
	return err
}

func (a *TencentCOSAdapter) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Object.Get(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (a *TencentCOSAdapter) Delete(ctx context.Context, key string) error {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return err
	}
	_, err = a.client.Object.Delete(ctx, key)
	return err
}

func (a *TencentCOSAdapter) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	resp, err := a.client.Object.Head(ctx, key, nil)
	if err != nil {
		return ObjectInfo{}, err
	}
	size, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	modified, _ := http.ParseTime(resp.Header.Get("Last-Modified"))
	return ObjectInfo{Key: key, Size: size, ContentType: resp.Header.Get("Content-Type"), ModifiedAt: modified}, nil
}

func (a *TencentCOSAdapter) Exists(ctx context.Context, key string) (bool, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return false, err
	}
	return a.client.Object.IsExist(ctx, key)
}

func (a *TencentCOSAdapter) PublicURL(key string) string {
	return joinPublicURL(a.publicBaseURL, key)
}

func (a *TencentCOSAdapter) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	key, err := normalizeObjectKey(key)
	if err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	signed, err := a.client.Object.GetPresignedURL(ctx, http.MethodGet, key, a.secretID, a.secretKey, ttl, nil)
	if err != nil {
		return "", err
	}
	return signed.String(), nil
}

func (a *TencentCOSAdapter) Probe(ctx context.Context) error {
	_, _, err := a.client.Bucket.Get(ctx, &cos.BucketGetOptions{MaxKeys: 1})
	return err
}

func deriveTencentBucketURL(bucket string, region string) string {
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)
	if bucket == "" || region == "" {
		return ""
	}
	return "https://" + bucket + ".cos." + region + ".myqcloud.com"
}
