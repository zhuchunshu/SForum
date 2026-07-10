package extensions

import (
	"errors"
	"fmt"
)

type WebReleaseStatus string

const (
	WebReleaseQueued     WebReleaseStatus = "queued"
	WebReleaseResolving  WebReleaseStatus = "resolving"
	WebReleaseInstalling WebReleaseStatus = "installing"
	WebReleaseBuilding   WebReleaseStatus = "building"
	WebReleaseVerifying  WebReleaseStatus = "verifying"
	WebReleaseReady      WebReleaseStatus = "ready"
	WebReleaseActivating WebReleaseStatus = "activating"
	WebReleaseActive     WebReleaseStatus = "active"
	WebReleaseInactive   WebReleaseStatus = "inactive"
	WebReleaseFailed     WebReleaseStatus = "failed"
	WebReleaseSuperseded WebReleaseStatus = "superseded"
	WebReleaseRolledBack WebReleaseStatus = "rolled_back"
)

var ErrInvalidWebReleaseTransition = errors.New("extensions: invalid web release transition")

type TransitionOptions struct {
	Compensation bool
}

func ValidateWebReleaseTransition(current WebReleaseStatus, next WebReleaseStatus) error {
	return ValidateWebReleaseTransitionWithOptions(current, next, TransitionOptions{})
}

func ValidateWebReleaseTransitionWithOptions(current WebReleaseStatus, next WebReleaseStatus, options TransitionOptions) error {
	if !isKnownWebReleaseStatus(current) || !isKnownWebReleaseStatus(next) {
		return invalidWebReleaseTransition(current, next)
	}
	if current == next {
		return nil
	}

	valid := false
	switch current {
	case WebReleaseQueued:
		valid = next == WebReleaseResolving || next == WebReleaseFailed || next == WebReleaseSuperseded
	case WebReleaseResolving:
		valid = next == WebReleaseInstalling || next == WebReleaseFailed || next == WebReleaseSuperseded
	case WebReleaseInstalling:
		valid = next == WebReleaseBuilding || next == WebReleaseFailed || next == WebReleaseSuperseded
	case WebReleaseBuilding:
		valid = next == WebReleaseVerifying || next == WebReleaseFailed || next == WebReleaseSuperseded
	case WebReleaseVerifying:
		valid = next == WebReleaseReady || next == WebReleaseFailed || next == WebReleaseSuperseded
	case WebReleaseReady:
		valid = next == WebReleaseActivating || next == WebReleaseFailed || next == WebReleaseSuperseded
	case WebReleaseActivating:
		valid = next == WebReleaseActive || next == WebReleaseFailed
	case WebReleaseActive:
		// rolled_back 只表示激活失败后的补偿替换，普通发布替换必须进入 inactive。
		valid = next == WebReleaseInactive || (next == WebReleaseRolledBack && options.Compensation)
	}
	if !valid {
		return invalidWebReleaseTransition(current, next)
	}
	return nil
}

func (status WebReleaseStatus) IsFinal() bool {
	switch status {
	case WebReleaseInactive, WebReleaseFailed, WebReleaseSuperseded, WebReleaseRolledBack:
		return true
	default:
		return false
	}
}

func (status WebReleaseStatus) IsLive() bool {
	switch status {
	case WebReleaseQueued,
		WebReleaseResolving,
		WebReleaseInstalling,
		WebReleaseBuilding,
		WebReleaseVerifying,
		WebReleaseReady,
		WebReleaseActivating,
		WebReleaseActive:
		return true
	default:
		return false
	}
}

func isKnownWebReleaseStatus(status WebReleaseStatus) bool {
	return status.IsLive() || status.IsFinal()
}

func invalidWebReleaseTransition(current WebReleaseStatus, next WebReleaseStatus) error {
	return fmt.Errorf("%w: %s -> %s", ErrInvalidWebReleaseTransition, current, next)
}
