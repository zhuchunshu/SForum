# ALTCHA Human Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add ALTCHA-backed human verification to registration with a reusable backend provider boundary, challenge endpoint, Redis replay/rate-limit storage, API contract updates, and Nuxt registration UI integration.

**Architecture:** Add `apps/api/app/Support/HumanVerify` as the shared verification boundary. Bootstrap wires an ALTCHA v2 provider plus Redis-backed store into the identity controller; tests can inject fake providers so existing identity behavior stays focused. Nuxt renders the ALTCHA widget on registration and sends `{ provider, token }` with the register request.

**Tech Stack:** Go Fiber v3, `github.com/altcha-org/altcha-lib-go/v2`, `github.com/redis/go-redis/v9`, Fiber Redis storage, Nuxt 4/Vue 3, ALTCHA Web Component, Bun.

---

### Task 1: Configuration And Dependency Baseline

**Files:**
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`
- Modify: `apps/api/config/config.go`
- Modify: `apps/api/config/config_test.go`
- Modify: `.env.example`
- Modify: `.env.production.example`

- [ ] **Step 1: Write failing config tests**

Add assertions to `apps/api/config/config_test.go`:

```go
if cfg.RedisPassword != "" {
	t.Fatalf("expected empty redis password default, got %q", cfg.RedisPassword)
}
if cfg.HumanVerificationProvider != "altcha" {
	t.Fatalf("expected altcha provider default, got %q", cfg.HumanVerificationProvider)
}
if cfg.AltchaChallengeTTL != 10*time.Minute {
	t.Fatalf("expected altcha challenge ttl 10m, got %s", cfg.AltchaChallengeTTL)
}
if cfg.AltchaCost != 1000 {
	t.Fatalf("expected altcha cost 1000, got %d", cfg.AltchaCost)
}
```

Add env parsing assertions in `TestLoadParsesWorkerConfigFromEnv`:

```go
t.Setenv("REDIS_PASSWORD", "secret")
t.Setenv("HUMAN_VERIFICATION_PROVIDER", "disabled")
t.Setenv("ALTCHA_SECRET", "test-altcha-secret")
t.Setenv("ALTCHA_CHALLENGE_TTL", "2m")
t.Setenv("ALTCHA_COST", "2000")
```

Expected assertions:

```go
if cfg.RedisPassword != "secret" {
	t.Fatalf("expected redis password from env, got %q", cfg.RedisPassword)
}
if cfg.HumanVerificationProvider != "disabled" {
	t.Fatalf("expected disabled provider from env, got %q", cfg.HumanVerificationProvider)
}
if cfg.AltchaSecret != "test-altcha-secret" {
	t.Fatalf("expected altcha secret from env, got %q", cfg.AltchaSecret)
}
if cfg.AltchaChallengeTTL != 2*time.Minute {
	t.Fatalf("expected altcha ttl from env, got %s", cfg.AltchaChallengeTTL)
}
if cfg.AltchaCost != 2000 {
	t.Fatalf("expected altcha cost from env, got %d", cfg.AltchaCost)
}
```

- [ ] **Step 2: Run config test and confirm RED**

Run: `go test ./config -run TestLoad -count=1`

Expected: FAIL with missing `RedisPassword`, `HumanVerificationProvider`, `AltchaSecret`, `AltchaChallengeTTL`, or `AltchaCost` fields.

- [ ] **Step 3: Implement config fields**

Add these fields to `config.Config`:

```go
RedisPassword             string
HumanVerificationProvider string
AltchaSecret              string
AltchaChallengeTTL        time.Duration
AltchaCost                int
```

Set defaults in `Load()`:

```go
RedisPassword:             env("REDIS_PASSWORD", ""),
HumanVerificationProvider: env("HUMAN_VERIFICATION_PROVIDER", "altcha"),
AltchaSecret:              env("ALTCHA_SECRET", "sforum-dev-altcha-secret"),
AltchaChallengeTTL:        envDuration("ALTCHA_CHALLENGE_TTL", 10*time.Minute),
AltchaCost:                envPositiveInt("ALTCHA_COST", 1000),
```

- [ ] **Step 4: Run config test and confirm GREEN**

Run: `go test ./config -run TestLoad -count=1`

Expected: PASS.

- [ ] **Step 5: Fix dependencies**

Use the ALTCHA v2 module path:

```sh
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
go get github.com/altcha-org/altcha-lib-go/v2@latest
go mod tidy
```

Expected: `apps/api/go.mod` requires `github.com/altcha-org/altcha-lib-go/v2` and does not keep unused v1.

- [ ] **Step 6: Update env examples**

Add:

```env
HUMAN_VERIFICATION_PROVIDER=altcha
ALTCHA_SECRET=sforum-dev-altcha-secret
ALTCHA_CHALLENGE_TTL=10m
ALTCHA_COST=1000
```

Use `change-me` for `ALTCHA_SECRET` in `.env.production.example`.

### Task 2: HumanVerify Support Package

**Files:**
- Create: `apps/api/app/Support/HumanVerify/types.go`
- Create: `apps/api/app/Support/HumanVerify/service.go`
- Create: `apps/api/app/Support/HumanVerify/memory_store.go`
- Create: `apps/api/app/Support/HumanVerify/redis_store.go`
- Create: `apps/api/app/Support/HumanVerify/altcha_provider.go`
- Test: `apps/api/app/Support/HumanVerify/service_test.go`
- Test: `apps/api/app/Support/HumanVerify/altcha_provider_test.go`
- Test: `apps/api/app/Support/HumanVerify/memory_store_test.go`

- [ ] **Step 1: Write failing service tests**

Cover these behaviors:

```go
func TestServiceRequiresTokenWhenEnabled(t *testing.T) {
	service := humanverify.NewService(humanverify.ServiceConfig{Enabled: true}, fakeProvider{}, humanverify.NewMemoryStore())
	err := service.Verify(context.Background(), humanverify.VerifyRequest{Purpose: humanverify.PurposeRegister})
	if !errors.Is(err, humanverify.ErrRequired) {
		t.Fatalf("expected ErrRequired, got %v", err)
	}
}

func TestServiceRejectsReplayedToken(t *testing.T) {
	store := humanverify.NewMemoryStore()
	provider := fakeProvider{code: humanverify.CodeOK}
	service := humanverify.NewService(humanverify.ServiceConfig{Enabled: true}, provider, store)
	req := humanverify.VerifyRequest{Purpose: humanverify.PurposeRegister, Provider: humanverify.ProviderAltcha, Token: "token-1"}
	if err := service.Verify(context.Background(), req); err != nil {
		t.Fatalf("first verify returned error: %v", err)
	}
	if err := service.Verify(context.Background(), req); !errors.Is(err, humanverify.ErrReplayed) {
		t.Fatalf("expected replay error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./app/Support/HumanVerify -count=1`

Expected: FAIL because package does not exist.

- [ ] **Step 3: Implement types and service**

Implement constants:

```go
const (
	ProviderAltcha = "altcha"
	CodeOK         = "ok"
	CodeRequired   = "human_verification.required"
	CodeInvalid    = "human_verification.invalid"
	CodeExpired    = "human_verification.expired"
	CodeReplayed   = "human_verification.replayed"
	CodeRateLimited = "rate_limit.exceeded"
)
```

Implement `Service.Challenge` and `Service.Verify` so disabled service returns no-op success, enabled service requires a token, delegates provider verification, maps provider result codes to errors, and calls `store.MarkUsed` for replay protection.

- [ ] **Step 4: Implement memory store**

Use a mutex-protected map with expiry timestamps for test and local disabled-provider flows. `MarkUsed(ctx, key, ttl)` returns `ErrReplayed` if the non-expired key already exists.

- [ ] **Step 5: Implement Redis store**

Use `go-redis/v9` with:

```go
SetNX(ctx, "humanverify:used:"+key, "1", ttl)
Incr(ctx, "humanverify:rate:"+key)
Expire(ctx, "humanverify:rate:"+key, ttl)
```

Keep the store small and closeable.

- [ ] **Step 6: Implement ALTCHA v2 provider**

Use `github.com/altcha-org/altcha-lib-go/v2`:

```go
challenge, err := altcha.CreateChallenge(altcha.CreateChallengeOptions{
	Algorithm:           "PBKDF2/SHA-256",
	Cost:                cfg.Cost,
	ExpiresAt:           &expiresAt,
	HMACSignatureSecret: cfg.Secret,
})
```

Verify payloads by base64-decoding JSON into `altcha.Payload` and calling:

```go
result, err := altcha.VerifySolution(altcha.VerifySolutionOptions{
	Challenge:           payload.Challenge,
	Solution:            payload.Solution,
	DeriveKey:           altcha.DeriveKeyPBKDF2(),
	HMACSignatureSecret: cfg.Secret,
})
```

Map `result.Expired` to `ErrExpired` and invalid signature/solution to `ErrInvalid`.

- [ ] **Step 7: Run package tests and confirm GREEN**

Run: `go test ./app/Support/HumanVerify -count=1`

Expected: PASS.

### Task 3: Identity Controller Enforcement And Challenge Route

**Files:**
- Modify: `apps/api/app/Http/Controllers/Identity/controller.go`
- Modify: `apps/api/app/Http/Controllers/Identity/routes.go`
- Modify: `apps/api/app/Http/server_test.go`

- [ ] **Step 1: Write failing HTTP tests**

Add tests:

```go
func TestRegisterEndpointRequiresHumanVerification(t *testing.T) {
	identityController := identitycontroller.NewControllerWithVerifier(identity.NewService(newHTTPFakeStore()), session.NewStore(), requiredFakeVerifier())
	app := NewApp(testConfig(), slog.Default(), Dependencies{RouteProviders: []RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`{"username":"admin","email":"admin@example.com","password":"correct horse battery staple"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}
```

Add a success test using token:

```json
{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","humanVerification":{"provider":"altcha","token":"valid-token"}}
```

- [ ] **Step 2: Run HTTP tests and confirm RED**

Run: `go test ./app/Http -run 'TestRegisterEndpoint.*HumanVerification|TestRegisterEndpointCreatesSession' -count=1`

Expected: FAIL because constructor/request fields do not exist.

- [ ] **Step 3: Add controller verifier wiring**

Add `NewControllerWithVerifier(service, sessions, verifier)` while keeping `NewController(service, sessions)` as disabled/default for existing tests. Extend `registerRequest`:

```go
HumanVerification humanVerificationRequest `json:"humanVerification"`
```

Before `service.Register`, call verifier with purpose `register`, client IP, provider, and token.

- [ ] **Step 4: Add challenge route**

Register:

```go
api.Get("/human-verification/challenge", h.humanVerificationChallenge)
```

Only allow supported purposes such as `register`. Return `422 validation.invalid` for unknown purpose.

- [ ] **Step 5: Map verification errors**

Map:

- `ErrRequired` -> `422 human_verification.required`
- `ErrInvalid` -> `422 human_verification.invalid`
- `ErrExpired` -> `422 human_verification.expired`
- `ErrReplayed` -> `422 human_verification.replayed`
- `ErrRateLimited` -> `429 rate_limit.exceeded`

- [ ] **Step 6: Run HTTP tests and confirm GREEN**

Run: `go test ./app/Http -run 'TestRegisterEndpoint.*HumanVerification|TestRegisterEndpointCreatesSession' -count=1`

Expected: PASS.

### Task 4: Bootstrap Runtime Wiring

**Files:**
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/app/Providers/identity.go`
- Modify: `apps/api/app/Support/Redis/session.go`
- Modify: `apps/api/app/Support/Redis/session_test.go`
- Test: `apps/api/bootstrap/app_test.go`

- [ ] **Step 1: Write failing bootstrap/Redis tests**

Update Redis session test so `NewStorage("redis:6379", "secret")` preserves password in config behavior, and add a bootstrap helper test for provider mode:

```go
cfg := config.Config{HumanVerificationProvider: "disabled"}
service := newHumanVerifyService(cfg, nil)
if service.Enabled() {
	t.Fatal("expected disabled provider to be disabled")
}
```

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./app/Support/Redis ./bootstrap -count=1`

Expected: FAIL because signatures/helpers do not exist.

- [ ] **Step 3: Wire Redis password and human verifier**

Change `redisplatform.NewStorage(cfg.RedisAddr)` to `redisplatform.NewStorage(cfg.RedisAddr, cfg.RedisPassword)`.

Create a Redis client for `HumanVerify.NewRedisStore`, wire ALTCHA provider for `altcha`, and wire disabled service for `disabled`.

- [ ] **Step 4: Run tests and confirm GREEN**

Run: `go test ./app/Support/Redis ./bootstrap -count=1`

Expected: PASS.

### Task 5: API Contract And Frontend Registration

**Files:**
- Modify: `contracts/openapi.yaml`
- Modify: `apps/web/package.json`
- Modify: `apps/web/bun.lock`
- Create: `apps/web/app/plugins/altcha.client.ts`
- Modify: `apps/web/app/pages/register.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add frontend dependency**

Run:

```sh
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
bun add altcha
```

Expected: `apps/web/package.json` and `apps/web/bun.lock` include `altcha`.

- [ ] **Step 2: Add plugin**

Create:

```ts
import 'altcha'

export default defineNuxtPlugin(() => {})
```

- [ ] **Step 3: Update register page**

Add token state:

```ts
const humanVerificationToken = ref('')

function handleAltchaVerified(event: Event) {
  const detail = (event as CustomEvent<{ payload?: string }>).detail
  humanVerificationToken.value = detail?.payload || ''
}
```

Send:

```ts
humanVerification: {
  provider: 'altcha',
  token: humanVerificationToken.value
}
```

Render:

```vue
<ClientOnly>
  <altcha-widget
    :challengeurl="`${apiBaseUrl}/human-verification/challenge?purpose=register`"
    @verified="handleAltchaVerified"
    @expired="humanVerificationToken = ''"
  />
</ClientOnly>
```

- [ ] **Step 4: Add translations**

Add auth labels and error messages:

```json
"humanVerification": "人机验证",
"humanVerificationHint": "请先完成人机验证。",
"humanVerificationRequired": "请先完成人机验证。",
"humanVerificationInvalid": "人机验证失败，请重新验证。",
"humanVerificationExpired": "人机验证已过期，请重新验证。",
"humanVerificationReplayed": "本次验证已使用，请重新验证。",
"rateLimited": "操作过于频繁，请稍后再试。"
```

Add English equivalents.

- [ ] **Step 5: Update OpenAPI**

Add:

- `GET /human-verification/challenge`
- `HumanVerification` schema
- `ChallengeResponse` schema
- `humanVerification` field in `RegisterRequest`
- `429` response

- [ ] **Step 6: Run frontend type/build verification**

Run: `bun run typecheck`

Expected: PASS.

### Task 6: Full Verification And Handoff

**Files:**
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/identity.md`
- Create: `knowledge/sessions/2026-07-04-altcha-human-verification-implementation.md`

- [ ] **Step 1: Run backend verification**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 2: Run frontend verification**

Run: `bun run typecheck`

Expected: PASS.

- [ ] **Step 3: Run whitespace check**

Run: `git diff --check`

Expected: PASS.

- [ ] **Step 4: Update knowledge handoff**

Record changed files, decisions, next steps, and open questions.

- [ ] **Step 5: Stage only owned files**

Do not stage unrelated pre-existing user edits unless they are now intentionally part of registration UI integration.

- [ ] **Step 6: Commit**

```sh
git commit -m "feat(auth): add altcha human verification"
```
