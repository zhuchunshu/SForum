# 密码合格度进度条支持红/黄/主题色分档

## 目标
把当前"只改宽度、颜色恒为主题色"的密码合格度进度条，改成随合格度分段变色：空=中性灰、弱=红、中=黄、强=主题色。强档用 `--sf-accent`（随用户选的主题变化），贴合 SForum 主题个性化理念。

## 颜色分档（确认方案：空/弱/中/强四档）
| 进度 | 级别 | 填充颜色 token |
|------|------|----------------|
| 0%（空输入） | empty | 不强调填充（用 `--sf-muted` 中性灰，避免红色压迫） |
| 1%–50% | weak | `var(--sf-danger)` 红 `#b91c1c` |
| 51%–99% | medium | `var(--sf-warning)` 黄/琥珀 `#b45309` |
| 100% | strong | `var(--sf-accent)` 主题色（随主题变化） |

全部复用项目现有 token（`sforum-components.css` 中已定义 `--sf-danger`/`--sf-warning`/`--sf-accent`），不引入新色值。百分比文字也跟随同档变色，保持一致；宽度过渡已有 `transition: width 0.18s ease`，颜色新增 `background-color` 过渡。

## 改动文件

### 1. `apps/web/app/composables/useWebOptions.ts` — 新增级别纯函数
在 `passwordPolicyProgress` 之后新增导出函数，集中分档阈值逻辑（便于单测、便于复用）：

```ts
export type PasswordProgressLevel = 'empty' | 'weak' | 'medium' | 'strong'

// 按合格度分数返回语义级别，用于进度条颜色分档：
// 空=中性、弱=红、中=黄、强(100%)=主题色。
export function passwordPolicyProgressLevel(progress: number): PasswordProgressLevel {
  if (progress >= 100) return 'strong'
  if (progress >= 51) return 'medium'
  if (progress >= 1) return 'weak'
  return 'empty'
}
```

阈值取 51（而非 50）确保 50% 仍属"弱"——半数未达，红色提醒合理；51% 起才算"中"。此函数依赖 `passwordPolicyProgress` 的输出，与计分逻辑解耦。

### 2. `extensions/.../register.vue` — markup + script + CSS
- **script**（约 110 行附近）：增加
  ```ts
  const passwordProgressLevel = computed(() => passwordPolicyProgressLevel(passwordProgress.value, passwordPolicy.value))
  ```
  注意：`passwordPolicyProgressLevel` 接受的是百分比，所以传 `passwordProgress.value`（已算好的 0-100），**不是** `(progress, policy)`。修正上面写法：
  ```ts
  const passwordProgressLevel = computed(() => passwordPolicyProgressLevel(passwordProgress.value))
  ```
- **markup**（约 353-360 行）：给进度条容器和百分比文字加级别修饰类
  ```html
  <div id="password-hint" class="auth-password-policy">
    <div class="auth-password-policy__header">
      <span>{{ t('auth.passwordStrength') }}</span>
      <span :class="['auth-password-policy__value', `is-${passwordProgressLevel}`]">{{ passwordProgress }}%</span>
    </div>
    <div class="auth-password-policy__bar" :class="[`is-${passwordProgressLevel}`]" aria-hidden="true">
      <span :style="{ width: `${passwordProgress}%` }" />
    </div>
    ...
  </div>
  ```
- **CSS**（约 680-717 行）：把 `.auth-password-policy__bar span` 的固定 `background: var(--sf-accent)` 改为按父级 `.is-*` 切换，并加 `transition: background-color 0.18s ease`：
  ```css
  .auth-password-policy__bar span {
    display: block;
    height: 100%;
    min-width: 0;
    border-radius: inherit;
    background: var(--sf-muted);            /* empty 默认中性 */
    transition: width 0.18s ease, background-color 0.18s ease;
  }
  .auth-password-policy__bar.is-weak span   { background: var(--sf-danger); }
  .auth-password-policy__bar.is-medium span { background: var(--sf-warning); }
  .auth-password-policy__bar.is-strong span { background: var(--sf-accent); }
  /* 百分比文字同档变色 */
  .auth-password-policy__value.is-weak   { color: var(--sf-danger); }
  .auth-password-policy__value.is-medium { color: var(--sf-warning); }
  .auth-password-policy__value.is-strong { color: var(--sf-accent); }
  ```
  保留 `.auth-password-policy__list li.is-met` 现有逻辑不变（满足项仍用主题色，语义不同：那是"该项已满足"，不是"整体合格度"）。

### 3. `extensions/.../reset-password.vue` — 同样三处改动（前缀 `reset-password-`）
- script（约 27 行后）加 `passwordProgressLevel` computed
- markup（约 116-121 行）加 `auth-password-policy__value` → `reset-password-policy__value` 等同构改动
- CSS（约 194-219 行）加 `.reset-password-policy__bar.is-* span` 与 `.reset-password-policy__value.is-*` 规则

### 4. `apps/web/tests/useWebOptions.test.ts` — 新增级别函数单测
import 增加 `passwordPolicyProgressLevel`，新增测试块覆盖四档边界：
```ts
test('maps password progress to color level', () => {
  expect(passwordPolicyProgressLevel(0)).toBe('empty')
  expect(passwordPolicyProgressLevel(1)).toBe('weak')
  expect(passwordPolicyProgressLevel(50)).toBe('weak')
  expect(passwordPolicyProgressLevel(51)).toBe('medium')
  expect(passwordPolicyProgressLevel(99)).toBe('medium')
  expect(passwordPolicyProgressLevel(100)).toBe('strong')
})
```

### 5. 文档 `knowledge/sessions/2026-07-08-password-progress-granular.md` — 追加分档说明
在现有会话记录的 Decisions / Changed 段补充：进度条新增红/黄/主题色分档，新增 `passwordPolicyProgressLevel` 纯函数及阈值。

## 不做的事
- 不改 `passwordPolicyProgress` 计分逻辑（上一版已做连续长度分，保持稳定）。
- 不改 `.auth-password-policy__list li.is-met` 的颜色（单项满足仍用主题色，语义独立）。
- 不引入 zxcvbn/熵估计（标签是"密码合格度"= 策略符合度，非强度熵，沿用上次决策）。
- 不新增独立组件——两处页面内联改动小，暂不抽取；如未来出现第三个使用点再抽 `PasswordStrengthMeter.vue`。

## 验证
- `bun test apps/web/tests/useWebOptions.test.ts`（含新级别单测）
- `bun test apps/web/tests/authRouteRendering.test.ts`（确认页面仍含 `passwordPolicyProgress` 等关键字，未破坏）
- `bun run typecheck`（新增类型 `PasswordProgressLevel` 导出正确）
- 浏览器在 `http://localhost:3000/register`：空输入=灰、输 `phrase`(50%)=红、接近完成=黄、完全满足=主题色；切换主题（default/ocean_blue/violet/rose/amber）时强档颜色随之变化。