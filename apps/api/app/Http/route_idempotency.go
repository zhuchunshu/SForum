package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime"
	stdhttp "net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	idempotency "github.com/zhuchunshu/sforum/apps/api/app/Support/Idempotency"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

const routeIdempotencyCleanupTimeout = 2 * time.Second

type RequiredRouteIdempotency struct {
	store *idempotency.Store
}

func NewRequiredRouteIdempotency(store *idempotency.Store) *RequiredRouteIdempotency {
	return &RequiredRouteIdempotency{store: store}
}

func (r *RequiredRouteIdempotency) Begin(
	ctx context.Context,
	_ routes.RouteExecutionPlan,
	step routes.RouteExecutionStep,
	policy routes.RouteExecutionPolicy,
	request routes.DispatchRequest,
) (routes.RouteIdempotencyLease, *routes.DispatchResponse, error) {
	if r == nil || r.store == nil || ctx == nil || !policy.IdempotencyRequired ||
		step.Provider.Kind != routes.ProviderPlugin {
		return nil, nil, routes.ErrDispatchIdempotencyUnavailable
	}
	values := request.Headers.Values(idempotency.HeaderName)
	if len(values) != 1 || values[0] == "" {
		return nil, nil, routes.ErrDispatchIdempotencyKeyInvalid
	}
	actorScope, err := routeReplayActorScope(request)
	if err != nil {
		return nil, nil, err
	}
	fingerprint, err := routeReplayFingerprint(request)
	if err != nil {
		return nil, nil, routes.ErrDispatchIdempotencyKeyInvalid
	}
	artifact := step.Provider.Artifact
	lease, replay, err := r.store.BeginRequiredReplay(ctx, idempotency.RequiredReplayScope{
		ActorScope: actorScope, ExtensionID: artifact.ExtensionID,
		ExtensionVersion: artifact.ExtensionVersion, PackageDigest: artifact.PackageDigest,
		RouteID: step.RouteID, ContractVersion: step.ContractVersion, Method: request.Method,
	}, values[0], fingerprint)
	if err != nil {
		switch {
		case errors.Is(err, idempotency.ErrRequiredReplayInvalid):
			return nil, nil, routes.ErrDispatchIdempotencyKeyInvalid
		case errors.Is(err, idempotency.ErrRequiredReplayInProgress):
			return nil, nil, routes.ErrDispatchIdempotencyInProgress
		case errors.Is(err, idempotency.ErrRequiredReplayFingerprintConflict):
			return nil, nil, routes.ErrDispatchIdempotencyConflict
		default:
			return nil, nil, errors.Join(routes.ErrDispatchIdempotencyUnavailable, err)
		}
	}
	if replay != nil {
		headers := replay.Headers.Clone()
		headers.Set(idempotency.ReplayedHeader, "true")
		return nil, &routes.DispatchResponse{
			Status: replay.Status, Headers: headers, Body: append([]byte(nil), replay.Body...),
		}, nil
	}
	return &requiredRouteIdempotencyLease{store: r.store, lease: lease}, nil, nil
}

type requiredRouteIdempotencyLease struct {
	store *idempotency.Store
	lease idempotency.RequiredReplayLease
}

func (l *requiredRouteIdempotencyLease) Complete(ctx context.Context, response routes.DispatchResponse) error {
	if l == nil || l.store == nil {
		return routes.ErrDispatchIdempotencyUnavailable
	}
	cleanupCtx, cancel := routeIdempotencyCleanupContext(ctx)
	defer cancel()
	headers := response.Headers.Clone()
	headers.Del(idempotency.ReplayedHeader)
	return l.store.CompleteRequiredReplay(cleanupCtx, l.lease, idempotency.RequiredReplayResponse{
		Status: response.Status, Headers: headers, Body: append([]byte(nil), response.Body...),
	})
}

func (l *requiredRouteIdempotencyLease) Abort(ctx context.Context) error {
	if l == nil || l.store == nil {
		return routes.ErrDispatchIdempotencyUnavailable
	}
	cleanupCtx, cancel := routeIdempotencyCleanupContext(ctx)
	defer cancel()
	return l.store.AbortRequiredReplay(cleanupCtx, l.lease)
}

func routeIdempotencyCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), routeIdempotencyCleanupTimeout)
}

func routeReplayActorScope(request routes.DispatchRequest) (string, error) {
	if request.Authenticated {
		if request.ActorID <= 0 ||
			request.CredentialSource != routes.DispatchCredentialCookie && request.CredentialSource != routes.DispatchCredentialBearer {
			return "", routes.ErrDispatchIdempotencyUnavailable
		}
		return "actor:" + strconv.FormatInt(request.ActorID, 10) + ":" + string(request.CredentialSource), nil
	}
	if request.ActorID != 0 {
		return "", routes.ErrDispatchIdempotencyUnavailable
	}
	address, err := netip.ParseAddr(strings.TrimSpace(request.ClientIP))
	if err != nil {
		return "", routes.ErrDispatchIdempotencyUnavailable
	}
	digest := sha256.Sum256([]byte(address.Unmap().String()))
	return "anonymous:" + hex.EncodeToString(digest[:]), nil
}

func routeReplayFingerprint(request routes.DispatchRequest) (string, error) {
	query, err := canonicalRouteReplayQuery(request.Query)
	if err != nil {
		return "", err
	}
	contentType, err := canonicalRouteReplayContentType(request.Headers)
	if err != nil {
		return "", err
	}
	document, err := json.Marshal(struct {
		Method      string `json:"method"`
		Path        string `json:"path"`
		Query       string `json:"query"`
		ContentType string `json:"contentType"`
		Body        []byte `json:"body"`
	}{
		Method: request.Method, Path: request.Path, Query: query,
		ContentType: contentType, Body: request.Body,
	})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:]), nil
}

func canonicalRouteReplayQuery(raw string) (string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	for key := range values {
		sort.Strings(values[key])
	}
	return values.Encode(), nil
}

func canonicalRouteReplayContentType(headers stdhttp.Header) (string, error) {
	values := headers.Values("Content-Type")
	if len(values) == 0 {
		return "", nil
	}
	if len(values) != 1 {
		return "", routes.ErrDispatchIdempotencyKeyInvalid
	}
	mediaType, parameters, err := mime.ParseMediaType(values[0])
	if err != nil {
		return "", err
	}
	return mime.FormatMediaType(strings.ToLower(mediaType), parameters), nil
}

var _ routes.RouteIdempotencyController = (*RequiredRouteIdempotency)(nil)
var _ routes.RouteIdempotencyLease = (*requiredRouteIdempotencyLease)(nil)
