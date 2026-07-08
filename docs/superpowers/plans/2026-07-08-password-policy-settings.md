# Password Policy Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make password policy configurable in admin settings and show real-time password qualification feedback on registration and password reset forms.

**Architecture:** Store password policy in public/admin runtime options, validate all password creation paths through a shared identity policy model, and let Nuxt render policy progress from the same public options. Keep `HashPassword` focused on Argon2id hashing and keep API checks authoritative.

**Tech Stack:** Go/Fiber backend, PostgreSQL-backed `web_options`, Nuxt 4/Vue 3 frontend, Bun tests, Go tests, modular OpenAPI YAML.

---

## File Structure

- Modify `apps/api/app/Models/Identity/password.go`: add `PasswordPolicy`, recommended defaults, validation helpers, and remove mutable policy checks from `HashPassword`.
- Modify `apps/api/app/Models/Identity/password_test.go`: cover policy validation and hashing behavior.
- Modify `apps/api/app/Models/Identity/types.go`: add stable password policy message keys.
- Modify `apps/api/app/Models/Identity/service.go`: inject a password policy resolver and validate registration through it.
- Modify `apps/api/app/Models/Identity/service_test.go`: cover customized registration policy behavior.
- Modify `apps/api/app/Models/Identity/password_reset_service.go`: inject policy resolver and validate reset passwords through it.
- Modify `apps/api/app/Models/Identity/password_reset_service_test.go`: cover customized reset policy behavior.
- Modify `apps/api/app/Models/Options/types.go`: add `identity.password.*` option constants.
- Modify `apps/api/app/Models/Options/service.go`: define defaults, validation, public/admin exposure, and `PasswordPolicy(ctx)`.
- Modify `apps/api/app/Models/Options/service_test.go`: cover defaults, validation, merged min/max rejection, and public listing.
- Modify `apps/api/app/Providers/identity.go` and `apps/api/app/Http/Controllers/Identity/controller.go`: wire the options-backed policy resolver into identity services.
- Modify `apps/web/app/composables/useWebOptions.ts`: add password policy option parsing and requirement helpers.
- Modify `apps/web/tests/useWebOptions.test.ts`: cover password policy helper behavior.
- Modify `extensions/builtin/themes/sforum-default/layer/app/pages/register.vue`: render password progress and requirement rows below the password field.
- Modify `extensions/builtin/themes/sforum-default/layer/app/pages/reset-password.vue`: reuse policy for submit gating and progress feedback.
- Modify `apps/web/app/pages/admin/settings/index.vue`: add account-security controls and recommended policy reset in the basic settings tab.
- Modify `apps/web/i18n/locales/zh-CN.json` and `apps/web/i18n/locales/en-US.json`: add auth progress and admin policy strings.
- Modify `contracts/openapi/schemas/options.yaml` and `contracts/openapi/paths/identity.yaml`: document new option names and configurable password policy examples.
- Modify `knowledge/modules/identity.md`, `knowledge/modules/options.md`, and add `knowledge/sessions/2026-07-08-password-policy-settings.md`.

---

### Task 1: Backend Password Policy Model

**Files:**
- Modify: `apps/api/app/Models/Identity/password.go`
- Modify: `apps/api/app/Models/Identity/password_test.go`
- Modify: `apps/api/app/Models/Identity/types.go`

- [ ] **Step 1: Write failing policy tests**

Add tests to `apps/api/app/Models/Identity/password_test.go`:

```go
func TestPasswordPolicyValidatesLengthAndRequiredClasses(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:        10,
		MaxLength:        20,
		RequireLowercase: true,
		RequireUppercase: true,
		RequireNumber:    true,
		RequireSymbol:    true,
	}

	fields := policy.Validate("short")
	if len(fields[FieldPassword]) == 0 {
		t.Fatal("expected password policy errors")
	}
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordMin) {
		t.Fatalf("expected min length error, got %#v", fields)
	}
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordUppercase) {
		t.Fatalf("expected uppercase error, got %#v", fields)
	}

	fields = policy.Validate("Correct-1234")
	if len(fields) != 0 {
		t.Fatalf("expected valid password, got %#v", fields)
	}
}

func TestPasswordPolicyCountsUnicodeRunes(t *testing.T) {
	policy := PasswordPolicy{MinLength: 4, MaxLength: 8}

	if fields := policy.Validate("密码短"); !fieldMessagesContain(fields, FieldPassword, MessagePasswordMin) {
		t.Fatalf("expected unicode rune min length error, got %#v", fields)
	}
	if fields := policy.Validate("密码短句"); len(fields) != 0 {
		t.Fatalf("expected unicode password to pass, got %#v", fields)
	}
}

func TestHashPasswordDoesNotOwnRuntimePolicy(t *testing.T) {
	hash, err := HashPassword("short")
	if err != nil {
		t.Fatalf("HashPassword should only hash, got %v", err)
	}
	ok, err := VerifyPassword("short", hash)
	if err != nil || !ok {
		t.Fatalf("expected short password hash to verify, ok=%v err=%v", ok, err)
	}
}

func fieldMessagesContain(fields FieldMessages, field string, message string) bool {
	for _, item := range fields[field] {
		if item == message {
			return true
		}
	}
	return false
}
```

Remove `TestHashPasswordRejectsShortPassword`.

- [ ] **Step 2: Run tests and confirm red**

Run:

```bash
go test ./app/Models/Identity -run 'TestPasswordPolicy|TestHashPassword' -count=1
```

Expected: fail because `PasswordPolicy` and new message constants do not exist, and `HashPassword("short")` still rejects.

- [ ] **Step 3: Implement policy model**

In `apps/api/app/Models/Identity/types.go`, add:

```go
	MessagePasswordMax       = "auth.password_max_length"
	MessagePasswordLowercase = "auth.password_lowercase"
	MessagePasswordUppercase = "auth.password_uppercase"
	MessagePasswordNumber    = "auth.password_number"
	MessagePasswordSymbol    = "auth.password_symbol"
```

In `apps/api/app/Models/Identity/password.go`, add imports for `unicode` and implement:

```go
const (
	RecommendedPasswordMinLength = 12
	RecommendedPasswordMaxLength = 128
)

type PasswordPolicy struct {
	MinLength        int
	MaxLength        int
	RequireLowercase bool
	RequireUppercase bool
	RequireNumber    bool
	RequireSymbol    bool
}

func RecommendedPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinLength: RecommendedPasswordMinLength, MaxLength: RecommendedPasswordMaxLength}
}

func (p PasswordPolicy) Normalized() PasswordPolicy {
	if p.MinLength <= 0 {
		p.MinLength = RecommendedPasswordMinLength
	}
	if p.MaxLength <= 0 {
		p.MaxLength = RecommendedPasswordMaxLength
	}
	if p.MaxLength < p.MinLength {
		p.MaxLength = p.MinLength
	}
	return p
}

func (p PasswordPolicy) Validate(password string) FieldMessages {
	p = p.Normalized()
	fields := FieldMessages{}
	length := len([]rune(password))
	if length < p.MinLength {
		addFieldMessage(fields, FieldPassword, MessagePasswordMin)
	}
	if length > p.MaxLength {
		addFieldMessage(fields, FieldPassword, MessagePasswordMax)
	}

	var hasLower, hasUpper, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsNumber(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if p.RequireLowercase && !hasLower {
		addFieldMessage(fields, FieldPassword, MessagePasswordLowercase)
	}
	if p.RequireUppercase && !hasUpper {
		addFieldMessage(fields, FieldPassword, MessagePasswordUppercase)
	}
	if p.RequireNumber && !hasNumber {
		addFieldMessage(fields, FieldPassword, MessagePasswordNumber)
	}
	if p.RequireSymbol && !hasSymbol {
		addFieldMessage(fields, FieldPassword, MessagePasswordSymbol)
	}
	return fields
}
```

Remove the length check from `HashPassword`.

- [ ] **Step 4: Run tests and confirm green**

Run:

```bash
go test ./app/Models/Identity -run 'TestPasswordPolicy|TestHashPassword|TestVerifyPassword' -count=1
```

Expected: pass.

---

### Task 2: Runtime Options For Password Policy

**Files:**
- Modify: `apps/api/app/Models/Options/types.go`
- Modify: `apps/api/app/Models/Options/service.go`
- Modify: `apps/api/app/Models/Options/service_test.go`

- [ ] **Step 1: Write failing option tests**

Add to `apps/api/app/Models/Options/service_test.go`:

```go
func TestServicePasswordPolicyOptionsArePublicWithRecommendedDefaults(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if got := adminValueFromPublic(items, NameIdentityPasswordMinLength); got != "12" {
		t.Fatalf("expected public min length default, got %q", got)
	}
	if got := adminValueFromPublic(items, NameIdentityPasswordMaxLength); got != "128" {
		t.Fatalf("expected public max length default, got %q", got)
	}
	if got := adminValueFromPublic(items, NameIdentityPasswordRequireUppercase); got != "disabled" {
		t.Fatalf("expected uppercase disabled default, got %q", got)
	}
}

func TestServicePasswordPolicyOptionsValidation(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)
	actor := settingsActor()

	updated, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityPasswordMinLength, Value: "14"},
		{Name: NameIdentityPasswordMaxLength, Value: "160"},
		{Name: NameIdentityPasswordRequireLowercase, Value: "true"},
		{Name: NameIdentityPasswordRequireNumber, Value: "enabled"},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}
	if got := adminValue(updated, NameIdentityPasswordMinLength); got != "14" {
		t.Fatalf("expected min length 14, got %q", got)
	}
	if got := adminValue(updated, NameIdentityPasswordRequireLowercase); got != "enabled" {
		t.Fatalf("expected lowercase enabled, got %q", got)
	}

	cases := []UpdateInput{
		{Name: NameIdentityPasswordMinLength, Value: "7"},
		{Name: NameIdentityPasswordMinLength, Value: "129"},
		{Name: NameIdentityPasswordMaxLength, Value: "63"},
		{Name: NameIdentityPasswordMaxLength, Value: "513"},
		{Name: NameIdentityPasswordRequireSymbol, Value: "sometimes"},
	}
	for _, input := range cases {
		if _, err := service.Update(context.Background(), actor, input); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("expected invalid password option for %#v, got %v", input, err)
		}
	}

	if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameIdentityPasswordMinLength, Value: "80"},
		{Name: NameIdentityPasswordMaxLength, Value: "64"},
	}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected max below min to be invalid, got %v", err)
	}
}

func TestServiceResolvesPasswordPolicy(t *testing.T) {
	store := &fakeStore{items: map[string]string{
		NameIdentityPasswordMinLength:        "14",
		NameIdentityPasswordMaxLength:        "160",
		NameIdentityPasswordRequireUppercase: "enabled",
		NameIdentityPasswordRequireSymbol:    "enabled",
	}}
	service := NewServiceWithCacheTTL(store, time.Minute)

	policy, err := service.PasswordPolicy(context.Background())
	if err != nil {
		t.Fatalf("PasswordPolicy returned error: %v", err)
	}
	if policy.MinLength != 14 || policy.MaxLength != 160 || !policy.RequireUppercase || !policy.RequireSymbol {
		t.Fatalf("unexpected policy: %#v", policy)
	}
}
```

- [ ] **Step 2: Run tests and confirm red**

Run:

```bash
go test ./app/Models/Options -run 'TestServicePasswordPolicy' -count=1
```

Expected: fail because option constants and `PasswordPolicy` resolver do not exist.

- [ ] **Step 3: Implement option constants and defaults**

Add constants in `apps/api/app/Models/Options/types.go`:

```go
	NameIdentityPasswordMinLength        = "identity.password.min_length"
	NameIdentityPasswordMaxLength        = "identity.password.max_length"
	NameIdentityPasswordRequireLowercase = "identity.password.require_lowercase"
	NameIdentityPasswordRequireUppercase = "identity.password.require_uppercase"
	NameIdentityPasswordRequireNumber    = "identity.password.require_number"
	NameIdentityPasswordRequireSymbol    = "identity.password.require_symbol"
```

In `apps/api/app/Models/Options/service.go`, add min/max constants, option definitions with `public: true` and `settings.manage`, defaults, normalization cases, coerce checks, and value-set validation.

Add:

```go
const (
	passwordMinLengthMin = 8
	passwordMinLengthMax = 128
	passwordMaxLengthMin = 64
	passwordMaxLengthMax = 512
)

func (s *Service) PasswordPolicy(ctx context.Context) (identity.PasswordPolicy, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return identity.PasswordPolicy{}, err
	}
	minLength, _ := strictAtoi(values[NameIdentityPasswordMinLength])
	maxLength, _ := strictAtoi(values[NameIdentityPasswordMaxLength])
	return identity.PasswordPolicy{
		MinLength:        minLength,
		MaxLength:        maxLength,
		RequireLowercase: isEnabledOption(values[NameIdentityPasswordRequireLowercase]),
		RequireUppercase: isEnabledOption(values[NameIdentityPasswordRequireUppercase]),
		RequireNumber:    isEnabledOption(values[NameIdentityPasswordRequireNumber]),
		RequireSymbol:    isEnabledOption(values[NameIdentityPasswordRequireSymbol]),
	}.Normalized(), nil
}
```

Ensure `isValidValueSet` rejects `max_length < min_length`.

- [ ] **Step 4: Run tests and confirm green**

Run:

```bash
go test ./app/Models/Options -run 'TestServicePasswordPolicy|TestServiceListsOnlyPublicOptions|TestServiceRejectsUnknownOrEmptyOption' -count=1
```

Expected: pass.

---

### Task 3: Identity Service Integration

**Files:**
- Modify: `apps/api/app/Models/Identity/service.go`
- Modify: `apps/api/app/Models/Identity/service_test.go`
- Modify: `apps/api/app/Models/Identity/password_reset_service.go`
- Modify: `apps/api/app/Models/Identity/password_reset_service_test.go`
- Modify: `apps/api/app/Providers/identity.go`
- Modify: `apps/api/app/Http/Controllers/Identity/controller.go`

- [ ] **Step 1: Write failing registration tests**

Add to `apps/api/app/Models/Identity/service_test.go`:

```go
func TestRegisterUsesConfiguredPasswordPolicy(t *testing.T) {
	store := newFakeStore()
	service := NewServiceWithPasswordPolicy(store, staticPasswordPolicyResolver{policy: PasswordPolicy{
		MinLength:        8,
		MaxLength:        64,
		RequireUppercase: true,
		RequireNumber:    true,
	}})

	err := service.ValidateRegister(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "lowercaseonly",
	})
	fields := registerInvalidFields(t, err)
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordUppercase) {
		t.Fatalf("expected uppercase policy error, got %#v", fields)
	}
	if !fieldMessagesContain(fields, FieldPassword, MessagePasswordNumber) {
		t.Fatalf("expected number policy error, got %#v", fields)
	}

	_, err = service.Register(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "Alice123",
	})
	if err != nil {
		t.Fatalf("expected password matching configured policy to register: %v", err)
	}
}

type staticPasswordPolicyResolver struct {
	policy PasswordPolicy
	err    error
}

func (r staticPasswordPolicyResolver) PasswordPolicy(context.Context) (PasswordPolicy, error) {
	return r.policy, r.err
}
```

- [ ] **Step 2: Write failing password reset test**

Add to `apps/api/app/Models/Identity/password_reset_service_test.go`:

```go
func TestPasswordResetConfirmUsesConfiguredPasswordPolicy(t *testing.T) {
	store := newResetFakeStore()
	service := NewPasswordResetServiceWithPasswordPolicy(store, nil, PasswordResetConfig{}, staticPasswordPolicyResolver{policy: PasswordPolicy{
		MinLength:     8,
		MaxLength:     64,
		RequireSymbol: true,
	}})

	err := service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "some-token", NewPassword: "longenough"})
	if !errors.As(err, new(*RegisterInvalidError)) {
		t.Fatalf("expected field-aware password policy error, got %v", err)
	}

	err = service.ConfirmPasswordReset(context.Background(), ConfirmPasswordResetInput{Token: "some-token", NewPassword: "long-enough!"})
	if err != nil {
		t.Fatalf("expected password reset to accept configured policy: %v", err)
	}
}
```

- [ ] **Step 3: Run tests and confirm red**

Run:

```bash
go test ./app/Models/Identity -run 'TestRegisterUsesConfiguredPasswordPolicy|TestPasswordResetConfirmUsesConfiguredPasswordPolicy' -count=1
```

Expected: fail because constructors and policy resolver integration do not exist.

- [ ] **Step 4: Implement policy resolver injection**

In `service.go`, add:

```go
type PasswordPolicyResolver interface {
	PasswordPolicy(ctx context.Context) (PasswordPolicy, error)
}

type staticRecommendedPasswordPolicy struct{}

func (staticRecommendedPasswordPolicy) PasswordPolicy(context.Context) (PasswordPolicy, error) {
	return RecommendedPasswordPolicy(), nil
}
```

Extend `Service` with `passwordPolicies PasswordPolicyResolver`, add `NewServiceWithPasswordPolicy`, and have existing constructors use the recommended resolver. Replace `validateRegisterInput(..., password)` with a version that accepts `PasswordPolicy` and merges `policy.Validate(password)`.

In `password_reset_service.go`, add `NewPasswordResetServiceWithPasswordPolicy`, store the resolver, validate `input.NewPassword` before consuming the token, and return `NewRegisterInvalid(fields)` for policy failures.

- [ ] **Step 5: Wire options resolver into providers/controllers**

Update the provider/controller `optionsResolver` interfaces to include:

```go
PasswordPolicy(ctx context.Context) (identity.PasswordPolicy, error)
```

Use `identity.NewServiceWithPasswordPolicy(store, options)` and `identity.NewPasswordResetServiceWithPasswordPolicy(...)` where options are available; keep recommended defaults for tests or constructors without options.

- [ ] **Step 6: Run identity tests**

Run:

```bash
go test ./app/Models/Identity ./app/Http/Controllers/Identity ./app/Providers -count=1
```

Expected: pass.

---

### Task 4: Frontend Password Policy Helpers

**Files:**
- Modify: `apps/web/app/composables/useWebOptions.ts`
- Modify: `apps/web/tests/useWebOptions.test.ts`

- [ ] **Step 1: Write failing frontend helper tests**

Add to `apps/web/tests/useWebOptions.test.ts`:

```ts
import {
  resolvePasswordPolicy,
  passwordPolicyRequirements,
  passwordPolicyProgress,
  recommendedPasswordPolicy
} from '../app/composables/useWebOptions'

describe('password policy helpers', () => {
  test('resolves recommended password policy defaults', () => {
    const policy = resolvePasswordPolicy({})

    expect(policy).toEqual(recommendedPasswordPolicy)
  })

  test('computes password requirements and progress', () => {
    const policy = resolvePasswordPolicy({
      'identity.password.min_length': '8',
      'identity.password.max_length': '64',
      'identity.password.require_uppercase': 'enabled',
      'identity.password.require_number': 'enabled',
      'identity.password.require_symbol': 'enabled'
    })

    const weak = passwordPolicyRequirements('lowercase', policy)
    expect(weak.filter(item => item.met).map(item => item.key)).toEqual(['length'])
    expect(passwordPolicyProgress('lowercase', policy)).toBe(25)

    const strong = passwordPolicyRequirements('Passw0rd!', policy)
    expect(strong.every(item => item.met)).toBe(true)
    expect(passwordPolicyProgress('Passw0rd!', policy)).toBe(100)
  })
})
```

- [ ] **Step 2: Run tests and confirm red**

Run:

```bash
bun test tests/useWebOptions.test.ts
```

Expected: fail because helper exports do not exist.

- [ ] **Step 3: Implement frontend helper exports**

In `useWebOptions.ts`, add:

```ts
export type PasswordPolicy = {
  minLength: number
  maxLength: number
  requireLowercase: boolean
  requireUppercase: boolean
  requireNumber: boolean
  requireSymbol: boolean
}

export type PasswordRequirement = {
  key: 'length' | 'lowercase' | 'uppercase' | 'number' | 'symbol'
  met: boolean
}

export const recommendedPasswordPolicy: PasswordPolicy = {
  minLength: 12,
  maxLength: 128,
  requireLowercase: false,
  requireUppercase: false,
  requireNumber: false,
  requireSymbol: false
}
```

Add fallback option values, a `passwordPolicy` computed in `useWebOptions()`, and export:

```ts
export function resolvePasswordPolicy(values: Record<string, string>): PasswordPolicy
export function passwordPolicyRequirements(password: string, policy: PasswordPolicy): PasswordRequirement[]
export function passwordPolicyProgress(password: string, policy: PasswordPolicy): number
```

Use JavaScript unicode-aware regexes:

```ts
/\p{Ll}/u
/\p{Lu}/u
/\p{N}/u
/[\p{P}\p{S}]/u
```

- [ ] **Step 4: Run tests and confirm green**

Run:

```bash
bun test tests/useWebOptions.test.ts
```

Expected: pass.

---

### Task 5: Registration And Reset Password UI Feedback

**Files:**
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/register.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/reset-password.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `apps/web/tests/authRouteRendering.test.ts`

- [ ] **Step 1: Update auth translations**

Add these keys under `auth` in both locale files:

```json
"passwordStrength": "密码合格度",
"passwordRequirementLength": "长度在 {min}-{max} 个字符之间",
"passwordRequirementLowercase": "包含小写字母",
"passwordRequirementUppercase": "包含大写字母",
"passwordRequirementNumber": "包含数字",
"passwordRequirementSymbol": "包含符号",
"passwordPolicySummary": "建议使用较长短句；当前规则要求 {min}-{max} 个字符。"
```

English equivalents:

```json
"passwordStrength": "Password readiness",
"passwordRequirementLength": "Between {min} and {max} characters",
"passwordRequirementLowercase": "Contains a lowercase letter",
"passwordRequirementUppercase": "Contains an uppercase letter",
"passwordRequirementNumber": "Contains a number",
"passwordRequirementSymbol": "Contains a symbol",
"passwordPolicySummary": "Use a longer phrase; current policy requires {min}-{max} characters."
```

- [ ] **Step 2: Modify registration page**

Import `passwordPolicyRequirements` and `passwordPolicyProgress` from `useWebOptions`, destructure `passwordPolicy`, compute requirement rows, and replace the fixed hint with a progress block.

Use this shape below the password input:

```vue
<div id="password-hint" class="auth-password-policy">
  <div class="auth-password-policy__header">
    <span>{{ t('auth.passwordStrength') }}</span>
    <span>{{ passwordProgress }}%</span>
  </div>
  <div class="auth-password-policy__bar" aria-hidden="true">
    <span :style="{ width: `${passwordProgress}%` }" />
  </div>
  <p class="auth-field-hint">
    {{ t('auth.passwordPolicySummary', { min: passwordPolicy.minLength, max: passwordPolicy.maxLength }) }}
  </p>
  <ul class="auth-password-policy__list">
    <li v-for="item in passwordRequirementRows" :key="item.key" :class="{ 'is-met': item.met }">
      <UIcon :name="item.met ? 'i-lucide-check' : 'i-lucide-circle'" />
      <span>{{ item.label }}</span>
    </li>
  </ul>
</div>
```

Add scoped CSS using existing auth colors and stable dimensions.

- [ ] **Step 3: Modify reset password page**

Use the same helpers for:

```ts
const { passwordPolicy } = useWebOptions()
const passwordRequirementRows = computed(...)
const passwordProgress = computed(...)
const newPasswordMeetsPolicy = computed(() => passwordPolicyRequirements(newPassword.value, passwordPolicy.value).every(item => item.met))
const canSubmit = computed(() => token.value !== '' && newPasswordMeetsPolicy.value && passwordsMatch.value && !submitting.value)
```

Render the same compact policy block below the new-password input.

- [ ] **Step 4: Run frontend route tests**

Run:

```bash
bun test tests/useWebOptions.test.ts tests/authRouteRendering.test.ts
```

Expected: pass.

---

### Task 6: Admin Settings Controls

**Files:**
- Modify: `apps/web/app/pages/admin/settings/index.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add admin translations**

Add under `admin.settings.basic`:

```json
"accountSecurityTitle": "账号安全",
"accountSecurityDescription": "密码策略会影响新注册和密码重置；推荐默认值优先鼓励长密码短句。",
"passwordMinLength": "最小长度",
"passwordMaxLength": "最大长度",
"passwordRequireLowercase": "要求小写字母",
"passwordRequireUppercase": "要求大写字母",
"passwordRequireNumber": "要求数字",
"passwordRequireSymbol": "要求符号",
"passwordRecommended": "推荐默认值：12-128 个字符，不强制字符组合。",
"restorePasswordDefaults": "恢复密码推荐默认值"
```

Add English equivalents with the same keys.

- [ ] **Step 2: Extend settings form state**

Import `recommendedPasswordPolicy` and add form fields:

```ts
passwordMinLength: 12,
passwordMaxLength: 128,
passwordRequireLowercase: false,
passwordRequireUppercase: false,
passwordRequireNumber: false,
passwordRequireSymbol: false
```

Read/write these values from `adminOptionsMap` keys `identity.password.*`.

- [ ] **Step 3: Include password policy in basic save and dirty state**

Extend `hasBasicChanges` and `saveBasicSettings()` payload with:

```ts
{ name: 'identity.password.min_length', value: String(form.passwordMinLength) },
{ name: 'identity.password.max_length', value: String(form.passwordMaxLength) },
{ name: 'identity.password.require_lowercase', value: enabledOptionValue(form.passwordRequireLowercase) },
{ name: 'identity.password.require_uppercase', value: enabledOptionValue(form.passwordRequireUppercase) },
{ name: 'identity.password.require_number', value: enabledOptionValue(form.passwordRequireNumber) },
{ name: 'identity.password.require_symbol', value: enabledOptionValue(form.passwordRequireSymbol) }
```

- [ ] **Step 4: Add account security controls to the basic tab**

Add a bordered section below locale controls with numeric inputs and checkbox/toggle cards. Add a button:

```vue
<UButton type="button" color="neutral" variant="outline" leading-icon="i-lucide-rotate-ccw" @click="restoreRecommendedPasswordPolicy">
  {{ t('admin.settings.basic.restorePasswordDefaults') }}
</UButton>
```

Implement `restoreRecommendedPasswordPolicy()` by copying `recommendedPasswordPolicy` values into `form` and showing a neutral toast.

- [ ] **Step 5: Run admin/settings tests**

Run:

```bash
bun test tests/useWebOptions.test.ts tests/adminOverview.test.ts
```

Expected: pass.

---

### Task 7: OpenAPI And Knowledge Base

**Files:**
- Modify: `contracts/openapi/schemas/options.yaml`
- Modify: `contracts/openapi/paths/identity.yaml`
- Modify: `knowledge/modules/identity.md`
- Modify: `knowledge/modules/options.md`
- Add: `knowledge/sessions/2026-07-08-password-policy-settings.md`

- [ ] **Step 1: Update OpenAPI option enums**

Add the six `identity.password.*` option names wherever option names are enumerated in `contracts/openapi/schemas/options.yaml`.

- [ ] **Step 2: Update identity examples**

In `contracts/openapi/paths/identity.yaml`, replace hard-coded password validation example text with wording that mentions the configured password policy.

- [ ] **Step 3: Validate OpenAPI refs**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: pass.

- [ ] **Step 4: Update knowledge base**

Add concise notes:

- `knowledge/modules/identity.md`: password policy is runtime configurable and shared by registration/reset.
- `knowledge/modules/options.md`: list new public `identity.password.*` options and validation ranges.
- `knowledge/sessions/2026-07-08-password-policy-settings.md`: changed files, decisions, next steps, open questions.

---

### Task 8: Full Verification

**Files:**
- No code edits expected.

- [ ] **Step 1: Run backend focused tests**

Run:

```bash
go test ./app/Models/Identity ./app/Models/Options ./app/Http/Controllers/Identity ./app/Providers -count=1
```

Expected: pass.

- [ ] **Step 2: Run frontend focused tests**

Run:

```bash
bun test tests/useWebOptions.test.ts tests/authRouteRendering.test.ts
```

Expected: pass.

- [ ] **Step 3: Run contract validation**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: pass.

- [ ] **Step 4: Run project test script**

Run:

```bash
./scripts/test.sh
```

Expected: pass. If it fails for an unrelated dirty-worktree issue, record the failing command and the likely owner without reverting user changes.
