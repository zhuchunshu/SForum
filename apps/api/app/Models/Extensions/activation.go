package extensions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
)

const (
	ActivationTriggerEnable  = "enable"
	ActivationTriggerStartup = "startup"

	ActivationStatusStarting = "starting"
	ActivationStatusHealthy  = "healthy"
	ActivationStatusFailed   = "failed"
	ActivationStatusSkipped  = "skipped"
)

var ErrActivationAttemptNotFound = errors.New("extensions: activation attempt not found")

type ActivationAttempt struct {
	ID               int64      `json:"id"`
	ExtensionID      string     `json:"extensionId"`
	ExtensionVersion string     `json:"extensionVersion"`
	PackageDigest    string     `json:"packageDigest"`
	BootID           string     `json:"bootId"`
	Trigger          string     `json:"trigger"`
	Status           string     `json:"status"`
	ActorUserID      int64      `json:"actorUserId,omitempty"`
	FailureReason    string     `json:"failureReason,omitempty"`
	StartedAt        time.Time  `json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

type ActivationAttemptStore interface {
	LatestActivationAttempt(context.Context, string, string) (ActivationAttempt, error)
	BeginActivationAttempt(context.Context, Extension, string, string, int64) (ActivationAttempt, error)
	CompleteActivationAttempt(context.Context, int64, string, string) error
	RecordSkippedActivation(context.Context, Extension, string, string) error
}

type ActivationCoordinator struct {
	store   ActivationAttemptStore
	auditor audit.Writer
}

func NewActivationCoordinator(store ActivationAttemptStore) *ActivationCoordinator {
	return &ActivationCoordinator{store: store}
}

func (c *ActivationCoordinator) WithAuditor(writer audit.Writer) *ActivationCoordinator {
	c.auditor = writer
	return c
}

func (c *ActivationCoordinator) ShouldSkipStartup(ctx context.Context, extension Extension, bootID string) (bool, error) {
	if c == nil || c.store == nil || extension.Status != StatusEnabled || !isExecutablePlugin(extension) {
		return false, nil
	}
	latest, err := c.store.LatestActivationAttempt(ctx, extension.ID, extension.PackageDigest)
	if errors.Is(err, ErrActivationAttemptNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if latest.Status == ActivationStatusHealthy {
		return false, nil
	}
	if latest.Status == ActivationStatusStarting {
		if err := c.store.CompleteActivationAttempt(ctx, latest.ID, ActivationStatusFailed, "previous_boot_incomplete"); err != nil {
			return false, err
		}
	}
	if err := c.store.RecordSkippedActivation(ctx, extension, normalizedBootID(bootID), "boot_loop_guard"); err != nil {
		return false, err
	}
	c.appendAudit(ctx, audit.ActionExtensionActivationSkipped, extension, "boot_loop_guard")
	return true, nil
}

func (c *ActivationCoordinator) Start(
	ctx context.Context,
	runtime RuntimeManager,
	extension Extension,
	trigger string,
	actorUserID int64,
	bootID string,
) error {
	if runtime == nil || !isExecutablePlugin(extension) {
		return nil
	}
	if c == nil || c.store == nil {
		return runtime.Start(ctx, extension)
	}
	attempt, err := c.store.BeginActivationAttempt(ctx, extension, trigger, normalizedBootID(bootID), actorUserID)
	if err != nil {
		return err
	}
	if err := runtime.Start(ctx, extension); err != nil {
		_ = c.store.CompleteActivationAttempt(ctx, attempt.ID, ActivationStatusFailed, boundedActivationReason(err.Error()))
		c.appendAudit(ctx, audit.ActionExtensionActivationFailed, extension, err.Error())
		return err
	}
	if err := c.store.CompleteActivationAttempt(ctx, attempt.ID, ActivationStatusHealthy, ""); err != nil {
		_ = runtime.Stop(ctx, extension)
		return fmt.Errorf("complete extension activation attempt: %w", err)
	}
	return nil
}

func (c *ActivationCoordinator) appendAudit(ctx context.Context, action string, extension Extension, reason string) {
	if c == nil || c.auditor == nil {
		return
	}
	_ = c.auditor.Append(ctx, audit.Event{Action: action, Metadata: map[string]any{
		"extensionId": extension.ID, "version": extension.Version,
		"packageDigest": extension.PackageDigest, "reason": boundedActivationReason(reason),
	}})
}

func isExecutablePlugin(extension Extension) bool {
	return extension.Type == TypePlugin && strings.TrimSpace(extension.Manifest.Backend.Entry) != ""
}

func NewActivationBootID() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	return fmt.Sprintf("boot-%d", time.Now().UTC().UnixNano())
}

func normalizedBootID(bootID string) string {
	bootID = strings.TrimSpace(bootID)
	if bootID == "" {
		return NewActivationBootID()
	}
	return bootID
}

func boundedActivationReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 2048 {
		return reason[:2048]
	}
	return reason
}
