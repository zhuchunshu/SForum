-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

-- 原生推送设备注册表：Core 只拥有「设备令牌 ↔ 用户 ↔ App 实例」的归属与生命周期，
-- 实际投递（FCM/APNs）由通知通道 provider 插件通过投递记录完成。
-- token 只存 SHA-256（去重/查重）与 Core 密钥加密后的密文（投递时解密），
-- 两个字段都不进入任何 API 响应。
CREATE TABLE push_devices (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  device_id TEXT NOT NULL CHECK (char_length(device_id) BETWEEN 8 AND 128),
  platform TEXT NOT NULL CHECK (platform IN ('ios', 'android', 'desktop')),
  token_hash TEXT NOT NULL CHECK (token_hash ~ '^[a-f0-9]{64}$'),
  token_ciphertext TEXT NOT NULL,
  app_version TEXT NOT NULL DEFAULT '',
  locale TEXT NOT NULL DEFAULT '',
  device_name TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,
  -- 同一物理令牌只能归属一个用户：换账号登录后重新注册即完成改绑。
  CONSTRAINT push_devices_platform_token_key UNIQUE (platform, token_hash)
);
CREATE INDEX push_devices_user_status_idx ON push_devices (user_id, status, id DESC);

-- +goose Down
DROP TABLE push_devices;
