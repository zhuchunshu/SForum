package webreleaseruntime

import (
	"slices"
	"strings"
	"testing"
)

func TestInstallEnvironmentAllowsOnlyToolingAndNetworkConfiguration(t *testing.T) {
	source := []string{
		"PATH=/usr/bin", "HOME=/tmp/home", "TMPDIR=/tmp", "HTTP_PROXY=http://proxy",
		"BUN_CONFIG_REGISTRY=https://registry.example.com", "DATABASE_URL=postgres://secret",
		"SESSION_HASH_SECRET=secret", "NUXT_PUBLIC_API_BASE_URL=/api/v1",
	}
	environment := InstallEnvironment(source)
	joined := strings.Join(environment, "\n")
	for _, expected := range []string{"PATH=/usr/bin", "HOME=/tmp/home", "HTTP_PROXY=http://proxy", "BUN_CONFIG_REGISTRY=https://registry.example.com"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %s in install environment: %v", expected, environment)
		}
	}
	for _, secret := range []string{"DATABASE_URL", "SESSION_HASH_SECRET", "NUXT_PUBLIC_API_BASE_URL"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("install environment leaked %s: %v", secret, environment)
		}
	}
	if !slices.IsSorted(environment) {
		t.Fatalf("environment must be deterministic: %v", environment)
	}
}

func TestBuildEnvironmentAddsApprovedPublicAndReleaseValues(t *testing.T) {
	environment := BuildEnvironment(
		[]string{"PATH=/usr/bin", "NUXT_PUBLIC_API_BASE_URL=/api/v1", "APP_LOCALE=zh-CN", "REDIS_PASSWORD=secret"},
		map[string]string{"NUXT_BUILD_DIR": "/release/build", "SFORUM_ADMIN_REGISTRY_ROOT": "/release/registry"},
	)
	joined := strings.Join(environment, "\n")
	for _, expected := range []string{"NUXT_PUBLIC_API_BASE_URL=/api/v1", "APP_LOCALE=zh-CN", "NUXT_BUILD_DIR=/release/build"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %s in build environment: %v", expected, environment)
		}
	}
	if strings.Contains(joined, "REDIS_PASSWORD") {
		t.Fatalf("build environment leaked Redis credentials: %v", environment)
	}
}
