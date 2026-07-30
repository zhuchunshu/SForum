package identitycontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

func externalRegistrationContinuationHarness(
	t *testing.T,
	registrationActivated bool,
	registrationOpen bool,
) (*Controller, *identity.ExternalAuthService, identity.ExternalAuthAssertion) {
	t.Helper()
	digest := strings.Repeat("a", 64)
	live := identityregistry.ProviderContribution{
		Provider: identityregistry.Provider{
			ID: "demo.auth", ContractVersion: "demo.auth@1", Kind: identityregistry.ProviderKindAuth,
			Operations: []identityregistry.ProviderOperation{
				{Name: identity.AuthOperationLoginComplete},
				{Name: identity.AuthOperationRegistrationComplete},
			},
		},
		Artifact: identityregistry.Artifact{
			ExtensionID: "ext.demo.auth", ExtensionVersion: "1.0.0",
			PackageDigest: digest, RuntimeInstanceID: "runtime-demo-auth",
		},
	}
	activation := identity.NewMemoryProviderActivationStore()
	loginEnabled := true
	if _, err := activation.Upsert(t.Context(), identity.ProviderActivationInput{
		ProviderID: live.ID, OwnerExtensionID: live.Artifact.ExtensionID,
		OwnerPackageDigest: live.Artifact.PackageDigest,
		LoginEnabled:       &loginEnabled, RegistrationEnabled: &registrationActivated,
	}); err != nil {
		t.Fatal(err)
	}
	svc := identity.NewExternalAuthService(identity.ExternalAuthDeps{
		LinkStore:       newT1ELinkStore(),
		ActivationStore: activation,
		ProviderContribution: func(string) (identityregistry.ProviderContribution, error) {
			return live, nil
		},
		AnyUserExists: func(context.Context) (bool, error) { return true, nil },
		RegistrationEnabled: func(context.Context) (bool, error) {
			return registrationOpen, nil
		},
	})
	controller := &Controller{
		externalAuthService:     svc,
		registrationTicketStore: identity.NewInMemoryRegistrationTicketStore(),
	}
	assertion := identity.ExternalAuthAssertion{
		ProviderID: live.ID, ProviderContractVersion: live.ContractVersion,
		OwnerExtensionID: live.Artifact.ExtensionID, OwnerExtensionVersion: live.Artifact.ExtensionVersion,
		OwnerPackageDigest: digest, Operation: identity.ExternalAuthOperationLogin,
		SubjectDigest: digest, CorrelationID: "login-continuation",
		UsernameHint: "octocat", DisplayName: "The Octocat",
		EmailHint: "octocat@example.com", EmailVerified: true,
	}
	return controller, svc, assertion
}

func TestExternalLoginUnlinkedContinuesIntoExistingRegistration(t *testing.T) {
	controller, svc, assertion := externalRegistrationContinuationHarness(t, true, true)
	app := fiber.New()
	app.Get("/callback", func(c fiber.Ctx) error {
		return controller.handleExternalLoginCallback(c, identity.CallbackTransaction{}, assertion, "/topics")
	})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/callback", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}
	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil || location.Path != "/register" {
		t.Fatalf("location=%q err=%v", response.Header.Get("Location"), err)
	}
	token := location.Query().Get("ticket")
	if token == "" || location.Query().Get("redirect") != "/topics" {
		t.Fatalf("continuation query=%v", location.Query())
	}
	ticket, err := controller.registrationTicketStore.Inspect(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if ticket.Operation != identity.ExternalAuthOperationRegistration || ticket.SourceOperation != identity.ExternalAuthOperationLogin {
		t.Fatalf("ticket operations target=%q source=%q", ticket.Operation, ticket.SourceOperation)
	}
	preparation, err := svc.PrepareExternalRegistration(t.Context(), ticket)
	if err != nil {
		t.Fatal(err)
	}
	if preparation.UsernameHint != "octocat" || preparation.EmailHint != "octocat@example.com" || !preparation.EmailVerified {
		t.Fatalf("preparation=%#v", preparation)
	}
	if _, err := controller.registrationTicketStore.Consume(t.Context(), token); err != nil {
		t.Fatalf("prepare consumed ticket: %v", err)
	}
}

func TestExternalLoginUnlinkedContinuationFailsClosed(t *testing.T) {
	cases := []struct {
		name                  string
		registrationActivated bool
		registrationOpen      bool
		wantReason            string
	}{
		{name: "provider registration disabled", registrationActivated: false, registrationOpen: true, wantReason: "auth.provider_not_enabled"},
		{name: "site registration closed", registrationActivated: true, registrationOpen: false, wantReason: "auth.registration_disabled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller, _, assertion := externalRegistrationContinuationHarness(t, tc.registrationActivated, tc.registrationOpen)
			app := fiber.New()
			app.Get("/callback", func(c fiber.Ctx) error {
				return controller.handleExternalLoginCallback(c, identity.CallbackTransaction{}, assertion, "/topics")
			})
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/callback", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			location := response.Header.Get("Location")
			if !strings.HasPrefix(location, "/login?") || !strings.Contains(location, "ext_auth="+url.QueryEscape(tc.wantReason)) || strings.Contains(location, "ticket=") {
				t.Fatalf("location=%q", location)
			}
		})
	}
}

func TestExternalRegistrationPreparationIsRedactedAndNonConsuming(t *testing.T) {
	controller, _, assertion := externalRegistrationContinuationHarness(t, true, true)
	assertion.EmailVerified = false
	target, err := controller.createExternalRegistrationContinuation(t.Context(), assertion, "/")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(target)
	token := parsed.Query().Get("ticket")
	app := fiber.New()
	app.Post("/prepare", controller.externalRegistrationPreparation)
	payload, _ := json.Marshal(map[string]string{"ticket": token})
	request := httptest.NewRequest(http.MethodPost, "/prepare", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}
	raw := string(body)
	for _, forbidden := range []string{"providerSubject", "subjectDigest", "ownerPackageDigest", assertion.SubjectDigest, assertion.EmailHint} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("preparation leaked %q: %s", forbidden, raw)
		}
	}
	if !strings.Contains(raw, `"usernameHint":"octocat"`) || !strings.Contains(raw, `"emailVerified":false`) {
		t.Fatalf("preparation body=%s", raw)
	}
	if _, err := controller.registrationTicketStore.Consume(t.Context(), token); err != nil {
		t.Fatalf("prepare consumed ticket: %v", err)
	}
}
