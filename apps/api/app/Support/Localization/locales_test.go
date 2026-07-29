package localization

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestParseSupportedLocalesDefaults(t *testing.T) {
	locales := ParseSupportedLocales("")
	if len(locales) != 2 {
		t.Fatalf("expected default locales, got %v", locales)
	}
	if locales[0] != "zh-CN" || locales[1] != "en-US" {
		t.Fatalf("unexpected default locale order: %v", locales)
	}
}

func TestNormalizeUsesAliasesAndFallback(t *testing.T) {
	supported := []string{"zh-CN", "en-US"}

	if got := Normalize("en", supported); got != "en-US" {
		t.Fatalf("expected en-US, got %s", got)
	}

	if got := Normalize("fr-FR", supported); got != "zh-CN" {
		t.Fatalf("expected zh-CN fallback, got %s", got)
	}
}

func TestMessageReturnsLocalizedAPIMessages(t *testing.T) {
	if got := Message("zh-CN", "auth.required"); got != "请先登录。" {
		t.Fatalf("expected Chinese auth message, got %q", got)
	}

	if got := Message("en-US", "auth.required"); got != "Please sign in first." {
		t.Fatalf("expected English auth message, got %q", got)
	}

	if got := Message("zh-CN", "auth.password_min_length"); got != "密码长度低于当前站点要求。" {
		t.Fatalf("expected Chinese password message, got %q", got)
	}

	if got := Message("en-US", "auth.password_symbol"); got != "Include a symbol in your password." {
		t.Fatalf("expected English password symbol message, got %q", got)
	}

	if got := Message("en-US", "auth.register_invalid"); got != "Registration failed: fix the highlighted fields and submit again." {
		t.Fatalf("expected English registration message, got %q", got)
	}
}

func TestMessageFallsBackToDefaultLocaleAndKey(t *testing.T) {
	if got := Message("fr-FR", "permission.denied"); got != "没有权限执行此操作。" {
		t.Fatalf("expected default-locale fallback, got %q", got)
	}

	if got := Message("en-US", "unknown.reason"); got != "unknown.reason" {
		t.Fatalf("expected unknown key fallback, got %q", got)
	}
}

func TestMessageLocalizesSiteChromeAndRecentAdminCodes(t *testing.T) {
	cases := map[string][2]string{
		"site_chrome.invalid":                          {"站点导航/公告/友链配置不正确，请检查后重试。", "The site navigation, announcement, or friend-link data is invalid. Check it and try again."},
		"site_chrome.not_found":                        {"站点导航/公告/友链不存在，请刷新后重试。", "The site navigation, announcement, or friend link does not exist. Refresh and try again."},
		"jobs.schedule_disabled":                       {"该定时任务已禁用，请先启用后再触发。", "This scheduled job is disabled. Enable it before triggering."},
		"profile.invalid":                              {"用户资料不正确，请检查后重试。", "The user profile is invalid. Check it and try again."},
		"database.table_not_found":                     {"数据表不存在或不可访问。", "The data table does not exist or is not accessible."},
		"moderation.report_duplicate":                  {"你已经举报过该内容。", "You have already reported this content."},
		"mail.test_recipient_required":                 {"请填写测试邮件收件人。", "Enter a recipient for the test email."},
		"csrf.invalid":                                 {"请求校验失败，请刷新页面后重试。", "Request validation failed. Refresh the page and try again."},
		"route.probe_invalid":                          {"路由探测请求无效。", "The route probe request is invalid."},
		"storage.ok":                                   {"存储连接正常。", "Storage connection is healthy."},
		"extensions.admin_surface_invalid":             {"管理端界面扩展请求不正确，请检查契约版本和输入数据。", "The Admin Surface request is invalid. Check the contract version and input data."},
		"extensions.admin_surface_not_found":           {"管理端界面扩展不存在或当前不可用。", "The Admin Surface does not exist or is not currently available."},
		"extensions.admin_surface_not_invokable":       {"该管理端界面扩展未声明可调用的类型化处理器。", "This Admin Surface does not declare an invokable typed handler."},
		"extensions.admin_surface_stale":               {"管理端界面扩展契约或操作状态已变化，请刷新后重试。", "The Admin Surface contract or operation state has changed. Refresh and try again."},
		"extensions.admin_surface_unavailable":         {"管理端界面扩展运行时暂时不可用，请稍后重试。", "The Admin Surface runtime is temporarily unavailable. Please try again later."},
		"extensions.cache_inspector_invalid":           {"缓存检查请求不正确，limit 必须是 1 到 200 之间的整数。", "The cache inspection request is invalid. Limit must be an integer from 1 to 200."},
		"extensions.cache_inspector_conflict":          {"缓存注册表在检查期间发生变化，请刷新后重试。", "The cache registry changed during inspection. Refresh and try again."},
		"extensions.cache_inspector_unavailable":       {"缓存检查服务暂时不可用，请稍后重试。", "Cache inspection is temporarily unavailable. Please try again later."},
		"extensions.composition_inspector_invalid":     {"组件组合检查请求不正确，limit 必须是 1 到 200 之间的整数。", "The component composition inspection request is invalid. Limit must be an integer from 1 to 200."},
		"extensions.composition_inspector_unavailable": {"组件组合检查服务暂时不可用，请稍后重试。", "Component composition inspection is temporarily unavailable. Please try again later."},
		"extensions.navigation_inspector_invalid":      {"导航检查请求不正确，limit 必须是 1 到 200 之间的整数。", "The navigation inspection request is invalid. Limit must be an integer from 1 to 200."},
		"extensions.navigation_inspector_unavailable":  {"导航检查服务暂时不可用，请稍后重试。", "Navigation inspection is temporarily unavailable. Please try again later."},
		"extensions.template_inspector_invalid":        {"模板检查请求不正确，limit 必须是 1 到 200 之间的整数。", "The template inspection request is invalid. Limit must be an integer from 1 to 200."},
		"extensions.template_inspector_unavailable":    {"模板检查服务暂时不可用，请稍后重试。", "Template inspection is temporarily unavailable. Please try again later."},
		"extensions.asset_inspector_invalid":           {"资源检查请求不正确，limit 必须是 1 到 200 之间的整数。", "The asset inspection request is invalid. Limit must be an integer from 1 to 200."},
		"extensions.asset_inspector_unavailable":       {"资源检查服务暂时不可用，请稍后重试。", "Asset inspection is temporarily unavailable. Please try again later."},
	}
	for key, want := range cases {
		if got := Message("zh-CN", key); got != want[0] {
			t.Fatalf("%s zh-CN: got %q want %q", key, got, want[0])
		}
		if got := Message("en-US", key); got != want[1] {
			t.Fatalf("%s en-US: got %q want %q", key, got, want[1])
		}
	}
}

// 防止新增 Code* 错误码后忘记写入 messages 目录，导致 API message 原样返回 key。
func TestAPIErrorCodesHaveLocalizedMessages(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../.."))

	codes := collectAPIErrorCodes(t, apiRoot)
	if len(codes) == 0 {
		t.Fatal("expected to discover API error codes")
	}

	var missing []string
	for _, code := range codes {
		zh := Message("zh-CN", code)
		en := Message("en-US", code)
		if zh == code || en == code {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("API error codes missing localization (message falls back to key): %s", strings.Join(missing, ", "))
	}
}

var apiErrorCodeLiteral = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9_]+)+$`)

func collectAPIErrorCodes(t *testing.T, apiRoot string) []string {
	t.Helper()
	seen := map[string]struct{}{}

	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// 跳过本包 messages 定义文件，只扫业务错误码来源。
		if strings.Contains(path, filepath.Join("Support", "Localization")) {
			return nil
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return parseErr
		}

		ast.Inspect(file, func(n ast.Node) bool {
			// const CodeFoo = "domain.key"
			vs, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "Code") || i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value := strings.Trim(lit.Value, `"`)
				if value == "ok" || !apiErrorCodeLiteral.MatchString(value) {
					continue
				}
				seen[value] = struct{}{}
			}
			return true
		})

		// 控制器里直接写的 fiber.NewError / NewError 字符串 reason。
		text := string(src)
		for _, re := range []*regexp.Regexp{
			regexp.MustCompile(`fiber\.NewError\([^,]+,\s*"([a-z][a-z0-9_.]+)"`),
			regexp.MustCompile(`(?:apphttp\.)?NewError(?:WithFields)?\([^,]+,\s*"([a-z][a-z0-9_.]+)"`),
			regexp.MustCompile(`(?:apphttp\.)?Abort\([^,]+,\s*"([a-z][a-z0-9_.]+)"`),
		} {
			for _, match := range re.FindAllStringSubmatch(text, -1) {
				value := match[1]
				if value == "ok" || !apiErrorCodeLiteral.MatchString(value) {
					continue
				}
				// 排除 jobs.schedule.<id>.enabled 这类 options key 拼接片段。
				if strings.HasPrefix(value, "jobs.schedule.") && !strings.Contains(value, "schedule_not_found") && !strings.Contains(value, "schedule_disabled") {
					continue
				}
				seen[value] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk api root: %v", err)
	}

	out := make([]string, 0, len(seen))
	for code := range seen {
		out = append(out, code)
	}
	return out
}

func TestNegotiateAcceptLanguage(t *testing.T) {
	supported := []string{"zh-CN", "en-US"}

	if got := NegotiateAcceptLanguage("en-US,en;q=0.9,zh-CN;q=0.8", supported, "zh-CN"); got != "en-US" {
		t.Fatalf("expected en-US, got %q", got)
	}

	if got := NegotiateAcceptLanguage("fr-FR,en;q=0.9", supported, "zh-CN"); got != "en-US" {
		t.Fatalf("expected en-US after unsupported locale, got %q", got)
	}

	if got := NegotiateAcceptLanguage("fr-FR,zh;q=0.9", supported, "zh-CN"); got != "zh-CN" {
		t.Fatalf("expected zh-CN fallback, got %q", got)
	}
}
