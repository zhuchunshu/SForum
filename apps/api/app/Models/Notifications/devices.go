package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// 原生推送设备注册（App / 桌面端）。
//
// Core 只拥有归属与生命周期：设备令牌 ↔ 用户 ↔ App 实例。真正的投递（FCM/APNs）
// 属于通知通道 provider 插件的职责，Core 只负责把「该发给谁」记录清楚。
// 令牌本身既不进入 API 响应，也不以明文落库（配置了 Core 密钥时存 AES-GCM 密文）。
var (
	ErrPushDeviceInvalid       = errors.New("notifications: push device invalid")
	ErrPushDeviceNotFound      = errors.New("notifications: push device not found")
	ErrPushDeviceTokenTooLong  = errors.New("notifications: push device token too long")
	ErrPushDevicePlatformKnown = errors.New("notifications: push device platform unknown")
)

const (
	pushDeviceIDMinRunes    = 8
	pushDeviceIDMaxRunes    = 128
	pushDeviceTokenMinRunes = 16
	pushDeviceTokenMaxRunes = 4096
	pushDeviceAppVersionMax = 32
	pushDeviceLocaleMax     = 16
	pushDeviceNameMax       = 64

	PushDeviceStatusActive  = "active"
	PushDeviceStatusRevoked = "revoked"
)

// 支持的平台集合。web 走 Web Push 订阅（web_push 通道），不进入设备注册表。
var pushDevicePlatforms = []string{"ios", "android", "desktop"}

var pushDeviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

// PushDevice 是设备注册的公开视图：永不含令牌或令牌指纹。
type PushDevice struct {
	ID         int64      `json:"id"`
	DeviceID   string     `json:"deviceId"`
	Platform   string     `json:"platform"`
	AppVersion string     `json:"appVersion,omitempty"`
	Locale     string     `json:"locale,omitempty"`
	DeviceName string     `json:"deviceName,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type RegisterPushDeviceInput struct {
	UserID     int64
	DeviceID   string
	Platform   string
	Token      string
	AppVersion string
	Locale     string
	DeviceName string
}

// PushDeviceCipher 由 Core 密钥实现（app/Support/Crypto.OptionCipher）。
// 为 nil 时按开发环境约定真存明文；生产装配必须有密钥。
type PushDeviceCipher interface {
	Encrypt(string) (string, error)
}

// PushDeviceStore 是控制器依赖的最小能力面。
type PushDeviceStore interface {
	RegisterPushDevice(context.Context, RegisterPushDeviceInput) (PushDevice, error)
	ListPushDevices(context.Context, int64, bool) ([]PushDevice, error)
	RevokePushDevice(context.Context, int64, string) error
}

// WithPushDeviceCipher 在装配阶段注入 Core 密钥（nil 保持明文，开发环境兼容）。
func (s *PostgresStore) WithPushDeviceCipher(cipher PushDeviceCipher) *PostgresStore {
	if s != nil {
		s.pushDeviceCipher = cipher
	}
	return s
}

// WithPushDeviceClock 便于测试注入时间源。
func (s *PostgresStore) WithPushDeviceClock(now func() time.Time) *PostgresStore {
	if s != nil && now != nil {
		s.pushDeviceNow = now
	}
	return s
}

// NormalizePushDeviceInput 是设备注册的领域输入校验：控制器在调用 store 前必须先过这里，
// store 内再调用一次做纵深防御（同一实现，不是两套规则）。
func NormalizePushDeviceInput(input RegisterPushDeviceInput) (RegisterPushDeviceInput, error) {
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Token = strings.TrimSpace(input.Token)
	input.AppVersion = strings.TrimSpace(input.AppVersion)
	input.Locale = strings.TrimSpace(input.Locale)
	input.DeviceName = strings.TrimSpace(input.DeviceName)

	if input.UserID <= 0 || !pushDeviceIDPattern.MatchString(input.DeviceID) {
		return RegisterPushDeviceInput{}, ErrPushDeviceInvalid
	}
	if !slicesContains(pushDevicePlatforms, input.Platform) {
		return RegisterPushDeviceInput{}, ErrPushDevicePlatformKnown
	}
	tokenRunes := utf8.RuneCountInString(input.Token)
	if tokenRunes < pushDeviceTokenMinRunes {
		return RegisterPushDeviceInput{}, ErrPushDeviceInvalid
	}
	// FCM/APNs 令牌是 ASCII；非 ASCII 一律视为客户端错误，避免超长 Unicode 触发存储放大。
	if tokenRunes > pushDeviceTokenMaxRunes || len(input.Token) > pushDeviceTokenMaxRunes {
		return RegisterPushDeviceInput{}, ErrPushDeviceTokenTooLong
	}
	if !isPrintablePushToken(input.Token) {
		return RegisterPushDeviceInput{}, ErrPushDeviceInvalid
	}
	if utf8.RuneCountInString(input.AppVersion) > pushDeviceAppVersionMax ||
		utf8.RuneCountInString(input.Locale) > pushDeviceLocaleMax ||
		utf8.RuneCountInString(input.DeviceName) > pushDeviceNameMax {
		return RegisterPushDeviceInput{}, ErrPushDeviceInvalid
	}
	return input, nil
}

func isPrintablePushToken(token string) bool {
	for _, r := range token {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}

func slicesContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hashPushDeviceToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (s *PostgresStore) now() time.Time {
	if s != nil && s.pushDeviceNow != nil {
		return s.pushDeviceNow()
	}
	return time.Now().UTC()
}

// RegisterPushDevice 幂等注册/刷新设备。
//
// 语义：同一 (platform, token_hash) 只保留一行，重新注册会改绑到当前用户并清除撤销态
// （换账号登录、令牌轮换、卸载重装都走这条路径）；同一用户同一 device_id 的旧令牌行
// 会被标记撤销，避免向已失效令牌投递。
func (s *PostgresStore) RegisterPushDevice(ctx context.Context, input RegisterPushDeviceInput) (PushDevice, error) {
	normalized, err := NormalizePushDeviceInput(input)
	if err != nil {
		return PushDevice{}, err
	}
	tokenHash := hashPushDeviceToken(normalized.Token)
	stored, err := s.encryptPushDeviceToken(normalized.Token)
	if err != nil {
		return PushDevice{}, err
	}

	// 先淘汰同一安装实例上的其它活跃令牌，再做 upsert，避免新行随后被撤销。
	if _, err := s.runner.Exec(ctx, `
		UPDATE push_devices SET status='revoked', revoked_at=now(), updated_at=now()
		WHERE user_id=$1 AND device_id=$2 AND token_hash<>$3 AND status='active'`,
		normalized.UserID, normalized.DeviceID, tokenHash); err != nil {
		return PushDevice{}, err
	}

	var item PushDevice
	err = s.runner.QueryRow(ctx, `
		INSERT INTO push_devices
		  (user_id, device_id, platform, token_hash, token_ciphertext, app_version, locale, device_name)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT ON CONSTRAINT push_devices_platform_token_key DO UPDATE SET
		  user_id=EXCLUDED.user_id,
		  device_id=EXCLUDED.device_id,
		  token_ciphertext=EXCLUDED.token_ciphertext,
		  app_version=EXCLUDED.app_version,
		  locale=EXCLUDED.locale,
		  device_name=EXCLUDED.device_name,
		  status='active',
		  revoked_at=NULL,
		  updated_at=now(),
		  last_seen_at=now()
		RETURNING id,device_id,platform,app_version,locale,device_name,status,created_at,updated_at,last_seen_at,revoked_at`,
		normalized.UserID, normalized.DeviceID, normalized.Platform, tokenHash, stored,
		normalized.AppVersion, normalized.Locale, normalized.DeviceName,
	).Scan(&item.ID, &item.DeviceID, &item.Platform, &item.AppVersion, &item.Locale, &item.DeviceName,
		&item.Status, &item.CreatedAt, &item.UpdatedAt, &item.LastSeenAt, &item.RevokedAt)
	if err != nil {
		return PushDevice{}, err
	}
	return item, nil
}

// ListPushDevices 返回当前用户的设备；默认只列活跃设备。
func (s *PostgresStore) ListPushDevices(ctx context.Context, userID int64, includeRevoked bool) ([]PushDevice, error) {
	if userID <= 0 {
		return nil, ErrPushDeviceInvalid
	}
	rows, err := s.runner.Query(ctx, `
		SELECT id,device_id,platform,app_version,locale,device_name,status,created_at,updated_at,last_seen_at,revoked_at
		FROM push_devices
		WHERE user_id=$1 AND ($2 OR status='active')
		ORDER BY last_seen_at DESC, id DESC`, userID, includeRevoked)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]PushDevice, 0, 8)
	for rows.Next() {
		var item PushDevice
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.Platform, &item.AppVersion, &item.Locale,
			&item.DeviceName, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.LastSeenAt, &item.RevokedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// RevokePushDevice 按客户端已知的稳定 deviceId 撤销，所有权由 user_id 过滤保证。
func (s *PostgresStore) RevokePushDevice(ctx context.Context, userID int64, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if userID <= 0 || deviceID == "" {
		return ErrPushDeviceInvalid
	}
	tag, err := s.runner.Exec(ctx, `
		UPDATE push_devices SET status='revoked', revoked_at=now(), updated_at=now()
		WHERE user_id=$1 AND device_id=$2 AND status='active'`, userID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPushDeviceNotFound
	}
	return nil
}

func (s *PostgresStore) encryptPushDeviceToken(token string) (string, error) {
	if s == nil || s.pushDeviceCipher == nil {
		return token, nil
	}
	encrypted, err := s.pushDeviceCipher.Encrypt(token)
	if err != nil {
		return "", err
	}
	return encrypted, nil
}
