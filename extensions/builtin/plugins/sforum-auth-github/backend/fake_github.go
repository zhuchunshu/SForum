package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"
)

// FakeGitHub 是协议级单测用的本地 GitHub OAuth + API 替身。
// 覆盖 authorize 参数记录、token 交换（含 PKCE S256 校验）、/user 与 /user/emails。
type FakeGitHub struct {
	Server *httptest.Server

	mu sync.Mutex

	// 可配置行为
	ClientID     string
	ClientSecret string
	User         FakeGitHubUser
	Emails       []FakeGitHubEmail

	// 故障注入
	DenyAuthorize     bool
	TokenError        string // 非空时 token 端点返回该 error 字段
	TokenHTTPStatus   int
	UserHTTPStatus    int
	UserBody          string // 非空时覆盖 /user 响应体
	EmailsHTTPStatus  int
	EmailsBody        string
	APIRootHTTPStatus int
	APIRootBody       string
	SlowToken         time.Duration
	RateLimitUser     bool

	// 观测
	LastAuthorizeQuery url.Values
	TokenExchanges     int
	UserFetches        int
	EmailFetches       int

	// 已颁发 code → expected verifier / used
	codes map[string]fakeCodeRecord
}

type fakeCodeRecord struct {
	Verifier string
	Used     bool
}

// FakeGitHubUser 模拟 api.github.com/user 主体。
type FakeGitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// FakeGitHubEmail 模拟 /user/emails 条目。
type FakeGitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// NewFakeGitHub 启动本地 httptest 服务器，默认返回一个可用用户与主验证邮箱。
func NewFakeGitHub() *FakeGitHub {
	f := &FakeGitHub{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		User: FakeGitHubUser{
			ID:        424242,
			Login:     "octocat",
			Name:      "The Octocat",
			AvatarURL: "https://avatars.example/octocat",
		},
		Emails: []FakeGitHubEmail{
			{Email: "octocat@example.com", Primary: true, Verified: true},
			{Email: "secondary@example.com", Primary: false, Verified: true},
		},
		codes: make(map[string]fakeCodeRecord),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/authorize", f.handleAuthorize)
	mux.HandleFunc("/login/oauth/access_token", f.handleToken)
	mux.HandleFunc("/user/emails", f.handleEmails)
	mux.HandleFunc("/user", f.handleUser)
	mux.HandleFunc("/", f.handleAPIRoot)
	f.Server = httptest.NewServer(mux)
	return f
}

// Close 关闭底层服务器。
func (f *FakeGitHub) Close() {
	if f != nil && f.Server != nil {
		f.Server.Close()
	}
}

// Endpoints 返回可注入 GitHubOAuth 的端点集合。
func (f *FakeGitHub) Endpoints() GitHubEndpoints {
	base := f.Server.URL
	return GitHubEndpoints{
		AuthURL:  base + "/login/oauth/authorize",
		TokenURL: base + "/login/oauth/access_token",
		APIURL:   base,
	}
}

// IssueCode 预置一个授权码及其期望的 PKCE verifier（S256 校验）。
func (f *FakeGitHub) IssueCode(code, verifier string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.codes[code] = fakeCodeRecord{Verifier: verifier}
}

func (f *FakeGitHub) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.LastAuthorizeQuery = r.URL.Query()
	deny := f.DenyAuthorize
	f.mu.Unlock()

	if deny {
		http.Error(w, "access_denied", http.StatusForbidden)
		return
	}
	// 真实 GitHub 会 302 到 callback；协议测试只需确认参数被接受。
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("authorize-ok"))
}

func (f *FakeGitHub) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	values, _ := url.ParseQuery(string(body))
	// 也接受 form-encoded Content-Type 由 r.ParseForm 解析。
	_ = r.ParseForm()
	for k, vs := range r.Form {
		if values.Get(k) == "" && len(vs) > 0 {
			values.Set(k, vs[0])
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.TokenExchanges++

	if f.SlowToken > 0 {
		delay := f.SlowToken
		f.mu.Unlock()
		time.Sleep(delay)
		f.mu.Lock()
	}

	if f.TokenError != "" {
		status := f.TokenHTTPStatus
		if status == 0 {
			status = http.StatusBadRequest
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             f.TokenError,
			"error_description": "injected token error",
		})
		return
	}

	if values.Get("client_id") != f.ClientID || values.Get("client_secret") != f.ClientSecret {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "incorrect_client_credentials",
			"error_description": "client credentials mismatch",
		})
		return
	}

	code := values.Get("code")
	verifier := values.Get("code_verifier")
	rec, ok := f.codes[code]
	if !ok || rec.Used {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "bad_verification_code",
			"error_description": "code is invalid or already used",
		})
		return
	}
	// PKCE S256：challenge = BASE64URL(SHA256(verifier))，此处直接比对预置 verifier。
	if rec.Verifier != "" && verifier != rec.Verifier {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "code_verifier mismatch",
		})
		return
	}
	rec.Used = true
	f.codes[code] = rec

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"access_token": "gho_test_access_token_not_for_production",
		"token_type":   "bearer",
		"scope":        "read:user,user:email",
	})
}

func (f *FakeGitHub) handleUser(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	f.UserFetches++
	status := f.UserHTTPStatus
	body := f.UserBody
	rateLimit := f.RateLimitUser
	user := f.User
	f.mu.Unlock()

	if rateLimit {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
		return
	}
	if status != 0 {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
		return
	}
	if body != "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (f *FakeGitHub) handleEmails(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	f.EmailFetches++
	status := f.EmailsHTTPStatus
	body := f.EmailsBody
	emails := append([]FakeGitHubEmail(nil), f.Emails...)
	f.mu.Unlock()

	if status != 0 {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
		return
	}
	if body != "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(emails)
}

func (f *FakeGitHub) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	f.mu.Lock()
	status := f.APIRootHTTPStatus
	body := f.APIRootBody
	f.mu.Unlock()
	if status != 0 {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
		return
	}
	if body != "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"current_user_url": f.Server.URL + "/user",
		"emails_url":       f.Server.URL + "/user/emails",
	})
}

func (f *FakeGitHub) authorized(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return strings.HasPrefix(strings.ToLower(auth), "bearer ") && len(auth) > 7
}

// S256Challenge 计算 PKCE S256 code_challenge，供测试与 Host 对齐。
func S256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
