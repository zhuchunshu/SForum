package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	devhygiene "github.com/zhuchunshu/sforum/apps/api/app/Support/DevHygiene"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// 开发期信任会话 CLI：与 tests/admin-extension-activation.sh 同一套凭据约定。
//   SFORUM_API_BASE          默认 http://127.0.0.1:8080/api/v1
//   SFORUM_SESSION_COOKIE    super_admin 的 sforum_session 值
//   SFORUM_ADMIN_EMAIL/PASSWORD  未提供 cookie 时自动登录
const devTrustDefaultAPIBase = "http://127.0.0.1:8080/api/v1"

type devTrustClient struct {
	base      *url.URL
	http      *http.Client
	cookie    string
	csrfToken string
}

func newDevTrustClient() (*devTrustClient, error) {
	raw := strings.TrimSpace(os.Getenv("SFORUM_API_BASE"))
	if raw == "" {
		raw = devTrustDefaultAPIBase
	}
	base, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid SFORUM_API_BASE: %w", err)
	}
	client := &devTrustClient{
		base:      base,
		http:      &http.Client{Timeout: 30 * time.Second},
		cookie:    strings.TrimSpace(os.Getenv("SFORUM_SESSION_COOKIE")),
		csrfToken: strings.TrimSpace(os.Getenv("SFORUM_CSRF_TOKEN")),
	}
	if client.cookie == "" {
		email := strings.TrimSpace(os.Getenv("SFORUM_ADMIN_EMAIL"))
		password := os.Getenv("SFORUM_ADMIN_PASSWORD")
		if email == "" || password == "" {
			return nil, fmt.Errorf("dev trust requires SFORUM_SESSION_COOKIE or SFORUM_ADMIN_EMAIL+SFORUM_ADMIN_PASSWORD")
		}
		if err := client.login(email, password); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func (c *devTrustClient) login(email, password string) error {
	body, _ := json.Marshal(map[string]string{"login": email, "password": password})
	resp, err := c.request(http.MethodPost, "/auth/login", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", resp.Status)
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sforum_session" {
			c.cookie = cookie.Value
		}
		if strings.HasPrefix(cookie.Name, "csrf_") {
			c.csrfToken = cookie.Value
		}
	}
	if c.cookie == "" {
		return fmt.Errorf("login succeeded but no sforum_session cookie returned")
	}
	return nil
}

func (c *devTrustClient) request(method, path string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.base.String()+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Origin", c.base.Scheme+"://"+c.base.Host)
	if c.cookie != "" {
		req.AddCookie(&http.Cookie{Name: "sforum_session", Value: c.cookie})
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	// 会话续期可能换发 cookie；保持最新值。
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sforum_session" {
			c.cookie = cookie.Value
		}
		if strings.HasPrefix(cookie.Name, "csrf_") {
			c.csrfToken = cookie.Value
		}
	}
	return resp, nil
}

func (c *devTrustClient) doJSON(method, path string, body []byte, out any) error {
	resp, err := c.request(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s -> %s: %s", method, path, resp.Status, strings.TrimSpace(string(payload)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(payload, out)
}

// devTrustPackageRef 从包清单读取扩展 ID 与根路径（会话键）。
type devTrustPackageRef struct {
	ExtensionID string
	RootPath    string
	PackageSlug string
}

func readDevTrustPackageRef(root string) (devTrustPackageRef, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return devTrustPackageRef{}, err
	}
	manifestPath := filepath.Join(abs, extensionmanifest.ManifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return devTrustPackageRef{}, fmt.Errorf("read manifest: %w", err)
	}
	var manifest struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return devTrustPackageRef{}, fmt.Errorf("parse manifest: %w", err)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return devTrustPackageRef{}, fmt.Errorf("manifest id is empty")
	}
	return devTrustPackageRef{
		ExtensionID: manifest.ID,
		RootPath:    abs,
		PackageSlug: manifest.ID + "@" + abs,
	}, nil
}

// devTrustRefresher 供 `sforum extension dev --auto-trust` 在每次构建成功后调用：
// 读取包清单得到 extensionId，POST /admin/extensions/:id/dev-trust-refresh。
type devTrustRefresher struct {
	client      *devTrustClient
	extensionID string
}

func newDevTrustRefresher(root string) (*devTrustRefresher, error) {
	ref, err := readDevTrustPackageRef(root)
	if err != nil {
		return nil, err
	}
	client, err := newDevTrustClient()
	if err != nil {
		return nil, err
	}
	return &devTrustRefresher{client: client, extensionID: ref.ExtensionID}, nil
}

// Refresh 幂等：digest 已授权时服务端直接成功。
func (r *devTrustRefresher) Refresh(ctx context.Context, packageDigest string) error {
	body, _ := json.Marshal(map[string]string{"packageDigest": packageDigest})
	return r.client.doJSON(http.MethodPost,
		"/admin/extensions/"+url.PathEscape(r.extensionID)+"/dev-trust-refresh", body, nil)
}

func newExtensionDevTrustCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev-trust",
		Short: "Development trust session helpers (requires a running dev API)",
	}
	cmd.AddCommand(newExtensionDevTrustCreateCommand())
	cmd.AddCommand(newExtensionDevTrustListCommand())
	cmd.AddCommand(newExtensionDevTrustRevokeCommand())
	return cmd
}

func newExtensionDevTrustCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [path]",
		Short: "Create a dev trust session for one extensions/dev package",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			if !devhygiene.ShouldEnableDevelopmentOrphanReaper(os.Getenv("APP_ENV")) {
				return fmt.Errorf("%w: APP_ENV must be development", errExtensionDevDisabled)
			}
			abs, err := resolveDevPackageRoot(root)
			if err != nil {
				return err
			}
			ref, err := readDevTrustPackageRef(abs)
			if err != nil {
				return err
			}
			client, err := newDevTrustClient()
			if err != nil {
				return err
			}
			body, _ := json.Marshal(map[string]string{
				"extensionId": ref.ExtensionID,
				"packageSlug": ref.PackageSlug,
				"rootPath":    ref.RootPath,
				"confirm":     "true",
			})
			var created map[string]any
			if err := client.doJSON(http.MethodPost, "/admin/extensions/dev-trust-sessions", body, &created); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "dev trust session created for %s\n", ref.ExtensionID)
			return nil
		},
	}
	return cmd
}

func newExtensionDevTrustListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dev trust sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDevTrustClient()
			if err != nil {
				return err
			}
			var sessions []map[string]any
			if err := client.doJSON(http.MethodGet, "/admin/extensions/dev-trust-sessions?includeRevoked=true", nil, &sessions); err != nil {
				return err
			}
			encoded, _ := json.MarshalIndent(sessions, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
			return nil
		},
	}
	return cmd
}

func newExtensionDevTrustRevokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <sessionID>",
		Short: "Revoke a dev trust session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newDevTrustClient()
			if err != nil {
				return err
			}
			if err := client.doJSON(http.MethodDelete, "/admin/extensions/dev-trust-sessions/"+url.PathEscape(args[0]), nil, nil); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "revoked %s\n", args[0])
			return nil
		},
	}
	return cmd
}
