package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type ExactPluginGuardRuntime interface {
	InspectRuntimeInstance(extensionsruntime.RuntimeInstanceIdentity) (extensionsruntime.RuntimeInstanceSnapshot, error)
	AcquireRuntimeCall(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.RuntimeCallClass) (*extensionsruntime.RuntimeAdmissionLease, error)
	InvokeGuardInstance(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.ProtocolV2GuardRequest) error
}

type RuntimePluginRouteGuardEvaluator struct {
	runtime ExactPluginGuardRuntime
	policy  ExtensionGuardPolicy
}

func NewRuntimePluginRouteGuardEvaluator(
	runtime ExactPluginGuardRuntime,
	policy ExtensionGuardPolicy,
) *RuntimePluginRouteGuardEvaluator {
	return &RuntimePluginRouteGuardEvaluator{runtime: runtime, policy: policy}
}

func (e *RuntimePluginRouteGuardEvaluator) EvaluatePluginGuard(
	ctx context.Context,
	evaluation routes.PluginGuardEvaluation,
) error {
	if e == nil || e.runtime == nil || e.policy == nil || ctx == nil {
		return preInvocationPluginGuardFailure(nil)
	}
	step := evaluation.Step
	artifact := step.Provider.Artifact
	wireAuthority, err := protocolV2RequestAuthority(evaluation.Authority)
	if err != nil {
		return preInvocationPluginGuardFailure(err)
	}
	rawAuthority := evaluation.Authority.Mode == routes.RequestAuthorityRaw
	lookup, ok := e.policy.Lookup(artifact.ExtensionID)
	if !ok || lookup.Revision == 0 || !lookup.Found || lookup.SafeMode || lookup.Entry.ExtensionID != artifact.ExtensionID ||
		lookup.Entry.ExtensionType != extensions.TypePlugin || lookup.Entry.Status != extensions.StatusEnabled ||
		lookup.Entry.Version != artifact.ExtensionVersion || lookup.Entry.PackageDigest != artifact.PackageDigest ||
		(lookup.Entry.CurrentTrustRequired || rawAuthority) && !lookup.Entry.CurrentArtifactTrusted {
		return preInvocationPluginGuardFailure(nil)
	}
	// core.guard.raw_request confirms that this exact trusted handler owns policy.
	// Credential forwarding remains closed until raw-session authority is frozen.
	if step.PluginGuard.ID == "" {
		return nil
	}
	identity := extensionsruntime.RuntimeInstanceIdentity{
		ExtensionID: artifact.ExtensionID, InstanceID: artifact.RuntimeInstanceID,
	}
	snapshot, err := e.runtime.InspectRuntimeInstance(identity)
	if err != nil || !snapshot.Active || snapshot.Identity != identity ||
		snapshot.ExtensionVersion != artifact.ExtensionVersion || snapshot.ArtifactDigest != artifact.PackageDigest {
		return preInvocationPluginGuardFailure(err)
	}
	lease, err := e.runtime.AcquireRuntimeCall(ctx, identity, extensionsruntime.RuntimeCallGuard)
	if err != nil {
		return preInvocationPluginGuardFailure(err)
	}
	defer lease.Release()
	query, queryValues, err := exactProtocolV2RouteQuery(evaluation.Request.Query)
	if err != nil {
		return preInvocationPluginGuardFailure(err)
	}
	body, present, err := exactProtocolV2RouteBody(evaluation.Request.Body, step.RequestSchema, "request")
	if err != nil {
		return preInvocationPluginGuardFailure(err)
	}
	headers := make(stdhttp.Header)
	if err := copyRouteRequestHeaders(headers, evaluation.Request.Headers, evaluation.Authority); err != nil {
		return preInvocationPluginGuardFailure(err)
	}
	timeout := 3 * time.Second
	if step.TimeoutMS > 0 {
		timeout = time.Duration(step.TimeoutMS) * time.Millisecond
	}
	err = e.runtime.InvokeGuardInstance(lease.Context, identity, extensionsruntime.ProtocolV2GuardRequest{
		GuardID: step.PluginGuard.ID, GuardContractVersion: step.PluginGuard.ContractVersion,
		RouteID: step.RouteID, RouteContractVersion: step.ContractVersion,
		Method: evaluation.RequestMethod, Path: evaluation.RequestPath, Headers: headers,
		PathParameters: evaluation.Request.Params, QueryParameters: query, QueryParameterValues: queryValues,
		RequestSchema: step.RequestSchema, Body: body, BodyPresent: present,
		Authority: wireAuthority,
		Actor: extensionsruntime.NewProtocolV2RouteActor(
			evaluation.Request.ActorID, evaluation.Request.Authenticated, evaluation.Request.Permissions,
		),
		Timeout: timeout,
	})
	return mappedPluginGuardInvocationFailure(err)
}

func mappedPluginGuardInvocationFailure(err error) error {
	if err == nil {
		return nil
	}
	var callFailure *extensionsruntime.ProtocolV2GuardCallFailure
	if errors.As(err, &callFailure) {
		kind := routes.PluginGuardFailureUnavailable
		switch callFailure.Kind() {
		case extensionsruntime.ProtocolV2GuardFailureDenied:
			kind = routes.PluginGuardFailureDenied
		case extensionsruntime.ProtocolV2GuardFailureCrash:
			kind = routes.PluginGuardFailureCrash
		case extensionsruntime.ProtocolV2GuardFailureTimeout:
			kind = routes.PluginGuardFailureTimeout
		case extensionsruntime.ProtocolV2GuardFailureProtocol:
			kind = routes.PluginGuardFailureProtocol
		case extensionsruntime.ProtocolV2GuardFailureCanceled:
			kind = routes.PluginGuardFailureCanceled
		}
		return routes.NewPluginGuardFailure(kind, callFailure.RuntimeExecutionObserved())
	}
	// Compatibility runtimes may still return the legacy deny sentinel directly.
	if errors.Is(err, extensionsruntime.ErrProtocolV2GuardDenied) {
		return routes.NewPluginGuardFailure(routes.PluginGuardFailureDenied, true)
	}
	return preInvocationPluginGuardFailure(err)
}

func preInvocationPluginGuardFailure(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return routes.NewPluginGuardFailure(routes.PluginGuardFailureTimeout, false)
	case errors.Is(err, context.Canceled):
		return routes.NewPluginGuardFailure(routes.PluginGuardFailureCanceled, false)
	default:
		return routes.NewPluginGuardFailure(routes.PluginGuardFailureUnavailable, false)
	}
}

var _ routes.PluginGuardEvaluator = (*RuntimePluginRouteGuardEvaluator)(nil)
