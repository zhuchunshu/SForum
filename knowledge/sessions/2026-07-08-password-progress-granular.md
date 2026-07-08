# 2026-07-08 Password Progress Granular Feedback

## Changed

- Updated frontend password readiness progress to score length gradually before the minimum length is met.
- Added a regression test for the recommended length-only password policy so short input no longer reports only `0%`.
- Verified the registration page shows `50%` for the six-character test password `phrase`.
- 进度条改为按合格度分档变色：空=中性灰、弱(1-50%)=红、中(51-99%)=黄、强(100%)=主题色。注册页与重置密码页同步更新，百分比文字与进度条同档变色。
- 新增纯函数 `passwordPolicyProgressLevel(progress)` 集中分档阈值（0/1/51/100），并补充边界单测。

## Decisions

- Keep API password policy validation authoritative; frontend progress remains guidance only.
- Do not introduce a password-strength dependency yet because the UI label is policy readiness rather than entropy estimation.
- 强档颜色用 `--sf-accent`（随用户选的主题变化），而非固定绿色，贴合 SForum 主题个性化理念。
- 弱/中档填充色脱离 `--sf-*` token，用亮色阶 `#ef4444`（红）/`#f59e0b`（琥珀）。原因：`--sf-danger #b91c1c` 与 `--sf-warning #b45309` 都偏暗、色相接近（红↔棕橙），放在进度条这种大色块上难辨。百分比文字仍用 `--sf-danger`/`--sf-warning` 暗色阶——小号文字可读性更好，且与亮色填充条形成层次。
- 阈值取 51（非 50）：半数未达仍属"弱"，红色提醒合理；51% 起才算"中"。
- 暂不抽取独立的 `PasswordStrengthMeter.vue` 组件——两处页面内联改动小；如出现第三个使用点再抽。
- 复用现有 `--sf-danger`/`--sf-warning`/`--sf-accent` token，不引入新色值；单项已满足的颜色（`li.is-met`）保持主题色不变，语义独立。

## Verification

- `bun test tests/useWebOptions.test.ts`
- `bun test tests/useWebOptions.test.ts tests/authRouteRendering.test.ts`
- `bun run typecheck`
- Browser check on `http://localhost:3000/register`: entering `phrase` showed `50%` and the progress bar width was `50%`.

## Next

- If the label later changes from policy readiness to password strength, re-evaluate a mature strength-estimation library instead of extending policy scoring.
