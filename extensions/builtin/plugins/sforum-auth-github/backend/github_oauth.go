package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// IdentityAssertion 是插件返回给 Host 的有界外部主体断言。
// 不含 digest、access token 或任何 Host 会话材料。
type IdentityAssertion struct {
	ProviderSubject string // GitHub 稳定数字 id（十进制字符串）
	DisplayName     string
	EmailHint       string
}

// ProbeResult 是有界配置/可达性探测结果。
// 在没有授权码的情况下无法证明 Client Secret 正确；不得声称「密钥已验证」。
type ProbeResult struct {
	OK      bool
	Reason  string
	Message string
}

// GitHubOAuth 封装 Authorization Code + PKCE S256 协议。
// 不拥有 state/callback 事务/会话；仅使用 Host 注入的材料与 GitHub 通信。
type GitHubOAuth struct {
	Config     GitHubConfig
	HTTPClient *http.Client
}

// NewGitHubOAuth 构造协议客户端。httpClient 可为 nil（使用带超时的默认客户端）。
func NewGitHubOAuth(cfg GitHubConfig, httpClient *http.Client) *GitHubOAuth {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.HTTPTimeout}
		if cfg.HTTPTimeout <= 0 {
			httpClient.Timeout = defaultHTTPTimeout
		}
	}
	return &GitHubOAuth{Config: cfg, HTTPClient: httpClient}
}

func (g *GitHubOAuth) oauth2Config(callbackURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     g.Config.ClientID,
		ClientSecret: g.Config.ClientSecret,
		RedirectURL:  strings.TrimSpace(callbackURL),
		Scopes:       strings.Fields(githubOAuthScope),
		Endpoint: oauth2.Endpoint{
			AuthURL:   g.Config.Endpoints.AuthURL,
			TokenURL:  g.Config.Endpoints.TokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

// BuildAuthorizeURL 使用 Host 拥有的 state / PKCE challenge / callbackURL 构造 GitHub 授权 URL。
func (g *GitHubOAuth) BuildAuthorizeURL(state, codeChallenge, callbackURL string) (string, error) {
	if g == nil {
		return "", publicErr("github.config_missing", "github oauth client is not configured")
	}
	if !g.Config.CredentialsPresent() {
		return "", publicErr("github.credentials_missing", "client id and client secret are required")
	}
	state = strings.TrimSpace(state)
	codeChallenge = strings.TrimSpace(codeChallenge)
	callbackURL = strings.TrimSpace(callbackURL)
	if state == "" || codeChallenge == "" || callbackURL == "" {
		return "", publicErr("github.start_input_invalid", "state, codeChallenge, and callbackUrl are required")
	}
	if err := requireAbsoluteHTTPSOrHTTP(callbackURL); err != nil {
		return "", err
	}

	cfg := g.oauth2Config(callbackURL)
	// GitHub 仅支持 S256；plain 不被接受。
	authURL := cfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	if authURL == "" {
		return "", publicErr("github.authorize_url_failed", "failed to build authorize URL")
	}
	return authURL, nil
}

// CompleteWithCode 用 Host 注入的 code + code_verifier + callbackURL 完成令牌交换并拉取用户/邮箱。
// 访问令牌仅在本函数栈内使用，返回前丢弃。
func (g *GitHubOAuth) CompleteWithCode(ctx context.Context, code, codeVerifier, callbackURL string) (IdentityAssertion, error) {
	if g == nil {
		return IdentityAssertion{}, publicErr("github.config_missing", "github oauth client is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !g.Config.CredentialsPresent() {
		return IdentityAssertion{}, publicErr("github.credentials_missing", "client id and client secret are required")
	}
	code = strings.TrimSpace(code)
	codeVerifier = strings.TrimSpace(codeVerifier)
	callbackURL = strings.TrimSpace(callbackURL)
	if code == "" || codeVerifier == "" || callbackURL == "" {
		return IdentityAssertion{}, publicErr("github.complete_input_invalid", "completionToken, codeVerifier, and callbackUrl are required")
	}
	if err := requireAbsoluteHTTPSOrHTTP(callbackURL); err != nil {
		return IdentityAssertion{}, err
	}

	cfg := g.oauth2Config(callbackURL)
	// 将可取消/超时的上下文与自定义 HTTP 客户端绑定到 oauth2 交换。
	httpClient := g.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	token, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return IdentityAssertion{}, mapTokenError(err)
	}
	accessToken := strings.TrimSpace(token.AccessToken)
	// 立即清空结构内令牌字段引用路径外的敏感值使用面；后续只用局部变量。
	token.AccessToken = ""
	token.RefreshToken = ""
	if accessToken == "" {
		return IdentityAssertion{}, publicErr("github.token_empty", "token response did not include access_token")
	}

	user, err := g.fetchUser(ctx, accessToken)
	if err != nil {
		return IdentityAssertion{}, err
	}
	emailHint, err := g.fetchPrimaryVerifiedEmail(ctx, accessToken)
	if err != nil {
		// 邮箱是注册提示，不是主体；失败时仍可用无 email 的断言（由 Host 决定是否要求邮箱）。
		// 但 malformed/auth 错误应 fail-closed。
		if isHardEmailError(err) {
			return IdentityAssertion{}, err
		}
		emailHint = ""
	}

	// 丢弃访问令牌：函数返回后局部变量离开作用域。
	accessToken = ""
	_ = accessToken

	display := strings.TrimSpace(user.Name)
	if display == "" {
		display = strings.TrimSpace(user.Login)
	}
	if len(display) > 200 {
		display = display[:200]
	}
	if len(emailHint) > 320 {
		emailHint = emailHint[:320]
	}

	return IdentityAssertion{
		ProviderSubject: strconv.FormatInt(user.ID, 10),
		DisplayName:     display,
		EmailHint:       strings.ToLower(strings.TrimSpace(emailHint)),
	}, nil
}

// Probe 有界探测：检查凭证是否配置，并对 API 根路径做可达性/形状检查。
// 不能在无授权码时证明 Client Secret；不得返回 ok 并声称密钥有效。
func (g *GitHubOAuth) Probe(ctx context.Context) ProbeResult {
	if g == nil {
		return ProbeResult{OK: false, Reason: "github.config_missing", Message: "github oauth client is not configured"}
	}
	if strings.TrimSpace(g.Config.ClientID) == "" {
		return ProbeResult{OK: false, Reason: "github.client_id_missing", Message: "Client ID is not configured"}
	}
	if strings.TrimSpace(g.Config.ClientSecret) == "" {
		return ProbeResult{OK: false, Reason: "github.client_secret_missing", Message: "Client Secret is not configured"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// 仅探测 API 根是否可达与 JSON 形状；不发送 Client Secret，不声称密钥已验证。
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(g.Config.Endpoints.APIURL, "/")+"/", nil)
	if err != nil {
		return ProbeResult{OK: false, Reason: "github.probe_request_invalid", Message: "failed to build probe request"}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sforum-auth-github")

	client := g.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResult{
			OK:      false,
			Reason:  "github.endpoint_unreachable",
			Message: RedactSensitive("GitHub API endpoint is unreachable"),
		}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		// 速率限制时仍算「端点可达」，但探测结果不应假装配置已完全验证。
		return ProbeResult{
			OK:      false,
			Reason:  "github.rate_limited",
			Message: "GitHub API rate limited the probe; credentials presence is configured but reachability is constrained",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 500 {
		return ProbeResult{
			OK:      false,
			Reason:  "github.endpoint_error",
			Message: fmt.Sprintf("GitHub API returned status %d", resp.StatusCode),
		}
	}
	// 形状：根响应应为 JSON 对象（官方 api.github.com 返回 current_user_url 等）。
	var shape map[string]any
	if err := json.Unmarshal(body, &shape); err != nil || len(shape) == 0 {
		return ProbeResult{
			OK:      false,
			Reason:  "github.response_malformed",
			Message: "GitHub API root response was not a JSON object",
		}
	}
	return ProbeResult{
		OK:     true,
		Reason: "github.probe_ok",
		Message: "Client ID/Secret are present and the API endpoint returned a JSON object. " +
			"Client Secret correctness cannot be proven without an authorization code.",
	}
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *GitHubOAuth) fetchUser(ctx context.Context, accessToken string) (githubUser, error) {
	var user githubUser
	status, body, err := g.apiGET(ctx, "/user", accessToken)
	if err != nil {
		return githubUser{}, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return githubUser{}, publicErr("github.user_unauthorized", "GitHub rejected the access token for /user")
	}
	if status == http.StatusTooManyRequests {
		return githubUser{}, publicErr("github.rate_limited", "GitHub rate limited /user")
	}
	if status < 200 || status >= 300 {
		return githubUser{}, publicErr("github.user_fetch_failed", fmt.Sprintf("GitHub /user returned status %d", status))
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return githubUser{}, publicErr("github.user_malformed", "GitHub /user response was not valid JSON")
	}
	if user.ID <= 0 {
		return githubUser{}, publicErr("github.subject_missing", "GitHub /user did not include a stable numeric id")
	}
	return user, nil
}

func (g *GitHubOAuth) fetchPrimaryVerifiedEmail(ctx context.Context, accessToken string) (string, error) {
	status, body, err := g.apiGET(ctx, "/user/emails", accessToken)
	if err != nil {
		return "", err
	}
	if status == http.StatusUnauthorized {
		return "", publicErr("github.email_unauthorized", "GitHub rejected the access token for /user/emails")
	}
	if status == http.StatusNotFound || status == http.StatusForbidden {
		// 无 user:email scope 或隐私限制：软失败，由调用方清空 emailHint。
		return "", &ErrPublic{Reason: "github.email_unavailable", Message: "email list unavailable"}
	}
	if status == http.StatusTooManyRequests {
		return "", publicErr("github.rate_limited", "GitHub rate limited /user/emails")
	}
	if status < 200 || status >= 300 {
		return "", publicErr("github.email_fetch_failed", fmt.Sprintf("GitHub /user/emails returned status %d", status))
	}
	var emails []githubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", publicErr("github.email_malformed", "GitHub /user/emails response was not valid JSON")
	}
	var fallback string
	for _, item := range emails {
		email := strings.ToLower(strings.TrimSpace(item.Email))
		if email == "" || !item.Verified {
			continue
		}
		if item.Primary {
			return email, nil
		}
		if fallback == "" {
			fallback = email
		}
	}
	return fallback, nil
}

func isHardEmailError(err error) bool {
	pub, ok := err.(*ErrPublic)
	if !ok || pub == nil {
		return true
	}
	switch pub.Reason {
	case "github.email_unavailable":
		return false
	default:
		return true
	}
}

func (g *GitHubOAuth) apiGET(ctx context.Context, path, accessToken string) (int, []byte, error) {
	base := strings.TrimRight(g.Config.Endpoints.APIURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return 0, nil, publicErr("github.api_request_invalid", "failed to build GitHub API request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "sforum-auth-github")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := g.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 0, nil, publicErr("github.timeout", "GitHub API request was cancelled or timed out")
		}
		return 0, nil, publicErr("github.api_unreachable", "GitHub API request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, publicErr("github.api_read_failed", "failed to read GitHub API response")
	}
	return resp.StatusCode, body, nil
}

// requireAbsoluteHTTPSOrHTTP 拒绝相对路径与无 scheme 的 callback；Host 应注入可信站点绝对地址。
func requireAbsoluteHTTPSOrHTTP(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return publicErr("github.callback_invalid", "callbackUrl must be an absolute URL")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https", "http":
		return nil
	default:
		return publicErr("github.callback_invalid", "callbackUrl must use http or https")
	}
}

func mapTokenError(err error) error {
	if err == nil {
		return nil
	}
	msg := RedactSensitive(err.Error())
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "access_denied") || strings.Contains(lower, "incorrect_client_credentials"):
		return publicErr("github.token_denied", "GitHub denied the token exchange")
	case strings.Contains(lower, "bad_verification_code") || strings.Contains(lower, "incorrect_client_credentials"):
		return publicErr("github.token_invalid", "GitHub rejected the authorization code")
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return publicErr("github.timeout", "GitHub token exchange timed out")
	default:
		return publicErr("github.token_exchange_failed", "GitHub token exchange failed")
	}
}

// 确保 time 被使用（HTTPTimeout 文档）。
var _ = time.Second
