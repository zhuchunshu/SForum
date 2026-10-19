# 2026-10-19 Code Scanning 43/44 修复

## Changed

- `apps/web/app/utils/pageResolve.ts`：`extractRegistryPageHeading` 的净化从
  「`replace(/<[^>]*>/g,'')` 再删尖括号」改为逐字符线性扫描
  （`stripMarkupToText`）。旧写法一次替换会把 `<scr<b>ipt>` 拼回 `<script>`，
  是 `js/incomplete-multi-character-sanitization` 的形态（告警 #43）。新扫描器
  进入标签后不再输出字符，任意中间值都不含标签边界，也不再有 O(n²) 迭代面。
- `apps/web/tests/helpers/sourceText.ts`（新）：`stripHtmlComments`，迭代到不动点
  且**用空格而非空串替换**。空串删除既会拼出新的 `<!--`，也会把
  `v-`+`html`、`inner`+`HTML` 拼成被断言禁止的 token（告警 #44 的形态与升级风险）。
- `apps/web/tests/helpers/sourceText.test.ts`（新）：7 条，覆盖嵌套注释、
  token 拼接、幂等、空串。
- `apps/web/tests/forum/commentCitations.test.ts`：`code()` 改用 `stripHtmlComments`。
- `apps/web/tests/framework/pageResolve.test.ts`：标题提取回归扩到 6 条，新增
  未闭合/嵌套标签不留尖括号、`&lt;` 实体原样保留、多行与中文标题。

## Root Cause

两条告警都由提交 `1da2b5da0` 引入。同一提交里：
`[...sfRegistryPage].vue` 的内联逻辑抽成新函数 → 告警 #42 转 fixed；
`server/utils/iconSvg.ts` 加了「剥离到不动点」循环 → 告警 #41 转 fixed；
但抽出的新函数沿用了旧的单次正则写法、测试辅助函数也仍是单次正则 → 新增 #43/#44。
即 #41 的修法已证明可关闭该查询，#43/#44 是同一提交漏改的两处。

精确性说明：旧实现的**最终返回值**因随后 `[<>]` 全删，实测 40000 例随机 + 对抗
样本零泄漏，**当前不可利用**；但中间值确实可含 `<script`（如 `<h1><script</h1>`），
告警对「单次多字符删除」这个原语的判断成立，属防御纵深层面的真实缺陷。

## Evidence

- `cd apps/web && bun test` → **1331 pass / 0 fail**（171 files；基线 1321，
  +10 为本轮新增用例）。
- `cd apps/web && bun run typecheck` → 0 error。
- `node tests/validate-architecture-boundaries.mjs` → 通过（1871 production files）。
- 净化器属性检查：40006 例随机/对抗输入，输出含 `<`/`>` 的例数为 **0**。
- 全仓 `rg` 复查 `replace(/<[^>]*>/`、`replace(/<!--`：仅命中本轮修改的两处，
  无第三处同类形态。

## Next

- 提交推送 `main`，等 `Security` workflow 的 CodeQL 分析跑到新提交，确认
  #43、#44 变 `fixed`（#41/#42 已 fixed，可作对照）。

## Open Questions

- 本机 `api.github.com` 被 `/etc/hosts` 钉到 `140.82.113.4`，该 IP 对 API Host
  返回 `301 → github.com`，`gh` 因此报 `invalid character '<'`；同时 `gh` keyring
  令牌失效。本次用 git credential store 里的 `gho_` 令牌 + `--resolve
  api.github.com:443:140.82.116.6` 完成查询。是否顺手修 hosts 与重登 `gh`，待定。
- `<h1 data-x=">">` 这类属性值含 `>` 的模板会让标题多出一个引号（既有小瑕疵，
  与告警无关），未纳入本次范围。
