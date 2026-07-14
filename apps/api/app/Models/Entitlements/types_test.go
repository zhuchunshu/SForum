package entitlements

import (
	"errors"
	"testing"
	"time"
)

func TestGrantFingerprintUsesNormalizedProviderNeutralInput(t *testing.T) {
	validFrom := time.Date(2026, 7, 15, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	validUntil := validFrom.Add(time.Hour)
	base := GrantInput{
		Subject:   Subject{Type: " user ", ID: " 42 "},
		Scope:     Scope{Kind: ScopeCapability, Capability: " forum.export "},
		Source:    Source{Type: " plugin ", ID: " demo.provider "},
		ValidFrom: validFrom, ValidUntil: &validUntil,
		ActorUserID: 7, IdempotencyKey: "grant:42:export",
	}
	prepared, fingerprint, err := prepareGrant(base)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Subject.Type != "user" || prepared.Subject.ID != "42" ||
		prepared.Scope.Capability != "forum.export" || prepared.Source.ID != "demo.provider" {
		t.Fatalf("prepared grant = %#v", prepared)
	}

	same := base
	same.ValidFrom = validFrom.UTC()
	sameUntil := validUntil.UTC()
	same.ValidUntil = &sameUntil
	_, sameFingerprint, err := prepareGrant(same)
	if err != nil {
		t.Fatal(err)
	}
	if sameFingerprint != fingerprint {
		t.Fatalf("same instant fingerprint = %q want %q", sameFingerprint, fingerprint)
	}

	changed := base
	changed.ActorUserID = 8
	_, changedFingerprint, err := prepareGrant(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFingerprint == fingerprint {
		t.Fatal("actor change must change the request fingerprint")
	}
}

func TestGrantValidationRejectsMixedScopeAndInvalidWindow(t *testing.T) {
	start := time.Now().UTC()
	end := start
	input := GrantInput{
		Subject: Subject{Type: "user", ID: "42"},
		Scope: Scope{
			Kind: ScopeResource, ResourceType: "topic", ResourceID: "9", Capability: "topic.read",
		},
		Source:    Source{Type: "plugin", ID: "demo"},
		ValidFrom: start, ValidUntil: &end, IdempotencyKey: "grant:42:topic:9",
	}
	if _, _, err := prepareGrant(input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("mixed scope error = %v", err)
	}
	input.Scope.Capability = ""
	if _, _, err := prepareGrant(input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid window error = %v", err)
	}
}

func TestTransitionFingerprintBindsActionActorAndEntitlement(t *testing.T) {
	input := TransitionInput{EntitlementID: 11, ActorUserID: 7, IdempotencyKey: "transition:11"}
	_, revoke, err := prepareTransition(ActionRevoke, input)
	if err != nil {
		t.Fatal(err)
	}
	_, expire, err := prepareTransition(ActionExpire, input)
	if err != nil {
		t.Fatal(err)
	}
	if revoke == expire {
		t.Fatal("transition action must be fingerprint-bound")
	}
	input.ActorUserID++
	_, changedActor, err := prepareTransition(ActionRevoke, input)
	if err != nil {
		t.Fatal(err)
	}
	if changedActor == revoke {
		t.Fatal("transition actor must be fingerprint-bound")
	}
}
