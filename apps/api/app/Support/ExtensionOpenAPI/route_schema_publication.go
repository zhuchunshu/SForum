package extensionopenapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

var (
	ErrRouteSchemaPublicationInvalid = errors.New("extension openapi: invalid route schema publication")
	ErrRouteSchemaRevisionConflict   = errors.New("extension openapi: route schema publication revision conflict")
)

type PublishedRouteSchemaArtifact struct {
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
}

type routeSchemaPublicationSnapshot struct {
	revision  uint64
	catalog   *RouteSchemaCatalog
	artifacts []Artifact
}

// RouteSchemaPublicationSnapshot is detached inspection data plus a private
// restore handle. Only the owner that created it may restore the frozen catalog.
type RouteSchemaPublicationSnapshot struct {
	Revision        uint64                         `json:"revision"`
	CatalogRevision string                         `json:"catalogRevision"`
	Artifacts       []PublishedRouteSchemaArtifact `json:"artifacts"`
	owner           *RouteSchemaPublication
	source          *routeSchemaPublicationSnapshot
}

// PreparedRouteSchemaPublication is compiled off-snapshot and can be published
// only by its owner after the lifecycle transaction reaches its publication fence.
type PreparedRouteSchemaPublication struct {
	owner        *RouteSchemaPublication
	baseRevision uint64
	catalog      *RouteSchemaCatalog
	artifacts    []Artifact
}

func (p *PreparedRouteSchemaPublication) CatalogRevision() string {
	if p == nil || p.catalog == nil {
		return ""
	}
	return p.catalog.Revision()
}

type RouteSchemaPublication struct {
	writeMu  sync.Mutex
	core     []CoreOperation
	snapshot atomic.Pointer[routeSchemaPublicationSnapshot]
}

func NewRouteSchemaPublication(core []CoreOperation) (*RouteSchemaPublication, error) {
	owner := &RouteSchemaPublication{core: append([]CoreOperation(nil), core...)}
	catalog, err := BuildRouteSchemaCatalog(BuildInput{Core: owner.core})
	if err != nil {
		return nil, err
	}
	owner.snapshot.Store(&routeSchemaPublicationSnapshot{catalog: catalog})
	return owner, nil
}

func (p *RouteSchemaPublication) Prepare(artifacts []Artifact) (*PreparedRouteSchemaPublication, error) {
	if p == nil {
		return nil, ErrRouteSchemaPublicationInvalid
	}
	baseRevision := p.Revision()
	frozen, err := cloneRouteSchemaPublicationArtifacts(artifacts)
	if err != nil {
		return nil, err
	}
	catalog, err := BuildRouteSchemaCatalog(BuildInput{Core: append([]CoreOperation(nil), p.core...), Artifacts: frozen})
	if err != nil {
		return nil, err
	}
	return &PreparedRouteSchemaPublication{
		owner: p, baseRevision: baseRevision, catalog: catalog, artifacts: frozen,
	}, nil
}

func (p *RouteSchemaPublication) Publish(artifacts []Artifact) (RouteSchemaPublicationSnapshot, error) {
	prepared, err := p.Prepare(artifacts)
	if err != nil {
		return RouteSchemaPublicationSnapshot{}, err
	}
	if p == nil {
		return RouteSchemaPublicationSnapshot{}, ErrRouteSchemaPublicationInvalid
	}
	return p.PublishPrepared(prepared, prepared.baseRevision)
}

func (p *RouteSchemaPublication) PublishPrepared(
	prepared *PreparedRouteSchemaPublication,
	expectedRevision uint64,
) (RouteSchemaPublicationSnapshot, error) {
	if p == nil || prepared == nil || prepared.owner != p || prepared.catalog == nil {
		return RouteSchemaPublicationSnapshot{}, ErrRouteSchemaPublicationInvalid
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	current := p.loadSnapshot()
	if prepared.baseRevision != expectedRevision || current.revision != expectedRevision {
		return p.publicationSnapshot(current), fmt.Errorf(
			"%w: expected %d, current %d", ErrRouteSchemaRevisionConflict, expectedRevision, current.revision,
		)
	}
	next := &routeSchemaPublicationSnapshot{
		revision: current.revision + 1, catalog: prepared.catalog, artifacts: prepared.artifacts,
	}
	p.snapshot.Store(next)
	return p.publicationSnapshot(next), nil
}

func (p *RouteSchemaPublication) Restore(
	snapshot RouteSchemaPublicationSnapshot,
	expectedRevision uint64,
) (RouteSchemaPublicationSnapshot, error) {
	if p == nil || snapshot.owner != p || snapshot.source == nil || snapshot.source.catalog == nil {
		return RouteSchemaPublicationSnapshot{}, ErrRouteSchemaPublicationInvalid
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	current := p.loadSnapshot()
	if current.revision != expectedRevision {
		return p.publicationSnapshot(current), fmt.Errorf(
			"%w: expected %d, current %d", ErrRouteSchemaRevisionConflict, expectedRevision, current.revision,
		)
	}
	next := &routeSchemaPublicationSnapshot{
		revision: current.revision + 1, catalog: snapshot.source.catalog, artifacts: snapshot.source.artifacts,
	}
	p.snapshot.Store(next)
	return p.publicationSnapshot(next), nil
}

func (p *RouteSchemaPublication) Revision() uint64 {
	if p == nil {
		return 0
	}
	return p.loadSnapshot().revision
}

func (p *RouteSchemaPublication) PublicationSnapshot() RouteSchemaPublicationSnapshot {
	if p == nil {
		return RouteSchemaPublicationSnapshot{}
	}
	return p.publicationSnapshot(p.loadSnapshot())
}

func (p *RouteSchemaPublication) Bindings() []RouteSchemaBinding {
	if p == nil {
		return nil
	}
	return p.loadSnapshot().catalog.Bindings()
}

func (p *RouteSchemaPublication) ValidateRouteSchema(
	ctx context.Context,
	artifact routes.PluginArtifact,
	direction string,
	routeID string,
	method string,
	actualMethod string,
	contractVersion string,
	action string,
	reference string,
	mediaType string,
	responseStatus int,
	payload []byte,
) error {
	if p == nil {
		return ErrRouteSchemaPublicationInvalid
	}
	return p.loadSnapshot().catalog.ValidateRouteSchema(
		ctx, artifact, direction, routeID, method, actualMethod, contractVersion,
		action, reference, mediaType, responseStatus, payload,
	)
}

func (p *RouteSchemaPublication) loadSnapshot() *routeSchemaPublicationSnapshot {
	if snapshot := p.snapshot.Load(); snapshot != nil {
		return snapshot
	}
	return &routeSchemaPublicationSnapshot{}
}

func (p *RouteSchemaPublication) publicationSnapshot(source *routeSchemaPublicationSnapshot) RouteSchemaPublicationSnapshot {
	result := RouteSchemaPublicationSnapshot{Revision: source.revision, owner: p, source: source}
	if source.catalog != nil {
		result.CatalogRevision = source.catalog.Revision()
	}
	result.Artifacts = make([]PublishedRouteSchemaArtifact, len(source.artifacts))
	for index, artifact := range source.artifacts {
		result.Artifacts[index] = PublishedRouteSchemaArtifact{
			ExtensionID: artifact.ExtensionID, ExtensionVersion: artifact.Version, PackageDigest: artifact.PackageDigest,
		}
	}
	return result
}

func cloneRouteSchemaPublicationArtifacts(values []Artifact) ([]Artifact, error) {
	result := make([]Artifact, len(values))
	for index, value := range values {
		manifest, err := json.Marshal(value.Manifest)
		if err != nil {
			return nil, fmt.Errorf("%w: clone manifest: %v", ErrRouteSchemaPublicationInvalid, err)
		}
		result[index] = Artifact{
			Root: value.Root, ExtensionID: value.ExtensionID, Version: value.Version,
			PackageDigest: value.PackageDigest, Policies: append([]RoutePolicy(nil), value.Policies...),
		}
		if err := json.Unmarshal(manifest, &result[index].Manifest); err != nil {
			return nil, fmt.Errorf("%w: clone manifest: %v", ErrRouteSchemaPublicationInvalid, err)
		}
	}
	return result, nil
}
