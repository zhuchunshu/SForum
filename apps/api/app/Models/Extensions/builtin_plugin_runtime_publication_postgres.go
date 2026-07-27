package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// builtinPluginRuntimeSync 描述一次 inert plugin SaveBuiltin 在已持有
// pluginRuntimeDesiredSetLock 且已推进 active_version 之后，对不可变
// desired full-set 的精确增量。已启用 plugin 的新制品只会暂存，必须由
// lifecycle 统一推进 runtime、identity 和其他 Host registry。
// 无关成员原样保留，缺失成员永不因 SyncBuiltins 复活。
//
// upgrade 的 source 权威是 latest immutable publication 成员，不是
// extensions.active_version_id：旧 SyncBuiltins 可能已把 active 推到 B，
// 而 publication 仍停在 A，必须以 A 为 CAS 源修复到 B。
type builtinPluginRuntimeSync struct {
	Created bool
	Status  string
	Latest  PluginRuntimePublication
	Target  Extension // 同步后 exact 新制品
}

// publishBuiltinPluginRuntimeSync 在已有 publication 时，按 enable/upgrade
// 规则追加至多一条 actorless revision。无 membership 变更时不插入。
// 调用方必须先持有 desired-set xact lock，且扩展行已更新到 Target。
func publishBuiltinPluginRuntimeSync(
	ctx context.Context,
	tx pgx.Tx,
	sync builtinPluginRuntimeSync,
) error {
	if ctx == nil || tx == nil {
		return ErrPluginRuntimePublicationConflict
	}
	targetMember, targetExecutable, err := exactPluginRuntimeTransitionArtifact(sync.Target)
	if err != nil {
		return err
	}

	current, found := pluginRuntimeMemberForExtension(sync.Latest.Members, sync.Target.ID)
	// 最新 immutable 成员已是 exact target：幂等，不追加（即使本事务再次
	// 写了同一 active_version_id）。
	if found && current == targetMember {
		return nil
	}

	var transition PluginRuntimePublicationTransition
	switch {
	case !found:
		// 无成员：仅「首次插入的可执行 builtin」才 enable；已存在行（含
		// trust-revocation 摘除成员后 status 仍 enabled）保持缺席，禁止复活。
		if !sync.Created || !targetExecutable {
			return nil
		}
		transition = PluginRuntimePublicationTransition{
			Target: sync.Target, Activate: true,
			Reason: PluginRuntimePublicationEnable, ActorUserID: 0,
		}
	default:
		// 有成员且不是 target：以 immutable 成员 A 为 upgrade source，
		// 即使 mutable active 已经是 B 也必须 A→B 修复 publication。
		if sync.Status != StatusEnabled {
			return fmt.Errorf(
				"%w: builtin desired member exists while extension is not enabled",
				ErrPluginRuntimePublicationConflict,
			)
		}
		source, loadErr := loadBuiltinPluginRuntimeMemberArtifact(ctx, tx, current)
		if loadErr != nil {
			return loadErr
		}
		sourceMember, _, sourceErr := exactPluginRuntimeTransitionArtifact(source)
		if sourceErr != nil {
			return fmt.Errorf(
				"%w: builtin publication member artifact is invalid",
				ErrPluginRuntimePublicationConflict,
			)
		}
		// 成员命名的 version 行必须与成员元组逐字段一致，否则 fail-closed。
		if sourceMember != current {
			return fmt.Errorf(
				"%w: builtin publication member does not match immutable version row",
				ErrPluginRuntimePublicationConflict,
			)
		}
		// source→target 身份已不同；declaration-only 由 backend.entry 决定
		// 成员增删（upgrade + Activate 且 target 不可执行会移除成员）。
		transition = PluginRuntimePublicationTransition{
			Source: &source, Target: sync.Target, Activate: true,
			Reason: PluginRuntimePublicationUpgrade, ActorUserID: 0,
		}
	}

	next, err := TransitionPluginRuntimeDesiredMembers(sync.Latest.Members, transition)
	if err != nil {
		return err
	}
	_, nextDigest, err := canonicalPluginRuntimeMembers(next)
	if err != nil {
		return err
	}
	// 成员集未变则不推进 revision（与 lifecycle 的“总是插入”不同）。
	if nextDigest == sync.Latest.MembersDigest && len(next) == sync.Latest.MemberCount {
		return nil
	}
	if _, err := insertPluginRuntimePublication(
		ctx, tx, transition.Reason, 0, next,
	); err != nil {
		return fmt.Errorf("insert builtin plugin runtime publication: %w", err)
	}
	return nil
}

// loadBuiltinPluginRuntimeMemberArtifact 按 immutable publication 成员命名的
// 精确 (id, extension_id, version, package_digest) 读取 extension_versions，
// 作为 upgrade CAS 的 source，从不读取 mutable active_version_id。
func loadBuiltinPluginRuntimeMemberArtifact(
	ctx context.Context,
	tx pgx.Tx,
	member PluginRuntimeMember,
) (Extension, error) {
	if !validPluginRuntimeMember(member) {
		return Extension{}, fmt.Errorf(
			"%w: builtin publication member identity is invalid",
			ErrPluginRuntimePublicationConflict,
		)
	}
	var version, digest string
	var versionID int64
	var extensionID string
	var manifestBody []byte
	err := tx.QueryRow(ctx, `
		SELECT id, extension_id, version, package_digest, manifest
		FROM extension_versions
		WHERE id = $1
		  AND extension_id = $2
		  AND version = $3
		  AND package_digest = $4
	`, member.ExtensionVersionID, member.ExtensionID,
		member.ExtensionVersion, member.PackageDigest,
	).Scan(&versionID, &extensionID, &version, &digest, &manifestBody)
	if err != nil {
		return Extension{}, fmt.Errorf(
			"%w: load builtin publication member version: %w",
			ErrPluginRuntimePublicationConflict, err,
		)
	}
	if versionID != member.ExtensionVersionID || extensionID != member.ExtensionID ||
		version != member.ExtensionVersion || digest != member.PackageDigest {
		return Extension{}, fmt.Errorf(
			"%w: builtin publication member version row drifted",
			ErrPluginRuntimePublicationConflict,
		)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		return Extension{}, fmt.Errorf(
			"%w: decode builtin publication member manifest: %w",
			ErrPluginRuntimePublicationConflict, err,
		)
	}
	manifest = extensionmanifest.Normalize(manifest)
	extension := Extension{
		ID: extensionID, Name: strings.TrimSpace(manifest.Name),
		Type: TypePlugin, Status: StatusEnabled,
		Version: version, ActiveVersionID: versionID,
		PackageDigest: digest, Manifest: manifest,
	}
	if _, _, err := exactPluginRuntimeTransitionArtifact(extension); err != nil {
		return Extension{}, fmt.Errorf(
			"%w: builtin publication member artifact is invalid",
			ErrPluginRuntimePublicationConflict,
		)
	}
	return extension, nil
}
