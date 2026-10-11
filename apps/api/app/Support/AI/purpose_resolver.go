package ai

import (
	"fmt"
	"strings"
)

// 用途解析：把「站点配置 + 用途自身的声明」合成这次调用实际生效的用途参数。
//
// 优先级是固定的：站点按用途覆盖 > 用途声明的默认值。这条顺序不能反过来——
// 用途声明是 Core（或插件）写下的能力边界，站点配置是运营者的选择；运营者只能
// 在一个已声明的用途上做选择，不能凭空发明一个用途。

// ValidatedPurpose 是登记表里的一条声明，并已按站点配置完成覆盖解析。
type ValidatedPurpose struct {
	Declaration              PurposeDeclaration
	ToolsAllowed             []string
	ToolsOverridden          bool
	FailurePosture           string
	FailurePostureOverridden bool
}

// PurposeResolver 是「用途 → 生效参数」的唯一入口。它由网关与其它需要按用途
// 取参数的调用方共享，避免各处各自解释一遍覆盖优先级。
type PurposeResolver struct {
	registry *PurposeRegistry
	settings func() Settings
}

// NewPurposeResolver 构造解析器。registry 为 nil 时所有用途都视为未登记；
// settings 为 nil 时所有覆盖都不生效（等价于「站点未配置」）。
func NewPurposeResolver(registry *PurposeRegistry, settings func() Settings) *PurposeResolver {
	return &PurposeResolver{registry: registry, settings: settings}
}

// Registry 返回底层登记表，供控制台枚举用途。
func (r *PurposeResolver) Registry() *PurposeRegistry {
	if r == nil {
		return nil
	}
	return r.registry
}

// Validate 判定一个用途是否已登记。未登记即错误：调用方不能自定义用途。
func (r *PurposeResolver) Validate(purpose string) error {
	if r == nil || r.registry == nil {
		return nil
	}
	if _, ok := r.registry.Lookup(purpose); !ok {
		return fmt.Errorf("%w: %q", ErrPurposeUnknown, strings.TrimSpace(purpose))
	}
	return nil
}

// Resolve 返回用途的生效参数。用途未登记时返回 ErrPurposeUnknown——调用方
// 必须失败关闭，而不是拿一个零值继续跑。
func (r *PurposeResolver) Resolve(purpose string) (ValidatedPurpose, error) {
	if r == nil || r.registry == nil {
		return ValidatedPurpose{}, fmt.Errorf("%w: no purpose registry is wired", ErrPurposeUnknown)
	}
	declaration, ok := r.registry.Lookup(purpose)
	if !ok {
		return ValidatedPurpose{}, fmt.Errorf("%w: %q", ErrPurposeUnknown, strings.TrimSpace(purpose))
	}
	resolution := ValidatedPurpose{
		Declaration:             declaration,
		ToolsAllowed:            append([]string(nil), declaration.ToolAllowlist...),
		FailurePosture:          declaration.FailurePosture,
	}
	settings := Settings{}
	if r.settings != nil {
		settings = r.settings()
	}
	configured, ok := settings.Purposes[declaration.ID]
	if !ok {
		return resolution, nil
	}
	if allowed, overridden := configured.ToolAllowlist(); overridden {
		resolution.ToolsAllowed = allowed
		resolution.ToolsOverridden = true
	}
	if posture := strings.TrimSpace(configured.FailurePosture); posture != "" && ValidFailurePosture(posture) {
		resolution.FailurePosture = posture
		resolution.FailurePostureOverridden = true
	}
	return resolution, nil
}

// ValidatePurpose 判定一次补全请求的用途是否已登记。没有注入登记表时放行，
// 保持老部署与单元测试的行为不变；装配路径必须注入它。
func (g *Gateway) ValidatePurpose(purpose string) error {
	if g == nil || g.purposes == nil {
		return nil
	}
	if _, ok := g.purposes.Lookup(purpose); !ok {
		return fmt.Errorf("%w: %q", ErrPurposeUnknown, strings.TrimSpace(purpose))
	}
	return nil
}

// Purposes 返回网关注入的用途登记表，供控制台枚举。
func (g *Gateway) Purposes() *PurposeRegistry {
	if g == nil {
		return nil
	}
	return g.purposes
}
