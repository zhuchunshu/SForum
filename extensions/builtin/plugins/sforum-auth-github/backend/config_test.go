package main

import (
	"testing"
)

func TestT8C_LoadGitHubConfigIgnoresEndpointOverridesInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SFORUM_SETTING_CLIENT_ID", "cid")
	t.Setenv("SFORUM_SETTING_CLIENT_SECRET", "csecret")
	t.Setenv("SFORUM_AUTH_GITHUB_AUTH_URL", "http://evil.example/oauth/authorize")
	t.Setenv("SFORUM_AUTH_GITHUB_TOKEN_URL", "http://evil.example/oauth/token")
	t.Setenv("SFORUM_AUTH_GITHUB_API_URL", "http://evil.example/api")

	cfg := LoadGitHubConfigFromEnv()
	official := OfficialGitHubEndpoints()
	if cfg.Endpoints.AuthURL != official.AuthURL {
		t.Fatalf("production AuthURL = %q, want official %q", cfg.Endpoints.AuthURL, official.AuthURL)
	}
	if cfg.Endpoints.TokenURL != official.TokenURL {
		t.Fatalf("production TokenURL = %q, want official %q", cfg.Endpoints.TokenURL, official.TokenURL)
	}
	if cfg.Endpoints.APIURL != official.APIURL {
		t.Fatalf("production APIURL = %q, want official %q", cfg.Endpoints.APIURL, official.APIURL)
	}
	if cfg.ClientID != "cid" || cfg.ClientSecret != "csecret" {
		t.Fatalf("credentials should still load: id=%q secret=%q", cfg.ClientID, cfg.ClientSecret)
	}
}

func TestT8C_LoadGitHubConfigAllowsEndpointOverridesOutsideProduction(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SFORUM_AUTH_GITHUB_AUTH_URL", "http://127.0.0.1:9/oauth/authorize")
	t.Setenv("SFORUM_AUTH_GITHUB_TOKEN_URL", "http://127.0.0.1:9/oauth/token")
	t.Setenv("SFORUM_AUTH_GITHUB_API_URL", "http://127.0.0.1:9/api")

	cfg := LoadGitHubConfigFromEnv()
	if cfg.Endpoints.AuthURL != "http://127.0.0.1:9/oauth/authorize" {
		t.Fatalf("dev AuthURL = %q", cfg.Endpoints.AuthURL)
	}
	if cfg.Endpoints.TokenURL != "http://127.0.0.1:9/oauth/token" {
		t.Fatalf("dev TokenURL = %q", cfg.Endpoints.TokenURL)
	}
	if cfg.Endpoints.APIURL != "http://127.0.0.1:9/api" {
		t.Fatalf("dev APIURL = %q", cfg.Endpoints.APIURL)
	}
}

func TestT8C_GitHubEndpointOverridesAllowed(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if githubEndpointOverridesAllowed() {
		t.Fatal("production must disallow overrides")
	}
	t.Setenv("APP_ENV", "PRODUCTION")
	if githubEndpointOverridesAllowed() {
		t.Fatal("PRODUCTION must disallow overrides")
	}
	t.Setenv("APP_ENV", "development")
	if !githubEndpointOverridesAllowed() {
		t.Fatal("development must allow overrides")
	}
	t.Setenv("APP_ENV", "")
	if !githubEndpointOverridesAllowed() {
		t.Fatal("empty APP_ENV (local) must allow overrides for tests")
	}
}
