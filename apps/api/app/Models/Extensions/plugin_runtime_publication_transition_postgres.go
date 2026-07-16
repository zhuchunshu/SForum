package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PublishPluginRuntimePublicationTransitionTx 在调用方已持有的 pgx.Tx 内，
// 基于最新不可变 full-set 与一次精确生命周期切换插入新的 desired set revision。
//
// 行为约定：
//   - 调用方 journal 事务必须是 READ COMMITTED：advisory lock 在已打开的
//     事务内获取；本 helper 不改隔离级别，也不 Commit / Rollback。
//   - 读取最新 revision 前先拿 pluginRuntimeDesiredSetLock 的 xact advisory lock；
//   - 永不扫描 mutable extensions / extension_versions；
//   - 无历史时按空成员集起步；
//   - 每次未提交的 lifecycle transition 都插入新 immutable publication，
//     即使成员 digest 未变（声明型 enable/disable 也要推进 revision 以唤醒节点）；
//   - journal 重试幂等由调用方在已绑定 revision 时短路，不经过本 helper。
func PublishPluginRuntimePublicationTransitionTx(
	ctx context.Context,
	tx pgx.Tx,
	transition PluginRuntimePublicationTransition,
) (PluginRuntimePublication, error) {
	if ctx == nil || tx == nil {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	// reason/Activate 与制品形状在持锁前校验，避免无意义的锁竞争。
	if err := validatePluginRuntimePublicationTransition(transition); err != nil {
		return PluginRuntimePublication{}, err
	}
	if _, _, err := exactPluginRuntimeTransitionArtifact(transition.Target); err != nil {
		return PluginRuntimePublication{}, err
	}
	if transition.Source != nil {
		if _, _, err := exactPluginRuntimeTransitionArtifact(*transition.Source); err != nil {
			return PluginRuntimePublication{}, err
		}
	}
	latest, err := lockLatestPluginRuntimePublication(ctx, tx)
	var latestMembers []PluginRuntimeMember
	switch {
	case err == nil:
		latestMembers = latest.Members
	case errors.Is(err, ErrPluginRuntimePublicationNotFound):
		// 尚无任何 publication：以空 full-set 作为 CAS 起点。
		latestMembers = nil
	default:
		return PluginRuntimePublication{}, fmt.Errorf("load latest plugin runtime publication: %w", err)
	}

	nextMembers, err := TransitionPluginRuntimeDesiredMembers(latestMembers, transition)
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	// 生命周期 marker 必须推进 revision：即使成员 digest 相同也插入新行，
	// 以便节点 wakeup 后恢复声明型 Registry；重试幂等归 journal 绑定层。
	return insertPluginRuntimePublication(
		ctx, tx, transition.Reason, transition.ActorUserID, nextMembers,
	)
}

// lockLatestPluginRuntimePublication 为所有 desired full-set 生产者固定同一
// 隔离级别、锁顺序与等待后的新快照语义。
func lockLatestPluginRuntimePublication(ctx context.Context, tx pgx.Tx) (PluginRuntimePublication, error) {
	var isolation string
	if err := tx.QueryRow(ctx, `SHOW transaction_isolation`).Scan(&isolation); err != nil {
		return PluginRuntimePublication{}, fmt.Errorf("read plugin runtime transition isolation: %w", err)
	}
	// READ COMMITTED 在等待 xact lock 后会为下一条语句取得新快照；更强的
	// snapshot isolation 反而可能继续读取等待前的旧 full-set。
	if strings.TrimSpace(strings.ToLower(isolation)) != "read committed" {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationConflict
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
	`, pluginRuntimeDesiredSetLock); err != nil {
		return PluginRuntimePublication{}, fmt.Errorf("lock plugin runtime desired set: %w", err)
	}
	return loadPluginRuntimePublication(
		ctx,
		tx,
		pluginRuntimePublicationSelect+` ORDER BY revision DESC LIMIT 1`,
	)
}
