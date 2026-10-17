package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --ensure 幂等：已有活跃会话（409 extension.dev_trust_session_exists）在
// --ensure 下视为成功；无 --ensure 保持报错。凭据经 SFORUM_SESSION_COOKIE 注入，
// 由 httptest 假 API 承接（dev-trust create 的语义由服务端裁决，fake 只回固定码）。
func TestExtensionDevTrustCreateEnsure(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(repoRoot, "extensions", "fixtures", "plugins", "sforum-page-loader-reference")
	if _, err := os.Stat(filepath.Join(root, "sforum.extension.json")); err != nil {
		t.Fatalf("reference fixture missing: %v", err)
	}

	cases := []struct {
		name        string
		ensure      bool
		status      int
		body        string
		wantErr     string
		wantMessage string
	}{
		{
			name:        "ensure tolerates session exists",
			ensure:      true,
			status:      http.StatusConflict,
			body:        `{"code":"extension.dev_trust_session_exists"}`,
			wantMessage: "active dev trust session already exists for sforum.page-loader-reference",
		},
		{
			name:    "without ensure session exists fails",
			ensure:  false,
			status:  http.StatusConflict,
			body:    `{"code":"extension.dev_trust_session_exists"}`,
			wantErr: "extension.dev_trust_session_exists",
		},
		{
			name:        "ensure tolerates fresh creation",
			ensure:      true,
			status:      http.StatusOK,
			body:        `{"id":"s1"}`,
			wantMessage: "dev trust session created for sforum.page-loader-reference",
		},
		{
			name:    "ensure does not swallow other failures",
			ensure:  true,
			status:  http.StatusForbidden,
			body:    `{"code":"extension.dev_trust_disabled"}`,
			wantErr: "extension.dev_trust_disabled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/dev-trust-sessions") {
					http.Error(w, "unexpected call", http.StatusNotFound)
					return
				}
				body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if strings.Contains(string(body), `"confirm":"true"`) {
					seen = "true"
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			t.Setenv("APP_ENV", "development")
			t.Setenv("SFORUM_API_BASE", server.URL)
			t.Setenv("SFORUM_SESSION_COOKIE", "test-cookie")
			out := &bytes.Buffer{}
			command := newExtensionDevTrustCreateCommand()
			command.SetOut(out)
			command.SetErr(out)
			args := []string{root}
			if tc.ensure {
				args = append(args, "--ensure")
			}
			command.SetArgs(args)
			execErr := command.Execute()

			if seen != "true" {
				t.Fatalf("create must still carry confirm:true (the explicit human confirmation), got %q", seen)
			}
			if tc.wantErr != "" {
				if execErr == nil || !strings.Contains(execErr.Error(), tc.wantErr) {
					t.Fatalf("err = %v; want containing %q", execErr, tc.wantErr)
				}
				return
			}
			if execErr != nil {
				t.Fatalf("unexpected error: %v", execErr)
			}
			if !strings.Contains(out.String(), tc.wantMessage) {
				t.Fatalf("output = %q; want %q", out.String(), tc.wantMessage)
			}
		})
	}
}
