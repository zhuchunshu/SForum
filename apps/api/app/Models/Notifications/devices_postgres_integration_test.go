package notifications

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
)

// 设备注册的真实 SQL 语义：改绑、令牌轮换淘汰、按用户隔离、按 deviceId 撤销。
func TestPushDeviceLifecyclePostgres(t *testing.T) {
	ctx, tx := notificationChannelPostgresTx(t)
	firstUser := insertNotificationChannelUser(t, ctx, tx, "device-a")
	secondUser := insertNotificationChannelUser(t, ctx, tx, "device-b")
	store := newPostgresStore(tx)

	deviceID := fmt.Sprintf("install-%d", time.Now().UnixNano())
	token := fmt.Sprintf("token-%d-device-a", time.Now().UnixNano())
	created, err := store.RegisterPushDevice(ctx, RegisterPushDeviceInput{
		UserID: firstUser, DeviceID: deviceID, Platform: "ios", Token: token,
		AppVersion: "1.4.0", Locale: "zh-CN", DeviceName: "iPhone",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if created.Status != PushDeviceStatusActive || created.DeviceID != deviceID {
		t.Fatalf("unexpected device view: %#v", created)
	}

	// 幂等刷新：同一令牌重复注册不产生新行，只更新 last_seen。
	if _, err := store.RegisterPushDevice(ctx, RegisterPushDeviceInput{
		UserID: firstUser, DeviceID: deviceID, Platform: "ios", Token: token, AppVersion: "1.4.1",
	}); err != nil {
		t.Fatalf("re-register: %v", err)
	}
	var rows int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM push_devices WHERE device_id=$1`, deviceID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("re-registration must be idempotent, rows=%d", rows)
	}

	// 令牌轮换：同一安装实例换令牌后，旧行被撤销，只剩一条活跃设备。
	rotated := token + "-rotated"
	if _, err := store.RegisterPushDevice(ctx, RegisterPushDeviceInput{
		UserID: firstUser, DeviceID: deviceID, Platform: "ios", Token: rotated,
	}); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	active, err := store.ListPushDevices(ctx, firstUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 {
		t.Fatalf("expected exactly one active device after rotation, got %d", len(active))
	}

	// 换用户登录后重新注册同一物理安装：改绑到新用户，旧用户不再可见。
	if _, err := store.RegisterPushDevice(ctx, RegisterPushDeviceInput{
		UserID: secondUser, DeviceID: deviceID, Platform: "ios", Token: rotated,
	}); err != nil {
		t.Fatalf("rebind: %v", err)
	}
	firstActive, err := store.ListPushDevices(ctx, firstUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstActive) != 0 {
		t.Fatalf("rebound device must leave the previous user, got %#v", firstActive)
	}

	// 跨用户撤销必须失败：所有权由 user_id 过滤。
	if err := store.RevokePushDevice(ctx, firstUser, deviceID); err != ErrPushDeviceNotFound {
		t.Fatalf("cross-user revoke must not touch the device, got %v", err)
	}
	if err := store.RevokePushDevice(ctx, secondUser, deviceID); err != nil {
		t.Fatalf("owner revoke: %v", err)
	}
	afterRevoke, err := store.ListPushDevices(ctx, secondUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterRevoke) != 0 {
		t.Fatalf("revoked device must not be listed as active: %#v", afterRevoke)
	}
	withRevoked, err := store.ListPushDevices(ctx, secondUser, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(withRevoked) != 1 || withRevoked[0].Status != PushDeviceStatusRevoked || withRevoked[0].RevokedAt == nil {
		t.Fatalf("revoked device must remain visible with includeRevoked: %#v", withRevoked)
	}

	// 重新注册同一令牌应清除撤销态（重装/重新登录）。
	if _, err := store.RegisterPushDevice(ctx, RegisterPushDeviceInput{
		UserID: secondUser, DeviceID: deviceID, Platform: "ios", Token: rotated,
	}); err != nil {
		t.Fatalf("re-register after revoke: %v", err)
	}
	reActivated, err := store.ListPushDevices(ctx, secondUser, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reActivated) != 1 || reActivated[0].Status != PushDeviceStatusActive {
		t.Fatalf("re-registration must clear revocation: %#v", reActivated)
	}
}

// 令牌只以哈希 + 密文落库，明文不出现在任何列里。
func TestPushDeviceStoresTokenHashAndCiphertextOnlyPostgres(t *testing.T) {
	ctx, tx := notificationChannelPostgresTx(t)
	userID := insertNotificationChannelUser(t, ctx, tx, "device-token")
	store := newPostgresStore(tx).WithPushDeviceCipher(stubPushCipher{})

	token := fmt.Sprintf("secret-token-%d", time.Now().UnixNano())
	if _, err := store.RegisterPushDevice(ctx, RegisterPushDeviceInput{
		UserID: userID, DeviceID: fmt.Sprintf("install-%d", time.Now().UnixNano()),
		Platform: "android", Token: token,
	}); err != nil {
		t.Fatal(err)
	}

	var hash, stored string
	if err := tx.QueryRow(ctx, `SELECT token_hash, token_ciphertext FROM push_devices WHERE user_id=$1`, userID).Scan(&hash, &stored); err != nil {
		t.Fatal(err)
	}
	if hash != hashPushDeviceToken(token) {
		t.Fatalf("token_hash mismatch: %q", hash)
	}
	if strings.Contains(stored, token) {
		t.Fatalf("ciphertext must not embed the plaintext token: %q", stored)
	}
	if !strings.HasPrefix(stored, "wrapped:") {
		t.Fatalf("expected cipher-wrapped storage, got %q", stored)
	}
}

// stubPushCipher 用不可逆变换冒充加密：断言密文列里不含明文令牌。
type stubPushCipher struct{}

func (stubPushCipher) Encrypt(value string) (string, error) {
	digest := sha256.Sum256([]byte(value))
	return "wrapped:" + hex.EncodeToString(digest[:]), nil
}
