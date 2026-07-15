// Package plugin 是 SForum 后端插件的公开 Go SDK（Wave F4.1 / P7 catalogs）。
//
// 第三方插件应只依赖本包与 Host API 环境变量，不要 import
// app/Models 或其它宿主业务内部包。支持面：
//
//   - Serve：HashiCorp go-plugin 协议服务入口
//   - Protocol / Health / Hook / Mail：RPC 契约类型
//   - Host：sforum.host/v1 客户端（Ping、设置、权限、入队、审计等）
//   - 只读目录：事件、能力、贡献点、provider 槽位、核心 schedule
//   - P7 冻结家族：hooks / services / providers / jobs / schedules / commands
//     （目录 + 边界说明；可调用实现见 sdk/plugin/v2）
//
// 包校验与 CI 契约测试见同模块的 contract 辅助函数与
// extensions/fixtures 下的 fixture 插件。
//
// 宿主目录 Markdown 由 GenerateCatalogDocs / WriteCatalogDocs 写出
// （CLI: sforum extension docs generate），输出默认路径
// docs/extensions/catalogs/。
package plugin
