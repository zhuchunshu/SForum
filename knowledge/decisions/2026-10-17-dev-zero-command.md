# 决策：dev 零命令自动激活（watcher 入 host，无登录无授权交互）

日期：2026-10-17。取代本文件链上此前「凭据缓存/一次性登录」的折中方向。

## 操作者拍板（原话级语义）

运行 `./scripts/dev-plugin-loop.sh <pkg>`（以及任何 agent 会话）不经过任何验证：
无登录、无密码、无确认，保存即生效；**包括首次安装**——即使数据库里 0 个管理员
这条链也要工作。第二轮升级为 **B 方案：彻底 0 命令**——watcher 从 CLI 挪进 host
dev 装配，host 启动即自动生效，连 wrapper 本身也删除。

## 实现

- `app/Models/Extensions/dev_auto_runtime.go`：`DevAutoRuntime`——重扫外部源 →
  对每个包：精确保制品 grant → installed 自动启用 / enabled 且有 staged 自动
  upgrade（disabled 尊重操作者状态）。失败只记日志，不影响其余包。
- `app/Support/ExtensionPackage/dev_build.go`：host 进程内构建管线
  （go build backend/plugin + digest 写回含 SRI 重算），与 CLI 共用。
- `bootstrap/dev_plugin_watch.go`：dev 装配 fsnotify watcher（去抖 600ms，
  排除 dist/node_modules/.git 等产物目录），启动即对账一次。
- 合成身份：`dev-auto` 系统账号（无密码凭据不可登录；lifecycle V2 authority
  与 audit 需要合法 UserID，故不用 ID=0）。管理端可见，禁用/删除即关闭通道。
- CLI 侧：wrapper 删除；`dev_trust.go` 的登录提示/凭据缓存/PAT 层全部移除，
  回落 env-only（dev-trust CLI 变成遗留辅助路径）。

## 拒绝过的方向（留档）

- 保留门 + 凭据缓存/浏览器导入：被操作者否掉（「避免人工去授权」≠ 把登录做得
  更顺手）。
- dev 无认证 admin 路由：会撕开第二条激活路径；本轮实际未采用任何新 HTTP 面——
  自动化全在 host 进程内（watcher→runtime→lifecycle 内部调用），路由/守卫零新增。

## 边界（如实）

- dev 机器上「能写 dev 来源根的进程」等效宿主全权执行体（含任意包安装、路由
  注册、own_schema DB grant）。这是操作者明确接受的语义。
- 生产不受影响：dev watcher/DevAutoRuntime 仅 development 非 Safe Mode 装配；
  所有 HTTP 激活路由的 super_admin 门原样。
- CLI `extension dev --auto-trust/--auto-upgrade` 成为遗留路径（无凭据时提示
  指向 host 自动化）。
