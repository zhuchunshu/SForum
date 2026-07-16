package extensions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type pluginRuntimeCoordinatorTestRepository struct {
	mu sync.Mutex

	publications map[int64]PluginRuntimePublication
	latest       int64
	node         PluginRuntimeNode
	registered   bool
	acks         map[int64]PluginRuntimePublicationAck
	applied      map[int64][]PluginRuntimeAppliedMember

	registerCalls  int
	heartbeatCalls int
	latestCalls    int
	beginCalls     int
	completeCalls  int
	failCalls      int

	heartbeatErrorAfter int
	heartbeatError      error
	completeErrors      int
	completeCommits     bool
	completeError       error

	registeredSignal chan struct{}
	heartbeatSignal  chan struct{}
	latestSignal     chan struct{}
	appliedSignal    chan int64
	failedSignal     chan int64
}

func newPluginRuntimeCoordinatorTestRepository() *pluginRuntimeCoordinatorTestRepository {
	return &pluginRuntimeCoordinatorTestRepository{
		publications:     make(map[int64]PluginRuntimePublication),
		acks:             make(map[int64]PluginRuntimePublicationAck),
		applied:          make(map[int64][]PluginRuntimeAppliedMember),
		registeredSignal: make(chan struct{}, 8),
		heartbeatSignal:  make(chan struct{}, 64),
		latestSignal:     make(chan struct{}, 64),
		appliedSignal:    make(chan int64, 64),
		failedSignal:     make(chan int64, 64),
	}
}

func (r *pluginRuntimeCoordinatorTestRepository) addPublication(publication PluginRuntimePublication) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.publications[publication.Revision] = clonePluginRuntimePublication(publication)
	if publication.Revision > r.latest {
		r.latest = publication.Revision
	}
}

func (r *pluginRuntimeCoordinatorTestRepository) seedNode(
	identity PluginRuntimeNodeIdentity,
	lastApplied int64,
	lease time.Duration,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.node = PluginRuntimeNode{
		PluginRuntimeNodeIdentity: identity,
		LastAppliedRevision:       lastApplied,
		FirstSeenAt:               now,
		LastSeenAt:                now,
		LeaseExpiresAt:            now.Add(lease),
	}
	r.registered = true
}

func (r *pluginRuntimeCoordinatorTestRepository) PublishPluginRuntimePublication(
	ctx context.Context,
	reason PluginRuntimePublicationReason,
	actorUserID int64,
	members []PluginRuntimeMember,
) (PluginRuntimePublication, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimePublication{}, err
	}
	canonical, digest, err := canonicalPluginRuntimeMembers(members)
	if err != nil {
		return PluginRuntimePublication{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	revision := r.latest + 1
	publication := PluginRuntimePublication{
		Revision: revision, MemberCount: len(canonical), MembersDigest: digest,
		Members: canonical, Reason: reason, ActorUserID: actorUserID, CreatedAt: time.Now().UTC(),
	}
	r.publications[revision] = clonePluginRuntimePublication(publication)
	r.latest = revision
	return publication, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) LatestPluginRuntimePublication(
	ctx context.Context,
) (PluginRuntimePublication, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimePublication{}, err
	}
	r.mu.Lock()
	r.latestCalls++
	publication, found := r.publications[r.latest]
	r.mu.Unlock()
	pluginRuntimeCoordinatorTestSignal(r.latestSignal)
	if !found {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationNotFound
	}
	return clonePluginRuntimePublication(publication), nil
}

func (r *pluginRuntimeCoordinatorTestRepository) PluginRuntimePublicationByRevision(
	ctx context.Context,
	revision int64,
) (PluginRuntimePublication, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimePublication{}, err
	}
	r.mu.Lock()
	publication, found := r.publications[revision]
	r.mu.Unlock()
	if !found {
		return PluginRuntimePublication{}, ErrPluginRuntimePublicationNotFound
	}
	return clonePluginRuntimePublication(publication), nil
}

func (r *pluginRuntimeCoordinatorTestRepository) RegisterPluginRuntimeNode(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	lease time.Duration,
) (PluginRuntimeNode, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimeNode{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registerCalls++
	now := time.Now().UTC()
	if !r.registered {
		r.node = PluginRuntimeNode{
			PluginRuntimeNodeIdentity: identity,
			FirstSeenAt:               now,
		}
		r.registered = true
	}
	if r.node.PluginRuntimeNodeIdentity != identity {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	r.node.LastSeenAt = now
	r.node.LeaseExpiresAt = now.Add(lease)
	pluginRuntimeCoordinatorTestSignal(r.registeredSignal)
	return r.node, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) HeartbeatPluginRuntimeNode(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	lease time.Duration,
) (PluginRuntimeNode, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimeNode{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.heartbeatCalls++
	pluginRuntimeCoordinatorTestSignal(r.heartbeatSignal)
	if r.heartbeatErrorAfter > 0 && r.heartbeatCalls >= r.heartbeatErrorAfter {
		if r.heartbeatError == nil {
			return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
		}
		return PluginRuntimeNode{}, r.heartbeatError
	}
	if !r.registered || r.node.PluginRuntimeNodeIdentity != identity {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	now := time.Now().UTC()
	r.node.LastSeenAt = now
	r.node.LeaseExpiresAt = now.Add(lease)
	return r.node, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) GetPluginRuntimeNode(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
) (PluginRuntimeNode, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimeNode{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.registered || r.node.PluginRuntimeNodeIdentity != identity {
		return PluginRuntimeNode{}, ErrPluginRuntimeNodeLeaseLost
	}
	return r.node, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) BeginPluginRuntimePublicationApply(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	publicationRevision int64,
) (PluginRuntimePublicationAck, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beginCalls++
	if !r.registered || r.node.PluginRuntimeNodeIdentity != identity ||
		publicationRevision <= r.node.LastAppliedRevision {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	if _, found := r.publications[publicationRevision]; !found {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	now := time.Now().UTC()
	ack, found := r.acks[publicationRevision]
	if !found {
		ack = PluginRuntimePublicationAck{
			PublicationRevision:       publicationRevision,
			PluginRuntimeNodeIdentity: identity,
			Status:                    PluginRuntimeAckApplying, AttemptCount: 1, Revision: 1,
			StartedAt: now, UpdatedAt: now,
		}
	} else if ack.Status == PluginRuntimeAckFailed {
		ack.Status = PluginRuntimeAckApplying
		ack.AppliedMemberCount = nil
		ack.AppliedMembersDigest = ""
		ack.ErrorReason = ""
		ack.AppliedAt = nil
		ack.AttemptCount++
		ack.Revision++
		ack.StartedAt = now
		ack.UpdatedAt = now
	} else if ack.Status != PluginRuntimeAckApplying {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	r.acks[publicationRevision] = ack
	return ack, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) CompletePluginRuntimePublicationApply(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	publication PluginRuntimePublication,
	expectedAckRevision int64,
	applied []PluginRuntimeAppliedMember,
) (PluginRuntimePublicationAck, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completeCalls++
	if r.completeErrors > 0 && !r.completeCommits {
		r.completeErrors--
		return PluginRuntimePublicationAck{}, r.completeError
	}
	ack, found := r.acks[publication.Revision]
	if !found || ack.PluginRuntimeNodeIdentity != identity ||
		ack.Status != PluginRuntimeAckApplying || ack.Revision != expectedAckRevision {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	canonical, digest, err := canonicalPluginRuntimeAppliedMembers(publication.Members, applied)
	if err != nil || digest != publication.MembersDigest {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	now := time.Now().UTC()
	count := len(canonical)
	ack.Status = PluginRuntimeAckApplied
	ack.AppliedMemberCount = &count
	ack.AppliedMembersDigest = digest
	ack.ErrorReason = ""
	ack.Revision++
	ack.UpdatedAt = now
	ack.AppliedAt = &now
	r.acks[publication.Revision] = ack
	r.applied[publication.Revision] = append([]PluginRuntimeAppliedMember(nil), canonical...)
	r.node.LastAppliedRevision = publication.Revision
	pluginRuntimeCoordinatorTestRevisionSignal(r.appliedSignal, publication.Revision)
	if r.completeErrors > 0 {
		r.completeErrors--
		return PluginRuntimePublicationAck{}, r.completeError
	}
	return ack, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) FailPluginRuntimePublicationApply(
	ctx context.Context,
	identity PluginRuntimeNodeIdentity,
	publicationRevision int64,
	expectedAckRevision int64,
	reason string,
) (PluginRuntimePublicationAck, error) {
	if err := ctx.Err(); err != nil {
		return PluginRuntimePublicationAck{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failCalls++
	ack, found := r.acks[publicationRevision]
	if !found || ack.PluginRuntimeNodeIdentity != identity ||
		ack.Status != PluginRuntimeAckApplying || ack.Revision != expectedAckRevision {
		return PluginRuntimePublicationAck{}, ErrPluginRuntimeAckConflict
	}
	now := time.Now().UTC()
	ack.Status = PluginRuntimeAckFailed
	ack.ErrorReason = strings.TrimSpace(reason)
	ack.Revision++
	ack.UpdatedAt = now
	r.acks[publicationRevision] = ack
	pluginRuntimeCoordinatorTestRevisionSignal(r.failedSignal, publicationRevision)
	return ack, nil
}

func (r *pluginRuntimeCoordinatorTestRepository) snapshot() pluginRuntimeCoordinatorTestRepositorySnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	acks := make(map[int64]PluginRuntimePublicationAck, len(r.acks))
	for revision, ack := range r.acks {
		acks[revision] = ack
	}
	applied := make(map[int64][]PluginRuntimeAppliedMember, len(r.applied))
	for revision, members := range r.applied {
		applied[revision] = append([]PluginRuntimeAppliedMember(nil), members...)
	}
	return pluginRuntimeCoordinatorTestRepositorySnapshot{
		node: r.node, acks: acks, applied: applied,
		registerCalls: r.registerCalls, heartbeatCalls: r.heartbeatCalls,
		latestCalls: r.latestCalls, beginCalls: r.beginCalls,
		completeCalls: r.completeCalls, failCalls: r.failCalls,
	}
}

type pluginRuntimeCoordinatorTestRepositorySnapshot struct {
	node                          PluginRuntimeNode
	acks                          map[int64]PluginRuntimePublicationAck
	applied                       map[int64][]PluginRuntimeAppliedMember
	registerCalls, heartbeatCalls int
	latestCalls, beginCalls       int
	completeCalls, failCalls      int
}

type pluginRuntimeCoordinatorTestApplier struct {
	mu sync.Mutex

	calls     []int64
	active    int
	maxActive int
	apply     func(context.Context, PluginRuntimePublication, int) ([]PluginRuntimeAppliedMember, error)
}

func (a *pluginRuntimeCoordinatorTestApplier) ApplyPluginRuntimeFullSet(
	ctx context.Context,
	publication PluginRuntimePublication,
) ([]PluginRuntimeAppliedMember, error) {
	a.mu.Lock()
	a.calls = append(a.calls, publication.Revision)
	call := len(a.calls)
	a.active++
	if a.active > a.maxActive {
		a.maxActive = a.active
	}
	apply := a.apply
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.active--
		a.mu.Unlock()
	}()
	if apply != nil {
		return apply(ctx, publication, call)
	}
	return pluginRuntimeCoordinatorAppliedFixture(publication), nil
}

func (a *pluginRuntimeCoordinatorTestApplier) snapshot() ([]int64, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]int64(nil), a.calls...), a.maxActive
}

type pluginRuntimeCoordinatorTestNotifications struct {
	mu    sync.Mutex
	wake  func()
	ready chan struct{}
	once  sync.Once
	done  chan struct{}
}

func newPluginRuntimeCoordinatorTestNotifications() *pluginRuntimeCoordinatorTestNotifications {
	return &pluginRuntimeCoordinatorTestNotifications{
		ready: make(chan struct{}), done: make(chan struct{}),
	}
}

func (n *pluginRuntimeCoordinatorTestNotifications) WatchPluginRuntimePublications(
	ctx context.Context,
	wake func(),
) {
	n.mu.Lock()
	n.wake = wake
	n.mu.Unlock()
	n.once.Do(func() { close(n.ready) })
	<-ctx.Done()
	close(n.done)
}

func (n *pluginRuntimeCoordinatorTestNotifications) notify(count int) {
	n.mu.Lock()
	wake := n.wake
	n.mu.Unlock()
	for range count {
		wake()
	}
}

func pluginRuntimeCoordinatorPublicationFixture(
	revision int64,
	members []PluginRuntimeMember,
) PluginRuntimePublication {
	canonical, digest, err := canonicalPluginRuntimeMembers(members)
	if err != nil {
		panic(err)
	}
	return PluginRuntimePublication{
		Revision: revision, MemberCount: len(canonical), MembersDigest: digest,
		Members: canonical, Reason: PluginRuntimePublicationStartupReconcile,
		CreatedAt: time.Unix(1_700_000_000+revision, 0).UTC(),
	}
}

func pluginRuntimeCoordinatorMemberFixture(id string, versionID int64, digestByte string) PluginRuntimeMember {
	return PluginRuntimeMember{
		ExtensionID: id, ExtensionVersionID: versionID, ExtensionVersion: fmt.Sprintf("%d.0.0", versionID),
		PackageDigest: strings.Repeat(digestByte, 64),
	}
}

func pluginRuntimeCoordinatorAppliedFixture(publication PluginRuntimePublication) []PluginRuntimeAppliedMember {
	applied := make([]PluginRuntimeAppliedMember, 0, len(publication.Members))
	for _, member := range publication.Members {
		applied = append(applied, PluginRuntimeAppliedMember{
			PluginRuntimeMember: member,
			RuntimeInstanceID:   fmt.Sprintf("runtime-%d-%s", publication.Revision, member.ExtensionID),
		})
	}
	return applied
}

func pluginRuntimeCoordinatorConfigFixture(
	identity PluginRuntimeNodeIdentity,
) PluginRuntimeCoordinatorConfig {
	return PluginRuntimeCoordinatorConfig{
		Identity: identity, LeaseDuration: 600 * time.Millisecond,
		HeartbeatInterval: 20 * time.Millisecond, PollInterval: 10 * time.Millisecond,
	}
}

func pluginRuntimeCoordinatorTestSignal(ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func pluginRuntimeCoordinatorTestRevisionSignal(ch chan<- int64, revision int64) {
	select {
	case ch <- revision:
	default:
	}
}

func waitPluginRuntimeCoordinatorSignal(ctx context.Context, signal <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-signal:
		return nil
	}
}

func waitPluginRuntimeCoordinatorRevision(ctx context.Context, signal <-chan int64, revision int64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case got := <-signal:
			if got == revision {
				return nil
			}
		}
	}
}

func assertPluginRuntimeCoordinatorRunStopped(ctx context.Context, result <-chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-result:
		return err
	}
}

var errPluginRuntimeCoordinatorTestApply = errors.New("test plugin runtime apply failed")

var pluginRuntimeCoordinatorTestIdentitySequence atomic.Int64

func uniquePluginRuntimeCoordinatorTestIdentity(
	stem string,
	role PluginRuntimeProcessRole,
) PluginRuntimeNodeIdentity {
	sequence := pluginRuntimeCoordinatorTestIdentitySequence.Add(1)
	return PluginRuntimeNodeIdentity{
		NodeID: fmt.Sprintf("%s-%d", stem, sequence), ProcessRole: role,
		BootID: fmt.Sprintf("boot-%s-%d", stem, sequence),
	}
}
