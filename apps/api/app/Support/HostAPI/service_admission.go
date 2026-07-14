package hostapi

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrServiceProviderStale                = errors.New("hostapi: service provider runtime is stale")
	ErrServiceProviderDraining             = errors.New("hostapi: service provider runtime is draining")
	ErrServiceProviderAdmissionUnavailable = errors.New("hostapi: service provider admission is unavailable")
)

// ServiceProviderIdentity 固定 registry Winner 对应的 exact runtime。
type ServiceProviderIdentity struct {
	ExtensionID string
	InstanceID  string
}

func (i ServiceProviderIdentity) valid() bool {
	return strings.TrimSpace(i.ExtensionID) != "" && strings.TrimSpace(i.InstanceID) != ""
}

// ServiceProviderAdmissionLease 覆盖一次 unary 调用或完整 provider stream。
type ServiceProviderAdmissionLease interface {
	Context() context.Context
	Release()
}

// ServiceProviderAdmissionLeaseFailure 由 runtime adapter 区分 caller 取消和强制 drain。
type ServiceProviderAdmissionLeaseFailure interface {
	ServiceProviderAdmissionFailure() error
}

type ServiceProviderAdmission interface {
	AcquireServiceProvider(context.Context, ServiceProviderIdentity) (ServiceProviderAdmissionLease, error)
}
