package identity

import (
	"context"
	"errors"
	"strings"
	"testing"

	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

func TestAuthProviderFlowStartAndLoginComplete(t *testing.T) {
	provider := testAuthProviderContribution("demo.auth", []string{
		AuthOperationLoginStart, AuthOperationLoginComplete,
	})
	source := &fakeAuthProviderSource{provider: provider}
	invoker := &fakeAuthProviderInvoker{
		startOutput: map[string]any{
			"status": "redirect", "redirectUrl": "https://idp.example/auth", "continueToken": "tok-1",
		},
		completeOutput: map[string]any{
			"providerSubjectDigest": strings.Repeat("a", 64),
			"displayName":           "Demo User",
			"emailHint":             "demo@example.com",
		},
	}
	flow, err := NewAuthProviderFlow(source, invoker, nil)
	if err != nil {
		t.Fatal(err)
	}

	start, err := flow.Start(context.Background(), AuthProviderStartInput{
		ProviderID: "demo.auth", Operation: AuthOperationLoginStart, CorrelationID: "corr-1",
	})
	if err != nil || start.Status != AuthStartStatusRedirect || start.RedirectURL == "" || start.ContinueToken != "tok-1" {
		t.Fatalf("start=%#v err=%v", start, err)
	}

	complete, err := flow.Complete(context.Background(), AuthProviderCompleteInput{
		ProviderID: "demo.auth", Operation: AuthOperationLoginComplete,
		CorrelationID: "corr-1", CompletionToken: "code-1",
	})
	if err != nil || complete.SubjectDigest != strings.Repeat("a", 64) || complete.Link != nil {
		t.Fatalf("complete=%#v err=%v", complete, err)
	}
	if invoker.lastOperation != AuthOperationLoginComplete || invoker.lastActorUserID != 0 {
		t.Fatalf("invoker state=%#v", invoker)
	}
}

func TestAuthProviderFlowLinkCompleteWritesExternalLink(t *testing.T) {
	provider := testAuthProviderContribution("demo.auth", []string{AuthOperationLinkComplete})
	source := &fakeAuthProviderSource{provider: provider}
	links := &fakeExternalLinkStore{}
	invoker := &fakeAuthProviderInvoker{
		completeOutput: map[string]any{"providerSubjectDigest": strings.Repeat("b", 64)},
	}
	flow, err := NewAuthProviderFlow(source, invoker, links)
	if err != nil {
		t.Fatal(err)
	}
	result, err := flow.Complete(context.Background(), AuthProviderCompleteInput{
		ProviderID: "demo.auth", Operation: AuthOperationLinkComplete,
		ActorUserID: 42, TargetUserID: 42,
		CorrelationID: "corr-link", CompletionToken: "code-link",
		IdempotencyKey: "idem-link-1",
	})
	if err != nil || result.Link == nil || result.Link.Link.UserID != 42 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if links.lastInput.ProviderOperation != AuthOperationLinkComplete ||
		links.lastInput.ProviderSubjectDigest != strings.Repeat("b", 64) ||
		links.lastInput.ActorUserID != 42 || links.fenceCalled != 1 {
		t.Fatalf("link store=%#v", links)
	}
}

func TestAuthProviderFlowRejectsInvalidActorModesAndMissingDigest(t *testing.T) {
	provider := testAuthProviderContribution("demo.auth", []string{
		AuthOperationLoginStart, AuthOperationLoginComplete, AuthOperationLinkStart,
	})
	flow, err := NewAuthProviderFlow(
		&fakeAuthProviderSource{provider: provider},
		&fakeAuthProviderInvoker{startOutput: map[string]any{"status": "continue"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Start(context.Background(), AuthProviderStartInput{
		ProviderID: "demo.auth", Operation: AuthOperationLoginStart,
		ActorUserID: 9, CorrelationID: "c",
	}); !errors.Is(err, ErrAuthProviderFlowInvalid) {
		t.Fatalf("login start with actor: %v", err)
	}
	if _, err := flow.Start(context.Background(), AuthProviderStartInput{
		ProviderID: "demo.auth", Operation: AuthOperationLinkStart, CorrelationID: "c",
	}); !errors.Is(err, ErrAuthProviderFlowInvalid) {
		t.Fatalf("link start without actor: %v", err)
	}
	// complete 缺少主体摘要失败关闭。
	bad := &fakeAuthProviderInvoker{completeOutput: map[string]any{"displayName": "x"}}
	flow, _ = NewAuthProviderFlow(&fakeAuthProviderSource{provider: provider}, bad, nil)
	if _, err := flow.Complete(context.Background(), AuthProviderCompleteInput{
		ProviderID: "demo.auth", Operation: AuthOperationLoginComplete,
		CorrelationID: "c", CompletionToken: "t",
	}); !errors.Is(err, ErrAuthProviderFlowUnavailable) {
		t.Fatalf("missing digest: %v", err)
	}
}

func TestAuthProviderFlowNoFallbackOnMissingProvider(t *testing.T) {
	flow, err := NewAuthProviderFlow(
		&fakeAuthProviderSource{err: identityregistry.ErrNotFound},
		&fakeAuthProviderInvoker{},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Start(context.Background(), AuthProviderStartInput{
		ProviderID: "missing", Operation: AuthOperationLoginStart, CorrelationID: "c",
	}); !errors.Is(err, ErrAuthProviderNotFound) {
		t.Fatalf("missing provider: %v", err)
	}
}

func testAuthProviderContribution(id string, operations []string) identityregistry.ProviderContribution {
	ops := make([]identityregistry.ProviderOperation, 0, len(operations))
	for _, name := range operations {
		ops = append(ops, identityregistry.ProviderOperation{
			Name: name, InputSchema: "in@1", OutputSchema: "out@1",
			TimeoutMS: 1000, FailurePolicy: identityregistry.ProviderFailureFailClosed,
		})
	}
	return identityregistry.ProviderContribution{
		Provider: identityregistry.Provider{
			ID: id, ContractVersion: id + "@1", Kind: identityregistry.ProviderKindAuth,
			Handler: "identity.auth", Operations: ops,
		},
		Artifact: identityregistry.Artifact{
			ExtensionID: "demo.membership", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("1", 64), VersionID: 7,
			RuntimeInstanceID: "runtime-1",
		},
	}
}

type fakeAuthProviderSource struct {
	provider identityregistry.ProviderContribution
	err      error
}

func (f *fakeAuthProviderSource) ResolveAuthProvider(context.Context, string) (identityregistry.ProviderContribution, error) {
	if f.err != nil {
		return identityregistry.ProviderContribution{}, f.err
	}
	return f.provider, nil
}

type fakeAuthProviderInvoker struct {
	startOutput      map[string]any
	completeOutput   map[string]any
	err              error
	lastOperation    string
	lastActorUserID  int64
	lastInput        map[string]any
}

func (f *fakeAuthProviderInvoker) InvokeExact(
	_ context.Context,
	_ identityregistry.ProviderContribution,
	operation string,
	actorUserID int64,
	input map[string]any,
	accept func(context.Context, map[string]any, func() error) error,
) error {
	f.lastOperation = operation
	f.lastActorUserID = actorUserID
	f.lastInput = input
	if f.err != nil {
		return f.err
	}
	output := f.startOutput
	if strings.Contains(operation, "complete") {
		output = f.completeOutput
	}
	return accept(context.Background(), output, func() error { return nil })
}

type fakeExternalLinkStore struct {
	lastInput   LinkExternalIdentityInput
	fenceCalled int
	err         error
}

func (f *fakeExternalLinkStore) Link(
	_ context.Context,
	input LinkExternalIdentityInput,
	fence ExternalIdentityLinkCommitFence,
) (ExternalIdentityLinkMutation, error) {
	f.lastInput = input
	if f.err != nil {
		return ExternalIdentityLinkMutation{}, f.err
	}
	if fence != nil {
		f.fenceCalled++
		if err := fence(); err != nil {
			return ExternalIdentityLinkMutation{}, err
		}
	}
	return ExternalIdentityLinkMutation{
		Link: ExternalIdentityLink{UserID: input.UserID, ProviderID: input.Provider.ID, Status: ExternalIdentityLinkStatusActive},
	}, nil
}

func (f *fakeExternalLinkStore) Unlink(context.Context, TransitionExternalIdentityLinkInput) (ExternalIdentityLinkMutation, error) {
	return ExternalIdentityLinkMutation{}, errors.New("unused")
}
func (f *fakeExternalLinkStore) Erase(context.Context, TransitionExternalIdentityLinkInput) (ExternalIdentityLinkMutation, error) {
	return ExternalIdentityLinkMutation{}, errors.New("unused")
}
func (f *fakeExternalLinkStore) Get(context.Context, int64) (ExternalIdentityLink, error) {
	return ExternalIdentityLink{}, errors.New("unused")
}
func (f *fakeExternalLinkStore) FindActive(context.Context, string, string) (ExternalIdentityLink, error) {
	return ExternalIdentityLink{}, errors.New("unused")
}
func (f *fakeExternalLinkStore) ListUser(context.Context, int64) ([]ExternalIdentityLink, error) {
	return nil, errors.New("unused")
}
