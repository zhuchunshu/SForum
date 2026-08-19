package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/storageprovider"
)

const operationTimeout = 60 * time.Second

type s3Backend struct {
	client        *s3.Client
	presign       *s3.PresignClient
	bucket        string
	prefix        string
	publicBaseURL string
}

func newS3Backend(values map[string]string) (pluginsdk.StorageBackend, error) {
	region := strings.TrimSpace(values["region"])
	if region == "" {
		region = "us-east-1"
	}
	bucket := strings.TrimSpace(values["bucket"])
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(values["endpoint"]), "/")
	if endpoint != "" {
		parsed, err := url.Parse(endpoint)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, fmt.Errorf("endpoint must be an absolute http or https URL")
		}
	}
	accessKey, secretKey := strings.TrimSpace(values["access_key_id"]), strings.TrimSpace(values["secret_access_key"])
	if (accessKey == "") != (secretKey == "") {
		return nil, fmt.Errorf("access key id and secret access key must be configured together")
	}
	loadOptions := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if accessKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, strings.TrimSpace(values["session_token"]))))
	}
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.UsePathStyle = boolSetting(values["use_path_style"])
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})
	prefix := strings.Trim(strings.ReplaceAll(strings.TrimSpace(values["object_prefix"]), "\\", "/"), "/")
	if prefix == "." || strings.Contains(prefix, "..") {
		return nil, fmt.Errorf("object prefix is invalid")
	}
	return &s3Backend{client: client, presign: s3.NewPresignClient(client), bucket: bucket, prefix: prefix, publicBaseURL: strings.TrimRight(strings.TrimSpace(values["public_base_url"]), "/")}, nil
}

func (b *s3Backend) Probe() error {
	key := b.key(".sforum-probe/" + strconv.FormatInt(time.Now().UnixNano(), 36) + ".txt")
	payload := []byte("sforum-storage-probe")
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	if _, err := b.client.PutObject(ctx, &s3.PutObjectInput{Bucket: &b.bucket, Key: &key, Body: bytes.NewReader(payload), ContentType: aws.String("text/plain")}); err != nil {
		return fmt.Errorf("write probe object: %w", err)
	}
	defer func() {
		_, _ = b.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{Bucket: &b.bucket, Key: &key})
	}()
	if _, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.bucket, Key: &key}); err != nil {
		return fmt.Errorf("stat probe object: %w", err)
	}
	object, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &b.bucket, Key: &key})
	if err != nil {
		return fmt.Errorf("read probe object: %w", err)
	}
	got, readErr := io.ReadAll(object.Body)
	closeErr := object.Body.Close()
	if readErr != nil {
		return fmt.Errorf("read probe body: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close probe body: %w", closeErr)
	}
	if !bytes.Equal(got, payload) {
		return fmt.Errorf("probe object content mismatch")
	}
	if _, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &b.bucket, Key: &key}); err != nil {
		return fmt.Errorf("delete probe object: %w", err)
	}
	return nil
}

func (b *s3Backend) Put(key, contentType string, reader io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	objectKey := b.key(key)
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{Bucket: &b.bucket, Key: &objectKey, Body: reader, ContentType: optionalString(contentType)})
	return err
}

func (b *s3Backend) Open(key string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	objectKey := b.key(key)
	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &b.bucket, Key: &objectKey})
	if err != nil {
		cancel()
		return nil, err
	}
	return &cancelReadCloser{ReadCloser: result.Body, cancel: cancel}, nil
}

type cancelReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}

func (b *s3Backend) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	objectKey := b.key(key)
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &b.bucket, Key: &objectKey})
	return err
}

func (b *s3Backend) Stat(key string) (pluginsdk.StorageObjectInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	objectKey := b.key(key)
	result, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &b.bucket, Key: &objectKey})
	if isNotFound(err) {
		return pluginsdk.StorageObjectInfo{Exists: false}, nil
	}
	if err != nil {
		return pluginsdk.StorageObjectInfo{}, err
	}
	info := pluginsdk.StorageObjectInfo{Exists: true, ContentType: aws.ToString(result.ContentType)}
	if result.ContentLength != nil {
		info.Size = *result.ContentLength
	}
	if result.LastModified != nil {
		info.ModifiedUnix = result.LastModified.Unix()
	}
	return info, nil
}

func (b *s3Backend) Exists(key string) (bool, error) {
	info, err := b.Stat(key)
	return info.Exists, err
}

func (b *s3Backend) PublicURL(key string) string {
	if b.publicBaseURL == "" {
		return ""
	}
	return pluginsdk.JoinStoragePublicURL(b.publicBaseURL, b.key(key))
}

func (b *s3Backend) SignedURL(key string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	objectKey := b.key(key)
	result, err := b.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &b.bucket, Key: &objectKey}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func (b *s3Backend) key(key string) string {
	if b.prefix == "" {
		return key
	}
	return path.Join(b.prefix, key)
}

func boolSetting(value string) bool { return strings.EqualFold(strings.TrimSpace(value), "true") }

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NoSuchBucket"
	}
	return false
}
