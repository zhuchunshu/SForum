# 2026-10-22 Dependabot 告警清账：vite 升级 + 陈旧依赖图

## Changed

- `apps/web` 的 `vite`：8.1.5 → **8.3.4**（经 #174 合并）。原本按 8.1.6 修复的计划
  已被更高补丁覆盖，`package.json` 仍保持精确钉版本（不用 `^`）。
  清掉 5 条告警：GHSA-587h-7wmj-h5pm（high）、GHSA-698v-4r3x-3vcv、GHSA-r825-w8mm-5pxw、
  GHSA-rq7h-c2jc-7f22、GHSA-9jrq-w75r-8gcw（patched 均为 8.1.6，8.3.4 已包含）。
- GitHub 侧：dismiss 40 条 composer 告警（reason `not_used`）。仓库内已无任何 composer 清单。

## Why it mattered

45 条 open 告警里只有 5 条是真的。另外 40 条算的是 **2023 年 PHP 时代的 lock**：
`composer.lock` / `app/Plugins/Mail/composer.lock` 在 13 个远程分支、以及 `main` 全历史里都不存在
（`?ref=main` 全 404），但依赖图仍挂着 6 个 PHP 清单 + 3 个已删除的 workflow，
于是 8 月、10 月的新公告继续按旧版本刷告警。

判定依据（可复算）：

- 依赖图里那 6 个 PHP 路径（`composer.json`、`composer.lock`、`app/Plugins/Core|Mail/composer.json`、
  `app/Plugins/Mail/composer.lock`、`public/tabler/libs/tinymce/composer.json`）只存在于 tag 线
  `v2.6.4`…`v2.6.7` / `archive/php-v2-final` 的树。
- 版本指纹：告警 `= 3.15.1`（flysystem）、`= 7.8.0` / `= 7.5.0`（guzzle）、`= 2.6.1`（jmespath）
  与 `v2.6.7`（2023-09-18）的 lock 完全一致。
- `main` 根提交是 2026-07-03（历史重开）；`origin/V3` 也不是 PHP 分支（根提交同日，树是 Go/Nuxt）。

## Decisions

- composer 40 条按 `not_used` dismiss，不追改：代码不存在，改 lock 没有对象。
- vite 保持精确钉版本（不用 `^`）；本次由 dependabot 的 bun 生态 PR 直接升到 8.3.4，
  不再单独做 8.1.6 的补丁跳跃。
- 不再手工手术式编辑 `bun.lock`：`/apps/web` 的 Dependabot 已改为 `bun` 生态，
  它能正确重写 `bun.lock`（此前声明 npm 时每周都以 `misconfigured_tooling` 失败）。

## Next

- 在 Settings → Code security and analysis 把 Dependency graph 关掉再打开，清掉陈旧快照；
  之后确认 `dependencyGraphManifests` 不再含 composer / `app/Plugins` / `public/tabler` 条目。
  若重建后仍残留，提 GitHub Support（附 404 与 tag 指纹证据）。
- 观察下一个公告周期是否还会冒出 composer 新告警；冒出即说明图没清干净。

## Open Questions

- 是否把 CI 的 `bun-version` 从 1.3.14 提到 1.4.x（dependabot 已把 web 镜像推向 bun 1.4.3，
  镜像与 CI 的 bun 版本目前不一致）。
- `secret_scanning` / `secret_scanning_push_protection` 当前 disabled，与本议题无关，值得单独决策。
