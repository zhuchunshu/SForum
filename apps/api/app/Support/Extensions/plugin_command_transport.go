package extensionsruntime

import (
	"context"
	"fmt"
	"strings"
)

func (s *ProtocolStarter) InvokePluginCommand(
	ctx context.Context,
	identity RuntimeInstanceIdentity,
	contract PluginCommandContract,
	input map[string]any,
) (map[string]any, error) {
	if s == nil || ctx == nil {
		return nil, ErrPluginCommandRegistryInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	identity, err := normalizeRuntimeInstanceIdentity(identity)
	if err != nil {
		return nil, err
	}
	var v2 *protocolV2Client
	if err := func() error {
		unlock, err := s.lockExtensionLifecycleContext(ctx, identity.ExtensionID)
		if err != nil {
			return err
		}
		defer unlock()
		instance := s.protocolInstance(identity)
		if instance == nil {
			return protocolInstanceNotFound(identity)
		}
		if instance.protocolVersion != 2 {
			return ErrProtocolInstanceUnsupported
		}
		var ok bool
		v2, ok = instance.protocol.(*protocolV2Client)
		if !ok || v2.identity == nil || v2.identity.GetExtensionId() != identity.ExtensionID ||
			v2.identity.GetInstanceId() != identity.InstanceID ||
			contract.ExtensionID != identity.ExtensionID || contract.InstanceID != identity.InstanceID ||
			contract.ExtensionVersion != instance.extensionVersion || contract.ArtifactDigest != instance.artifactDigest {
			return fmt.Errorf("%w: exact runtime identity drift", ErrPluginCommandRuntimeStale)
		}
		s.mu.Lock()
		s.recordProtocolCallLocked(identity.ExtensionID)
		s.mu.Unlock()
		return nil
	}(); err != nil {
		return nil, err
	}
	return v2.invokePluginCommand(ctx, contract, input)
}

var _ pluginCommandRuntime = (*ProtocolStarter)(nil)

// --- 插件子进程环境构建（自 protocol.go 抽出；职责：进程 env 组装） ---

// 插件子进程环境白名单：不继承 DATABASE_URL / SESSION_* 等宿主密钥。
var pluginEnvAllowlist = map[string]bool{
	"PATH": true, "HOME": true, "TMPDIR": true, "TEMP": true, "TMP": true,
	"LANG": true, "LC_ALL": true, "LC_CTYPE": true, "TZ": true,
	// go-plugin / 测试 helper 需要的变量由 go-plugin 自行注入；宿主仅保留基础运行环境。
	"SFORUM_PLUGIN_HELPER": true,
	// APP_ENV 非密钥；插件用其判断是否允许测试端点覆盖（生产一律官方端点）。
	"APP_ENV": true,
}

// githubAuthEndpointOverrideEnvKeys 仅在非 production 宿主环境可注入插件子进程。
// 生产路径即使宿主误设这些变量也不得透传，避免 OAuth code/secret/token 打到任意端点。
var githubAuthEndpointOverrideEnvKeys = map[string]bool{
	"SFORUM_AUTH_GITHUB_AUTH_URL":  true,
	"SFORUM_AUTH_GITHUB_TOKEN_URL": true,
	"SFORUM_AUTH_GITHUB_API_URL":   true,
}

// 插件子进程 GC 调优默认值。GOGC=50 让小堆插件更积极归还内存；
// GOMEMLIMIT 是软上限——实测插件常驻 RSS 约 25-30MB，20MiB 会贴着 live heap
// 造成 GC 连续抖动，48MiB 留足峰值余量。已部署实例零配置即生效；
// 宿主进程环境里若存在 SFORUM_PLUGIN_GOMEMLIMIT（如 96MiB）则覆盖默认值。
const (
	pluginProcessGOGCDefault           = "GOGC=50"
	pluginProcessGOMEMLIMITDefault     = "GOMEMLIMIT=48MiB"
	pluginProcessGOMEMLIMITOverrideEnv = "SFORUM_PLUGIN_GOMEMLIMIT"
)

// buildPluginProcessEnv 只保留插件白名单环境；production 不透传 fake-GitHub
// 端点覆盖。disablethp 避免 Linux 为每个小型 Go 插件堆分配多余透明大页。
func buildPluginProcessEnv(hostEnv []string) []string {
	production := hostEnvIsProduction(hostEnv)
	out := []string{"GODEBUG=disablethp=1", pluginProcessGOGCDefault}
	if override, ok := lookupHostEnv(hostEnv, pluginProcessGOMEMLIMITOverrideEnv); ok && strings.TrimSpace(override) != "" {
		out = append(out, "GOMEMLIMIT="+strings.TrimSpace(override))
	} else {
		out = append(out, pluginProcessGOMEMLIMITDefault)
	}
	for _, entry := range hostEnv {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if githubAuthEndpointOverrideEnvKeys[key] {
			// 生产边界：拒绝透传测试端点覆盖。
			if production {
				continue
			}
			out = append(out, entry)
			continue
		}
		if pluginEnvAllowlist[key] || strings.HasPrefix(key, "SFORUM_SETTING_") {
			out = append(out, entry)
		}
	}
	return out
}

// lookupHostEnv 从宿主环境切片精确匹配键，返回原始值（不做 trim，
// 与白名单透传语义一致：值原样交给插件进程）。
func lookupHostEnv(hostEnv []string, key string) (string, bool) {
	for _, entry := range hostEnv {
		name, value, ok := strings.Cut(entry, "=")
		if ok && name == key {
			return value, true
		}
	}
	return "", false
}

// hostEnvIsProduction 从宿主环境切片读取 APP_ENV（大小写不敏感）。
func hostEnvIsProduction(hostEnv []string) bool {
	for _, entry := range hostEnv {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if key == "APP_ENV" {
			return strings.EqualFold(strings.TrimSpace(value), "production")
		}
	}
	return false
}

func pluginSettingEnvName(key string) string {
	var value strings.Builder
	value.WriteString("SFORUM_SETTING_")
	for _, char := range strings.ToUpper(key) {
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			value.WriteRune(char)
		} else {
			value.WriteByte('_')
		}
	}
	return value.String()
}
