package main

import (
	"os"
	"strings"
	"time"
)

// 官方 GitHub.com 端点（V1 固定；Enterprise 可配置端点延后）。
const (
	defaultGitHubAuthURL  = "https://github.com/login/oauth/authorize"
	defaultGitHubTokenURL = "https://github.com/login/oauth/access_token"
	defaultGitHubAPIURL   = "https://api.github.com"

	// V1 最小身份证明 scope：read:user 读认证用户资料；user:email 取主验证邮箱作注册提示。
	// 访问令牌在身份证明后立即丢弃，不保留 GitHub API 访问。
	githubOAuthScope = "read:user user:email"

	defaultHTTPTimeout = 8 * time.Second
)

// GitHubEndpoints 可在测试中指向本地 fake GitHub；生产固定官方 URL。
type GitHubEndpoints struct {
	AuthURL  string
	TokenURL string
	APIURL   string
}

// OfficialGitHubEndpoints 返回 V1 官方 GitHub.com 端点。
func OfficialGitHubEndpoints() GitHubEndpoints {
	return GitHubEndpoints{
		AuthURL:  defaultGitHubAuthURL,
		TokenURL: defaultGitHubTokenURL,
		APIURL:   defaultGitHubAPIURL,
	}
}

// GitHubConfig 是插件进程内的 OAuth 配置。Client Secret 来自 Host 注入的
// SFORUM_SETTING_* 环境变量（SecretStore），永不进入浏览器或日志。
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	Endpoints    GitHubEndpoints
	HTTPTimeout  time.Duration
}

// LoadGitHubConfigFromEnv 从 Host 注入的设置与可选测试端点覆盖加载配置。
//
// V1 生产固定官方 GitHub.com 端点。fake-GitHub 覆盖仅在明确非 production
// 边界生效（APP_ENV != production）；production 一律忽略覆盖，纵深防御
// Host 透传过滤（T8C）。
func LoadGitHubConfigFromEnv() GitHubConfig {
	cfg := GitHubConfig{
		ClientID:     strings.TrimSpace(os.Getenv("SFORUM_SETTING_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("SFORUM_SETTING_CLIENT_SECRET")),
		Endpoints:    OfficialGitHubEndpoints(),
		HTTPTimeout:  defaultHTTPTimeout,
	}
	if !githubEndpointOverridesAllowed() {
		return cfg
	}
	// 测试/开发钩子：指向本地 fake GitHub（Host 仅在非 production 注入这些变量）。
	if v := strings.TrimSpace(os.Getenv("SFORUM_AUTH_GITHUB_AUTH_URL")); v != "" {
		cfg.Endpoints.AuthURL = v
	}
	if v := strings.TrimSpace(os.Getenv("SFORUM_AUTH_GITHUB_TOKEN_URL")); v != "" {
		cfg.Endpoints.TokenURL = v
	}
	if v := strings.TrimSpace(os.Getenv("SFORUM_AUTH_GITHUB_API_URL")); v != "" {
		cfg.Endpoints.APIURL = v
	}
	return cfg
}

// githubEndpointOverridesAllowed 判断是否允许 fake-GitHub 端点覆盖。
// production（含大小写变体）一律拒绝；其余环境允许（开发/测试/空 APP_ENV 本地）。
func githubEndpointOverridesAllowed() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

// CredentialsPresent 判断启动 OAuth 所需凭证是否齐全。
func (c GitHubConfig) CredentialsPresent() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.ClientSecret) != ""
}
