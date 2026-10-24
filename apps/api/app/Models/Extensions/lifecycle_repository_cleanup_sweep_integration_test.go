package extensions

import (
	"fmt"
	"strings"
	"testing"
)

// TestPostgresLifecycleRepositoryPendingUninstallCleanups 验证 sweep 查询只返回
// 「台账已 succeeded 且 tombstone 仍 pending」的卸载清理条目，并支持按扩展过滤。
func TestPostgresLifecycleRepositoryPendingUninstallCleanups(t *testing.T) {
	ctx, pool, repository, extensionID := newLifecycleRepositoryIntegration(t)
	packageDigest := strings.Repeat("a", 64)
	var operationID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_lifecycle_operations (
			extension_id, extension_version, package_digest, artifact_digests,
			operation, state, plan_version, idempotency_key, request_fingerprint,
			authority_type, authority_snapshot, requested_by_user_id, removal_mode,
			terminal_result, started_at, completed_at
		) VALUES (
			$1, '1.0.0', $2, '{}'::jsonb,
			'uninstall', 'uninstalling', 'sweep.integration@1', $3, repeat('a', 64),
			'builtin', '{}'::jsonb, NULL, 'preserve',
			'succeeded', statement_timestamp(), statement_timestamp()
		)
		RETURNING id
	`, extensionID, packageDigest, "sweep-integration-key").Scan(&operationID); err != nil {
		t.Fatal(err)
	}
	var versionID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO extension_versions (extension_id, version, manifest, package_path, package_digest)
		VALUES ($1, '1.0.0', $2::jsonb, 'packages/retained', $3)
		RETURNING id
	`, extensionID, fmt.Sprintf(`{"manifestVersion": 3, "id": %q}`, extensionID), packageDigest).Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM extension_lifecycle_cleanup_records WHERE operation_id = $1`, operationID)
	})
	insertTombstone := func(t *testing.T, cleanupID string, status string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO extension_lifecycle_cleanup_records (
				cleanup_id, operation_id, operation, step_id, position,
				first_attempt, last_attempt, cleanup_mode, record_kind, status,
				retained_extension_id, retained_extension_version, retained_package_digest,
				retained_version_id, retained_runtime_instance_id, retained_package_path,
				identity_snapshot, package_snapshot, runtime_recovery_snapshot,
				target_extension_id, target_extension_version, target_package_digest,
				target_version_id, target_runtime_instance_id, target_package_path,
				retention_marker
			) VALUES (
				$1, $2, 'uninstall', 'lifecycle.uninstall.06.host.uninstalling', 6,
				1, 1, 'uninstall_preserve', 'uninstall_tombstone', $3,
				$4, '1.0.0', $5,
				$6, 'runtime-old', 'packages/retained',				'{"extension":"retained"}'::jsonb,
				'{"package":"retained"}'::jsonb,
				'{"runtime":"retained"}'::jsonb,
				$4, '1.0.0', $5,
				$6, '', 'packages/retained',
				'sweep-integration'
			)
		`, cleanupID, operationID, status, extensionID, packageDigest, versionID); err != nil {
			t.Fatal(err)
		}
	}
	insertTombstone(t, "sweep-integration-pending", "pending")

	items, err := repository.PendingUninstallCleanups(ctx, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.OperationID == operationID {
			found = true
			if item.ExtensionID != extensionID || item.RemovalMode != LifecycleRemovalPreserve {
				t.Fatalf("pending cleanup item = %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("pending tombstone missing from sweep result: %#v", items)
	}

	// 按扩展过滤必须命中同一批条目；其他扩展不返回该条目。
	filtered, err := repository.PendingUninstallCleanups(ctx, extensionID, 10)
	if err != nil || len(filtered) != 1 || filtered[0].OperationID != operationID {
		t.Fatalf("filtered sweep = %#v, err=%v", filtered, err)
	}
	other, err := repository.PendingUninstallCleanups(ctx, "other.extension", 10)
	if err != nil || len(other) != 0 {
		t.Fatalf("other extension sweep = %#v, err=%v", other, err)
	}

	// finalized 后不再返回：把 pending 行按终态要求补齐 finalized 列后翻转状态。
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_cleanup_records SET
			status = 'finalized',
			finalized_at = statement_timestamp(),
			finalized_operation_revision = 1,
			finalized_operation_completed_at = statement_timestamp(),
			purge_receipt_id = 'sweep-integration-receipt',
			purge_proof = '{"schema":"proof"}'::jsonb,
			purge_proof_digest = repeat('a', 64),
			physical_identity_present = FALSE,
			physical_package_present = FALSE,
			physical_runtime_recovery_present = FALSE
		WHERE cleanup_id = 'sweep-integration-pending'
	`); err != nil {
		t.Fatal(err)
	}
	remaining, err := repository.PendingUninstallCleanups(ctx, extensionID, 10)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("finalized tombstone still pending: %#v, err=%v", remaining, err)
	}
}
