-- +goose Up
-- V3 P4：Host safety gate 与插件 hook 共用 stable step/attempt/lease ledger，但必须使用
-- 独立 action identity，不能伪装成任一插件 lifecycle action。
ALTER TABLE extension_lifecycle_steps
  DROP CONSTRAINT extension_lifecycle_steps_lifecycle_action_check;

ALTER TABLE extension_lifecycle_steps
  ADD CONSTRAINT extension_lifecycle_steps_lifecycle_action_check
  CHECK (lifecycle_action IN (
    'install.plan', 'install', 'enable', 'disable',
    'upgrade.plan', 'upgrade.before', 'upgrade.after', 'rollback',
    'uninstall.plan', 'uninstall', 'uninstall.after', 'host.gate'
  ));

-- +goose Down
-- 已执行的 Host gate 是 retained lifecycle/audit history，Down 不得删除或改写它们。
-- NOT VALID 保留已有 host.gate rows，同时恢复旧代码的新写入边界。
ALTER TABLE extension_lifecycle_steps
  DROP CONSTRAINT extension_lifecycle_steps_lifecycle_action_check;

ALTER TABLE extension_lifecycle_steps
  ADD CONSTRAINT extension_lifecycle_steps_lifecycle_action_check
  CHECK (lifecycle_action IN (
    'install.plan', 'install', 'enable', 'disable',
    'upgrade.plan', 'upgrade.before', 'upgrade.after', 'rollback',
    'uninstall.plan', 'uninstall', 'uninstall.after'
  )) NOT VALID;
