package main

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func r2Settings() map[string]string {
	return map[string]string{
		"endpoint":          "https://abc123def456.r2.cloudflarestorage.com",
		"bucket":            "forum-attachments",
		"access_key_id":     "access",
		"secret_access_key": "secret",
	}
}

func TestNewS3BackendResolvesR2CompatibilityDefaults(t *testing.T) {
	backend, err := newS3Backend(r2Settings())
	if err != nil {
		t.Fatalf("newS3Backend: %v", err)
	}
	options := backend.(*s3Backend).client.Options()
	if !options.UsePathStyle {
		t.Fatalf("R2 endpoints must resolve to path-style addressing")
	}
	if options.Region != r2Region {
		t.Fatalf("R2 endpoints must default region to auto, got %q", options.Region)
	}
	if options.RequestChecksumCalculation != aws.RequestChecksumCalculationWhenRequired {
		t.Fatalf("request checksums must be when_required for third-party compatibility, got %v", options.RequestChecksumCalculation)
	}
	if options.ResponseChecksumValidation != aws.ResponseChecksumValidationWhenRequired {
		t.Fatalf("response checksum validation must be when_required, got %v", options.ResponseChecksumValidation)
	}
	if options.BaseEndpoint == nil || *options.BaseEndpoint != r2Settings()["endpoint"] {
		t.Fatalf("custom endpoint must be preserved verbatim")
	}
}

func TestNewS3BackendExplicitSettingsWinOverAutoDetection(t *testing.T) {
	values := r2Settings()
	values["region"] = "apac"
	values["use_path_style"] = "false"
	options := mustBackend(t, values).client.Options()
	if options.UsePathStyle {
		t.Fatalf("explicit false must disable path-style even for R2 endpoints")
	}
	if options.Region != "apac" {
		t.Fatalf("explicit region must win over auto detection, got %q", options.Region)
	}

	values["use_path_style"] = "true"
	options = mustBackend(t, values).client.Options()
	if !options.UsePathStyle {
		t.Fatalf("explicit true must enable path-style even without a custom endpoint")
	}
}

func TestNewS3BackendKeepsVirtualHostForAWS(t *testing.T) {
	values := map[string]string{
		"bucket":            "forum-attachments",
		"access_key_id":     "access",
		"secret_access_key": "secret",
	}
	options := mustBackend(t, values).client.Options()
	if options.UsePathStyle {
		t.Fatalf("no endpoint must keep virtual-host addressing for AWS")
	}
	if options.Region != defaultRegion {
		t.Fatalf("no endpoint and no region must fall back to %q, got %q", defaultRegion, options.Region)
	}

	values["endpoint"] = "https://s3.eu-central-1.amazonaws.com"
	if mustBackend(t, values).client.Options().UsePathStyle {
		t.Fatalf("amazonaws.com endpoints must keep virtual-host addressing")
	}
}

func TestNewS3BackendLegacyBooleanValuesStillParse(t *testing.T) {
	values := map[string]string{
		"endpoint":          "https://minio.example.com",
		"bucket":            "forum-attachments",
		"access_key_id":     "access",
		"secret_access_key": "secret",
		"use_path_style":    "true",
	}
	if !mustBackend(t, values).client.Options().UsePathStyle {
		t.Fatalf("legacy \"true\" value must enable path-style")
	}

	values["use_path_style"] = "false"
	if mustBackend(t, values).client.Options().UsePathStyle {
		t.Fatalf("legacy \"false\" value must disable path-style")
	}

	// 自定义端点留空（自动）时，MinIO 这类第三方服务必须落在 path-style。
	delete(values, "use_path_style")
	if !mustBackend(t, values).client.Options().UsePathStyle {
		t.Fatalf("custom non-AWS endpoints must default to path-style")
	}
}

func TestNewS3BackendPrefixAndPublicURL(t *testing.T) {
	values := map[string]string{
		"endpoint":          "https://abc123def456.r2.cloudflarestorage.com",
		"bucket":            "forum-attachments",
		"access_key_id":     "access",
		"secret_access_key": "secret",
		"object_prefix":     "attachments/2026",
		"public_base_url":   "https://cdn.example.com/assets/",
	}
	backend := mustBackend(t, values)
	if got := backend.key("a/b.png"); got != "attachments/2026/a/b.png" {
		t.Fatalf("prefix join: %q", got)
	}
	if got := backend.PublicURL("a/b.png"); got != "https://cdn.example.com/assets/attachments/2026/a/b.png" {
		t.Fatalf("public URL join: %q", got)
	}
	if got := mustBackend(t, r2Settings()).PublicURL("a/b.png"); got != "" {
		t.Fatalf("empty public base URL must keep host proxy, got %q", got)
	}
}

func TestNewS3BackendValidatesRequiredSettings(t *testing.T) {
	if _, err := newS3Backend(map[string]string{}); err == nil || !strings.Contains(err.Error(), "bucket is required") {
		t.Fatalf("missing bucket: %v", err)
	}
	values := r2Settings()
	delete(values, "secret_access_key")
	if _, err := newS3Backend(values); err == nil || !strings.Contains(err.Error(), "must be configured together") {
		t.Fatalf("half credentials: %v", err)
	}
	values["secret_access_key"] = "secret"
	values["endpoint"] = "not-a-url"
	if _, err := newS3Backend(values); err == nil || !strings.Contains(err.Error(), "endpoint must be an absolute") {
		t.Fatalf("invalid endpoint: %v", err)
	}
}

func mustBackend(t *testing.T, values map[string]string) *s3Backend {
	t.Helper()
	backend, err := newS3Backend(values)
	if err != nil {
		t.Fatalf("newS3Backend: %v", err)
	}
	return backend.(*s3Backend)
}
