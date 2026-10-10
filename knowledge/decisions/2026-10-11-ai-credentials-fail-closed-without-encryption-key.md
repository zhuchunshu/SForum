# Decision: AI Provider Credentials Fail Closed Without An Encryption Key

## Status

Accepted

## Date

2026-10-11

## Context

`APP_OPTION_ENC_KEY` 缺省为空，`NewOptionCipher("")` 返回透明 cipher，Secret Store
在 `AllowTransparent` 下会把明文原样落库。控制台 `GET /admin/ai/settings` 因此一直
返回 `encryptionEnabled=false` 并显示「当前部署没有可用的加密密钥」警告——但
`PUT /admin/ai/profiles/:profileId/credential` 并没有检查加密能力，密钥照样写进去了。

结果是界面在撒谎：警告说「保存供应商密钥会失败」，实际保存成功，只是明文存储。
运营者会以为凭证已加密。

## Decision

- `CredentialWriter.Put` 在没有可用加密密钥时返回 `ErrCredentialEncryptionRequired`；
  HTTP 层额外提前返回 422 `ai.encryption_required`。两道检查：写入前检查覆盖所有
  调用路径，HTTP 层提前检查给出明确错误码。
- 前端在 `encryptionEnabled=false` 时禁用密钥输入与保存按钮，并在
  `submitCredential` 内二次拦截；字段旁显示 `credentialBlockedHint` 而不是只在
  失败后弹 toast。
- 警告文案补出 `APP_OPTION_ENC_KEY` 变量名与 `openssl rand -hex 32` 生成命令。
- `scripts/dev.sh` 首次生成 `.env` 时自动补本地密钥，避免开发者被 fail-closed 卡住。
- 透明 cipher（AllowTransparent）只保留给单元测试。

## Consequences

- 未配置密钥的部署无法录入供应商密钥。这是刻意的：明文落库的凭证比「不能配置 AI」
  更危险，因为它看起来是安全的。
- 新增错误码 `ai.encryption_required` 必须同时在
  `Support/Localization/messages.go` 提供 zh-CN / en-US 译文（本地化测试会强制）。
- 开发环境只需一次 `openssl rand -hex 32`；`dev.sh` 已自动化。

## Revisit when

若将来引入密钥轮换或 KMS 托管密钥，`encryptionEnabled` 的来源应扩展为「任意可用
的加密后端」，而不是单一 env 变量。
