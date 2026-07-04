# API Response Envelope And Localized Message Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate SForum API responses to `{ code, message, data }`, localize backend API messages, and update Nuxt consumers to unwrap the envelope.

**Architecture:** Add backend locale/message helpers under `app/Support/Localization`, then centralize HTTP response helpers and error envelopes under `app/Http`. Controllers return through the helpers, OpenAPI documents the envelope, and Nuxt uses one small API client utility to send locale context, unwrap `data`, and prefer backend `message` for API feedback.

**Tech Stack:** Go 1.25, Fiber v3, Nuxt 4, Vue 3, TypeScript, OpenAPI 3.1, existing project localization config.

---

## File Structure

- Create `apps/api/app/Support/Localization/messages.go`: backend API message catalog, lookup, and `Accept-Language` negotiation.
- Modify `apps/api/app/Support/Localization/locales_test.go`: add tests for message lookup and header negotiation.
- Create `apps/api/app/Http/responses.go`: response envelope types and helpers (`OK`, `Created`, `NoData`, `ErrorResponse`).
- Create `apps/api/app/Http/locale.go`: request locale middleware and `Locale(c fiber.Ctx)`.
- Modify `apps/api/app/Http/errors.go`: emit localized envelope errors with `data.reason`.
- Modify `apps/api/app/Http/server.go`: install locale middleware and wrap `/health`.
- Modify `apps/api/app/Http/server_test.go`: update existing tests to decode envelopes and add locale-specific assertions.
- Modify `apps/api/app/Http/Controllers/Identity/controller.go`: return successful responses through `apphttp.OK`, `apphttp.Created`, and `apphttp.NoData`.
- Modify `contracts/openapi.yaml`: define envelope schemas and update existing endpoint responses.
- Create `apps/web/app/composables/useApiClient.ts`: Nuxt API envelope types, locale headers, `$fetch` wrapper, error helpers.
- Modify `apps/web/app/composables/useAuthSession.ts`: unwrap current-user envelope.
- Modify `apps/web/app/pages/login.vue`: use API client and backend error `message`.
- Modify `apps/web/app/pages/register.vue`: use API client, backend error `message`, and `data.reason` for human-verification branching.
- Modify `apps/web/app/pages/admin/roles.vue`: unwrap role-list envelope and send locale headers.
- Modify `apps/web/app/layouts/admin.vue`: use API client for logout.
- Update `knowledge/sessions/2026-07-04-api-response-envelope-localized-message.md`: implementation handoff after code is verified.

---

### Task 1: Backend Localization Message Boundary

**Files:**
- Create: `apps/api/app/Support/Localization/messages.go`
- Modify: `apps/api/app/Support/Localization/locales_test.go`

- [ ] **Step 1: Write failing localization tests**

Append these tests to `apps/api/app/Support/Localization/locales_test.go`:

```go
func TestMessageReturnsLocalizedAPIMessages(t *testing.T) {
	if got := Message("zh-CN", "auth.required"); got != "请先登录。" {
		t.Fatalf("expected Chinese auth message, got %q", got)
	}

	if got := Message("en-US", "auth.required"); got != "Please sign in first." {
		t.Fatalf("expected English auth message, got %q", got)
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
```

- [ ] **Step 2: Run localization tests and verify they fail**

Run:

```bash
cd apps/api
go test ./app/Support/Localization -run 'TestMessage|TestNegotiate' -count=1
```

Expected: FAIL with undefined `Message` and `NegotiateAcceptLanguage`.

- [ ] **Step 3: Add message catalog and negotiation helper**

Create `apps/api/app/Support/Localization/messages.go`:

```go
package localization

import (
	"sort"
	"strconv"
	"strings"
)

var messages = map[string]map[string]string{
	"zh-CN": {
		"ok":                             "OK",
		"auth.required":                  "请先登录。",
		"auth.invalid_credentials":       "账号或密码不正确。",
		"permission.denied":              "没有权限执行此操作。",
		"validation.invalid":             "请求参数不正确。",
		"human_verification.required":    "请先完成人机验证。",
		"human_verification.invalid":     "人机验证失败，请重新验证。",
		"human_verification.expired":     "人机验证已过期，请重新验证。",
		"human_verification.replayed":    "本次人机验证已使用，请重新验证。",
		"rate_limit.exceeded":            "操作过于频繁，请稍后再试。",
		"role.system_role_locked":        "系统角色不能执行此操作。",
		"role.default_role_locked":       "默认角色不能执行此操作。",
		"user.initial_super_admin_locked": "初始超级管理员不能执行此操作。",
		"auth.password_policy":           "密码不符合安全要求。",
		"not_found":                      "请求的资源不存在。",
		"method_not_allowed":             "不支持当前请求方法。",
		"internal_error":                 "服务器暂时不可用，请稍后再试。",
	},
	"en-US": {
		"ok":                             "OK",
		"auth.required":                  "Please sign in first.",
		"auth.invalid_credentials":       "The account or password is incorrect.",
		"permission.denied":              "You do not have permission to perform this action.",
		"validation.invalid":             "The request parameters are invalid.",
		"human_verification.required":    "Please complete human verification first.",
		"human_verification.invalid":     "Human verification failed. Please try again.",
		"human_verification.expired":     "Human verification expired. Please verify again.",
		"human_verification.replayed":    "This human verification has already been used. Please verify again.",
		"rate_limit.exceeded":            "Too many attempts. Please try again later.",
		"role.system_role_locked":        "System roles cannot perform this action.",
		"role.default_role_locked":       "The default role cannot perform this action.",
		"user.initial_super_admin_locked": "The initial super administrator cannot perform this action.",
		"auth.password_policy":           "The password does not meet the security policy.",
		"not_found":                      "The requested resource does not exist.",
		"method_not_allowed":             "This request method is not supported.",
		"internal_error":                 "The server is temporarily unavailable. Please try again later.",
	},
}

func Message(locale string, key string) string {
	normalized := Normalize(locale, []string{"zh-CN", "en-US"})
	if catalog, ok := messages[normalized]; ok {
		if message, ok := catalog[key]; ok {
			return message
		}
	}

	if catalog, ok := messages[DefaultLocale]; ok {
		if message, ok := catalog[key]; ok {
			return message
		}
	}

	return key
}

func NegotiateAcceptLanguage(header string, supported []string, fallback string) string {
	fallback = Normalize(fallback, supported)
	ranges := parseAcceptLanguage(header)
	if len(ranges) == 0 {
		return fallback
	}

	for _, item := range ranges {
		if locale, ok := matchSupportedLocale(item.tag, supported); ok {
			return locale
		}
	}

	return fallback
}

func matchSupportedLocale(locale string, supported []string) (string, bool) {
	candidate := strings.TrimSpace(locale)
	if candidate == "" {
		return "", false
	}

	if alias, ok := aliases[strings.ToLower(candidate)]; ok {
		candidate = alias
	}

	for _, item := range supported {
		if strings.EqualFold(candidate, item) {
			return item, true
		}
	}

	return "", false
}

type languageRange struct {
	tag   string
	q     float64
	order int
}

func parseAcceptLanguage(header string) []languageRange {
	parts := strings.Split(header, ",")
	ranges := make([]languageRange, 0, len(parts))

	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag := part
		q := 1.0
		if semi := strings.Index(part, ";"); semi >= 0 {
			tag = strings.TrimSpace(part[:semi])
			for _, param := range strings.Split(part[semi+1:], ";") {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "q=") {
					parsed, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64)
					if err == nil {
						q = parsed
					}
				}
			}
		}

		if tag != "" && tag != "*" && q > 0 {
			ranges = append(ranges, languageRange{tag: tag, q: q, order: index})
		}
	}

	sort.SliceStable(ranges, func(i, j int) bool {
		if ranges[i].q == ranges[j].q {
			return ranges[i].order < ranges[j].order
		}
		return ranges[i].q > ranges[j].q
	})

	return ranges
}
```

- [ ] **Step 4: Run localization tests and verify they pass**

Run:

```bash
cd apps/api
go test ./app/Support/Localization -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit localization boundary**

```bash
git add apps/api/app/Support/Localization/messages.go apps/api/app/Support/Localization/locales_test.go
git commit -m "feat(api): add localized API messages"
```

---

### Task 2: HTTP Envelope Helpers And Localized Error Handler

**Files:**
- Create: `apps/api/app/Http/responses.go`
- Create: `apps/api/app/Http/locale.go`
- Modify: `apps/api/app/Http/errors.go`
- Modify: `apps/api/app/Http/server.go`
- Modify: `apps/api/app/Http/server_test.go`

- [ ] **Step 1: Write failing HTTP envelope tests**

Add these helper types near the top of `apps/api/app/Http/server_test.go`:

```go
type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type apiErrorData struct {
	Reason string `json:"reason"`
}
```

Replace `TestHealthEndpoint` body decoding with this expected envelope:

```go
var body apiEnvelope[healthResponse]
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
	t.Fatalf("decode health response: %v", err)
}

if body.Code != nethttp.StatusOK {
	t.Fatalf("expected envelope code 200, got %d", body.Code)
}
if body.Message != "OK" {
	t.Fatalf("expected OK message, got %q", body.Message)
}
if body.Data.Status != "ok" {
	t.Fatalf("expected ok status, got %q", body.Data.Status)
}
if body.Data.Locale != "zh-CN" {
	t.Fatalf("expected zh-CN locale, got %q", body.Data.Locale)
}
if len(body.Data.SupportedLocales) != 2 {
	t.Fatalf("expected two supported locales, got %v", body.Data.SupportedLocales)
}
```

Add this test to `apps/api/app/Http/server_test.go`:

```go
func TestSessionEndpointReturnsLocalizedEnvelopeError(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{RouteProviders: []RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/session", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected envelope code 401, got %d", body.Code)
	}
	if body.Message != "Please sign in first." {
		t.Fatalf("expected localized message, got %q", body.Message)
	}
	if body.Data.Reason != "auth.required" {
		t.Fatalf("expected auth.required reason, got %q", body.Data.Reason)
	}
}
```

- [ ] **Step 2: Run HTTP tests and verify they fail**

Run:

```bash
cd apps/api
go test ./app/Http -run 'TestHealthEndpoint|TestSessionEndpointReturnsLocalizedEnvelopeError' -count=1
```

Expected: FAIL because current responses are not envelope-wrapped.

- [ ] **Step 3: Add locale middleware**

Create `apps/api/app/Http/locale.go`:

```go
package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

const requestLocaleKey = "sforum.locale"

func localeMiddleware(cfg config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		locale := localization.NegotiateAcceptLanguage(
			c.Get("Accept-Language"),
			cfg.SupportedLocales,
			cfg.AppLocale,
		)
		c.Locals(requestLocaleKey, locale)
		return c.Next()
	}
}

func Locale(c fiber.Ctx) string {
	if locale, ok := c.Locals(requestLocaleKey).(string); ok && locale != "" {
		return locale
	}
	return localization.DefaultLocale
}
```

- [ ] **Step 4: Add response helpers**

Create `apps/api/app/Http/responses.go`:

```go
package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Localization"
)

const MessageOK = "ok"

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ErrorData struct {
	Reason string              `json:"reason"`
	Fields map[string][]string `json:"fields,omitempty"`
}

func JSON(c fiber.Ctx, status int, messageKey string, data any) error {
	return c.Status(status).JSON(Response{
		Code:    status,
		Message: localization.Message(Locale(c), messageKey),
		Data:    data,
	})
}

func OK(c fiber.Ctx, data any) error {
	return JSON(c, fiber.StatusOK, MessageOK, data)
}

func Created(c fiber.Ctx, data any) error {
	return JSON(c, fiber.StatusCreated, MessageOK, data)
}

func NoData(c fiber.Ctx) error {
	return JSON(c, fiber.StatusOK, MessageOK, nil)
}

func ErrorResponse(c fiber.Ctx, status int, reason string) error {
	return JSON(c, status, reason, ErrorData{Reason: reason})
}
```

- [ ] **Step 5: Update error handler**

Replace `apps/api/app/Http/errors.go` with:

```go
package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		reason := "internal_error"

		if fiberErr, ok := err.(*fiber.Error); ok {
			status = fiberErr.Code
			reason = normalizeFiberErrorReason(status, fiberErr.Message)
		} else {
			logger.Error("request failed", "error", err)
		}

		return ErrorResponse(c, status, reason)
	}
}

func normalizeFiberErrorReason(status int, message string) string {
	switch {
	case message == "":
		return "internal_error"
	case status == fiber.StatusNotFound && message == "Not Found":
		return "not_found"
	case status == fiber.StatusMethodNotAllowed && message == "Method Not Allowed":
		return "method_not_allowed"
	default:
		return message
	}
}
```

- [ ] **Step 6: Install middleware and wrap health**

In `apps/api/app/Http/server.go`, add the locale middleware after recover:

```go
app.Use(requestid.New())
app.Use(recover.New())
app.Use(localeMiddleware(cfg))
```

Change the health route return statement to:

```go
return OK(c, healthResponse{
	Name:             cfg.AppName,
	Status:           "ok",
	Environment:      cfg.AppEnv,
	Locale:           cfg.AppLocale,
	SupportedLocales: cfg.SupportedLocales,
	Time:             time.Now().UTC(),
})
```

- [ ] **Step 7: Run HTTP tests and verify this slice passes**

Run:

```bash
cd apps/api
go test ./app/Http -run 'TestHealthEndpoint|TestSessionEndpointReturnsLocalizedEnvelopeError' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit HTTP envelope foundation**

```bash
git add apps/api/app/Http/responses.go apps/api/app/Http/locale.go apps/api/app/Http/errors.go apps/api/app/Http/server.go apps/api/app/Http/server_test.go
git commit -m "feat(api): add localized response envelope"
```

---

### Task 3: Wrap Identity And Role Controller Success Responses

**Files:**
- Modify: `apps/api/app/Http/Controllers/Identity/controller.go`
- Modify: `apps/api/app/Http/server_test.go`

- [ ] **Step 1: Update failing success-response tests**

In `TestHumanVerificationChallengeEndpoint`, decode the envelope:

```go
var body apiEnvelope[map[string]string]
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
	t.Fatalf("decode challenge response: %v", err)
}
if body.Code != nethttp.StatusOK || body.Message != "OK" {
	t.Fatalf("unexpected envelope: %#v", body)
}
if body.Data["challenge"] != "fake" {
	t.Fatalf("expected fake challenge, got %v", body.Data)
}
```

In `TestCreateRoleEndpointAllowsSuperAdmin`, decode the envelope:

```go
var body apiEnvelope[identity.Role]
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
	t.Fatalf("decode role response: %v", err)
}
if body.Code != nethttp.StatusCreated || body.Message != "OK" {
	t.Fatalf("unexpected envelope: %#v", body)
}
if body.Data.Key != "moderator" || body.Data.Alias != "版主" {
	t.Fatalf("unexpected role response: %#v", body.Data)
}
```

Add this logout test:

```go
func TestLogoutEndpointReturnsNoDataEnvelope(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{RouteProviders: []RouteProvider{identityController}})
	cookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body apiEnvelope[*json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode logout envelope: %v", err)
	}
	if body.Code != nethttp.StatusOK || body.Message != "OK" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	if body.Data != nil {
		t.Fatalf("expected null data, got %v", body.Data)
	}
}
```

- [ ] **Step 2: Run the targeted tests and verify they fail**

Run:

```bash
cd apps/api
go test ./app/Http -run 'TestHumanVerificationChallengeEndpoint|TestCreateRoleEndpointAllowsSuperAdmin|TestLogoutEndpointReturnsNoDataEnvelope' -count=1
```

Expected: FAIL because controller success responses are still unwrapped or `204`.

- [ ] **Step 3: Import HTTP helpers in the identity controller**

In `apps/api/app/Http/Controllers/Identity/controller.go`, add this import:

```go
apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
```

- [ ] **Step 4: Replace controller success responses**

In `controller.go`, make these replacements:

```go
return apphttp.Created(c, current)
```

for registration.

```go
return apphttp.OK(c, current)
```

for login and session.

```go
return apphttp.OK(c, challenge.Payload)
```

for human-verification challenge.

```go
return apphttp.NoData(c)
```

for logout, delete role, and replace role permissions.

```go
return apphttp.OK(c, roles)
```

for list roles.

```go
return apphttp.Created(c, role)
```

for create role.

```go
return apphttp.OK(c, role)
```

for update role.

- [ ] **Step 5: Keep register test helper focused on session setup**

Keep `registerHTTPUser` focused on status and cookie. No data decode is needed.
Do not add envelope decoding to this helper because its callers only need the
session cookie.


- [ ] **Step 6: Run all HTTP tests**

Run:

```bash
cd apps/api
go test ./app/Http -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit controller envelope migration**

```bash
git add apps/api/app/Http/Controllers/Identity/controller.go apps/api/app/Http/server_test.go
git commit -m "feat(api): wrap identity responses"
```

---

### Task 4: Complete Backend API Test Coverage

**Files:**
- Modify: `apps/api/app/Http/server_test.go`

- [ ] **Step 1: Update remaining error tests to inspect envelope data**

In `TestRegisterEndpointRequiresHumanVerification`, replace the old
`problemResponse` decode with:

```go
var body apiEnvelope[apiErrorData]
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
	t.Fatalf("decode error envelope: %v", err)
}
if body.Code != nethttp.StatusUnprocessableEntity {
	t.Fatalf("expected envelope code 422, got %d", body.Code)
}
if body.Message != "请先完成人机验证。" {
	t.Fatalf("expected human verification message, got %q", body.Message)
}
if body.Data.Reason != humanverify.CodeRequired {
	t.Fatalf("expected required reason, got %q", body.Data.Reason)
}
```

In `TestCreateRoleEndpointRejectsMember`, add:

```go
var body apiEnvelope[apiErrorData]
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
	t.Fatalf("decode error envelope: %v", err)
}
if body.Code != nethttp.StatusForbidden {
	t.Fatalf("expected envelope code 403, got %d", body.Code)
}
if body.Data.Reason != "permission.denied" {
	t.Fatalf("expected permission.denied reason, got %q", body.Data.Reason)
}
```

- [ ] **Step 2: Add unsupported-locale fallback test**

Add:

```go
func TestErrorEnvelopeFallsBackToDefaultLocale(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{RouteProviders: []RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/session", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer resp.Body.Close()

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Message != "请先登录。" {
		t.Fatalf("expected Chinese fallback message, got %q", body.Message)
	}
}
```

- [ ] **Step 3: Run backend package tests**

Run:

```bash
cd apps/api
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit backend test coverage**

```bash
git add apps/api/app/Http/server_test.go
git commit -m "test(api): cover response envelopes"
```

---

### Task 5: Update OpenAPI Envelope Contract

**Files:**
- Modify: `contracts/openapi.yaml`

- [ ] **Step 1: Replace successful response schemas with envelope schemas**

Under `components.schemas`, add these schemas:

```yaml
    ApiErrorData:
      type: object
      required:
        - reason
      properties:
        reason:
          type: string
        fields:
          type: object
          additionalProperties:
            type: array
            items:
              type: string
    ApiResponseNull:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
          examples: [200]
        message:
          type: string
          examples: [OK]
        data:
          type: "null"
    ApiErrorResponse:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          $ref: "#/components/schemas/ApiErrorData"
    ApiResponseCurrentUser:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          $ref: "#/components/schemas/CurrentUser"
    ApiResponseRole:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          $ref: "#/components/schemas/Role"
    ApiResponseRoleList:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          type: array
          items:
            $ref: "#/components/schemas/Role"
    ApiResponseAltchaChallenge:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          $ref: "#/components/schemas/AltchaChallenge"
    ApiResponseHealth:
      type: object
      required: [code, message, data]
      properties:
        code:
          type: integer
        message:
          type: string
        data:
          $ref: "#/components/schemas/Health"
```

Extract the current inline health object into `components.schemas.Health`.

- [ ] **Step 2: Update reusable error responses**

Change each reusable error response schema reference from `Problem` to:

```yaml
            $ref: "#/components/schemas/ApiErrorResponse"
```

Keep the existing `Problem` schema only if another non-API contract still uses
it. Otherwise remove it from `components.schemas`.

- [ ] **Step 3: Update endpoint success responses**

Use these success schema references:

- `/auth/register` `201`: `ApiResponseCurrentUser`
- `/auth/login` `200`: `ApiResponseCurrentUser`
- `/auth/logout` `200`: `ApiResponseNull`
- `/auth/session` `200`: `ApiResponseCurrentUser`
- `/human-verification/challenge` `200`: `ApiResponseAltchaChallenge`
- `/health` `200`: `ApiResponseHealth`
- `/roles` `GET 200`: `ApiResponseRoleList`
- `/roles` `POST 201`: `ApiResponseRole`
- `/roles/{roleKey}` `PATCH 200`: `ApiResponseRole`
- `/roles/{roleKey}` `DELETE 200`: `ApiResponseNull`
- `/roles/{roleKey}/permissions` `PUT 200`: `ApiResponseNull`

- [ ] **Step 4: Search for obsolete 204 responses**

Run:

```bash
rg '"204"|No Content|Problem' contracts/openapi.yaml
```

Expected: no `204` matches. `Problem` may remain only if deliberately kept and
not referenced by API responses.

- [ ] **Step 5: Commit OpenAPI contract**

```bash
git add contracts/openapi.yaml
git commit -m "docs(api): document response envelope"
```

---

### Task 6: Nuxt API Client And Auth Consumers

**Files:**
- Create: `apps/web/app/composables/useApiClient.ts`
- Modify: `apps/web/app/composables/useAuthSession.ts`
- Modify: `apps/web/app/pages/login.vue`
- Modify: `apps/web/app/pages/register.vue`
- Modify: `apps/web/app/layouts/admin.vue`

- [ ] **Step 1: Create API client utility**

Create `apps/web/app/composables/useApiClient.ts`:

```ts
export type ApiEnvelope<T> = {
  code: number
  message: string
  data: T
}

export type ApiErrorData = {
  reason?: string
  fields?: Record<string, string[]>
}

type ApiFetchOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  body?: unknown
  credentials?: RequestCredentials
  headers?: Record<string, string>
}

export function useApiClient() {
  const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
  const { locale } = useI18n()

  function apiLocale() {
    return locale.value === 'en' ? 'en-US' : locale.value
  }

  function apiHeaders(extra?: Record<string, string>) {
    return {
      ...(import.meta.server ? useRequestHeaders(['cookie']) : {}),
      'Accept-Language': apiLocale(),
      ...extra
    }
  }

  async function request<T>(path: string, options: ApiFetchOptions = {}) {
    const envelope = await $fetch<ApiEnvelope<T>>(`${apiBaseUrl}${path}`, {
      method: options.method,
      body: options.body,
      credentials: options.credentials ?? 'include',
      headers: apiHeaders(options.headers)
    })

    return envelope.data
  }

  return { apiBaseUrl, apiHeaders, request }
}

export function apiErrorMessage(error: unknown) {
  return (error as { data?: { message?: string } })?.data?.message || ''
}

export function apiErrorReason(error: unknown) {
  return (error as { data?: ApiEnvelope<ApiErrorData> })?.data?.data?.reason || ''
}
```

- [ ] **Step 2: Update auth session composable**

In `apps/web/app/composables/useAuthSession.ts`, replace `apiBaseUrl` and
direct `$fetch` usage with:

```ts
const { request } = useApiClient()
```

and:

```ts
user.value = await request<CurrentUser>('/auth/session')
```

- [ ] **Step 3: Update login page**

In `apps/web/app/pages/login.vue`, replace `apiBaseUrl` with:

```ts
const { request } = useApiClient()
```

Change the submit call to:

```ts
await request<CurrentUser>('/auth/login', {
  method: 'POST',
  body: {
    login: form.login,
    password: form.password
  }
})
```

Change the catch block to:

```ts
} catch (error) {
  errorMessage.value = apiErrorMessage(error) || t('errors.loginFailed')
} finally {
```

- [ ] **Step 4: Update register page**

In `apps/web/app/pages/register.vue`, replace `apiBaseUrl` with:

```ts
const { apiBaseUrl, request } = useApiClient()
```

Replace `registerErrorMessage` with:

```ts
function registerErrorMessage(error: unknown) {
  const message = apiErrorMessage(error)
  if (message) {
    return message
  }

  switch (apiErrorReason(error)) {
    case 'human_verification.required':
      return t('errors.humanVerificationRequired')
    case 'human_verification.invalid':
      return t('errors.humanVerificationInvalid')
    case 'human_verification.expired':
      return t('errors.humanVerificationExpired')
    case 'human_verification.replayed':
      return t('errors.humanVerificationReplayed')
    case 'rate_limit.exceeded':
      return t('errors.rateLimited')
    default:
      return t('errors.registerFailed')
  }
}
```

Replace the submit `$fetch` with:

```ts
await request<CurrentUser>('/auth/register', {
  method: 'POST',
  body: {
    username: form.username,
    email: form.email,
    password: form.password,
    displayName: form.displayName,
    locale: locale.value,
    humanVerification: {
      provider: 'altcha',
      token: humanVerificationToken.value
    }
  }
})
```

Keep `apiBaseUrl` for the ALTCHA challenge URL.

- [ ] **Step 5: Update admin logout**

In `apps/web/app/layouts/admin.vue`, replace direct `$fetch` with:

```ts
const { request } = useApiClient()
```

and:

```ts
await request<null>('/auth/logout', {
  method: 'POST'
}).catch(() => null)
```

- [ ] **Step 6: Run frontend typecheck/build**

Run:

```bash
cd apps/web
bun run build
```

Expected: PASS.

- [ ] **Step 7: Commit API client and auth consumers**

```bash
git add apps/web/app/composables/useApiClient.ts apps/web/app/composables/useAuthSession.ts apps/web/app/pages/login.vue apps/web/app/pages/register.vue apps/web/app/layouts/admin.vue
git commit -m "feat(web): unwrap API response envelopes"
```

---

### Task 7: Nuxt Roles Page Envelope Migration

**Files:**
- Modify: `apps/web/app/pages/admin/roles.vue`

- [ ] **Step 1: Update role page fetch to read the envelope**

In `apps/web/app/pages/admin/roles.vue`, add the type import:

```ts
import type { ApiEnvelope } from '~/composables/useApiClient'
```

Then replace the request setup with:

```ts
const { apiBaseUrl, apiHeaders } = useApiClient()

const { data: rolesEnvelope, pending, error, refresh } = await useFetch<ApiEnvelope<Role[]>>(`${apiBaseUrl}/roles`, {
  credentials: 'include',
  headers: apiHeaders(),
  default: () => ({
    code: 200,
    message: 'OK',
    data: []
  })
})
```

Then add:

```ts
const roles = computed(() => rolesEnvelope.value?.data ?? [])
```

Keep the existing `filteredRoles` computed using `roles.value`.

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd apps/web
bun run build
```

Expected: PASS.

- [ ] **Step 3: Commit roles page migration**

```bash
git add apps/web/app/pages/admin/roles.vue
git commit -m "feat(web): unwrap roles envelope"
```

---

### Task 8: Full Verification And Handoff

**Files:**
- Modify: `knowledge/sessions/2026-07-04-api-response-envelope-localized-message.md`

- [ ] **Step 1: Run backend tests**

Run:

```bash
cd apps/api
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd apps/web
bun run build
```

Expected: PASS.

- [ ] **Step 3: Run repository test script if available**

Run:

```bash
./scripts/test.sh
```

Expected: PASS. If the script fails because a required local service is not
running, record the exact failure and keep the backend/frontend verification
results in the handoff.

- [ ] **Step 4: Update implementation handoff**

Append this section to
`knowledge/sessions/2026-07-04-api-response-envelope-localized-message.md`:

```md
## Implementation Verification

- Backend API responses now return `{ code, message, data }`.
- Backend API `message` values are localized with `Accept-Language` support.
- Nuxt API consumers unwrap `data` and prefer backend `message` for API errors.
- Verification:
  - `cd apps/api && go test ./... -count=1`
  - `cd apps/web && bun run build`
  - `./scripts/test.sh`
```

If `./scripts/test.sh` could not run successfully, replace that bullet with the
actual command output summary.

- [ ] **Step 5: Commit handoff**

```bash
git add knowledge/sessions/2026-07-04-api-response-envelope-localized-message.md
git commit -m "docs: record API envelope implementation handoff"
```

- [ ] **Step 6: Final status check**

Run:

```bash
git status --short
```

Expected: no output.

---

## Self-Review Notes

- Spec coverage: tasks cover backend envelope, localized API messages,
  `data.reason`, no-payload `200` responses, OpenAPI, Nuxt unwrapping, locale
  headers, and verification.
- Boundary check: backend helpers live in `app/Http`; localization message
  lookup lives in `app/Support/Localization`; Nuxt uses one composable for API
  response handling.
- Type consistency: backend envelope fields are `code`, `message`, `data`;
  frontend type is `ApiEnvelope<T>` with the same fields; error details use
  `reason` and optional `fields`.
