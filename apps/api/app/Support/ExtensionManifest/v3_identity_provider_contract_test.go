package extensionmanifest

import (
	"encoding/json"
	"testing"
)

func TestManifestV3IdentityProviderKeepsLegacyDeclarationsInspectOnly(t *testing.T) {
	manifest := completeV3Manifest()
	if got := manifest.Identity.Providers[0].Operations; len(got) != 0 {
		t.Fatalf("legacy identity provider gained operations: %#v", got)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy inspect-only provider: %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("legacy embedded schema: %v", err)
	}

	minimal := minimalInspectOnlyIdentityManifest()
	if err := Validate(minimal); err != nil {
		t.Fatalf("inspect-only provider must not require a Protocol V2 backend: %v", err)
	}
	body, err = json.Marshal(minimal)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("minimal inspect-only embedded schema: %v", err)
	}
}

func TestManifestV3IdentityProviderNormalizesFixedDefaults(t *testing.T) {
	manifest := completeExecutableIdentityManifest("risk", "risk.evaluate")
	provider := &manifest.Identity.Providers[0]
	provider.Kind = " RISK "
	operation := &provider.Operations[0]
	operation.Name = " RISK.EVALUATE "
	operation.InputSchema = " demo.v3.identity.input@1 "
	operation.OutputSchema = " demo.v3.identity.output@1 "
	operation.TimeoutMS = 0
	operation.FailurePolicy = ""
	normalized := Normalize(manifest)
	operation = &normalized.Identity.Providers[0].Operations[0]
	if operation.Name != "risk.evaluate" || operation.TimeoutMS != ManifestIdentityProviderDefaultTimeoutMS ||
		operation.FailurePolicy != IdentityProviderFailureFailClosed {
		t.Fatalf("risk operation defaults = %#v", operation)
	}

	profile := completeExecutableIdentityManifest("profile", "sections.list")
	profile.Identity.Providers[0].Operations[0].FailurePolicy = ""
	if got := Normalize(profile).Identity.Providers[0].Operations[0].FailurePolicy; got != IdentityProviderFailureOmit {
		t.Fatalf("presentation failure policy = %q, want omit", got)
	}
}

func TestManifestV3IdentityProviderKeepsLegacyHandlerNames(t *testing.T) {
	manifest := completeExecutableIdentityManifest("risk", "risk.evaluate")
	manifest.Identity.Providers[0].Handler = "identity.risk"
	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy provider handler gained an owner-prefix requirement: %v", err)
	}
}

func TestManifestV3IdentityProviderAcceptsFrozenOperationCatalog(t *testing.T) {
	catalog := map[string][]string{
		"auth":     {"registration.start", "registration.complete", "login.start", "login.complete", "link.start", "link.complete"},
		"profile":  {"sections.list", "section.read", "section.update", "account.read", "account.update"},
		"recovery": {"recovery.start", "recovery.complete"},
		"session":  {"session.evaluate"},
		"risk":     {"risk.evaluate"},
	}
	for kind, names := range catalog {
		for _, name := range names {
			t.Run(kind+"/"+name, func(t *testing.T) {
				manifest := completeExecutableIdentityManifest(kind, name)
				if err := Validate(manifest); err != nil {
					t.Fatalf("valid identity operation: %v", err)
				}
				body, err := json.Marshal(manifest)
				if err != nil {
					t.Fatal(err)
				}
				if err := ValidateV3JSONSchema(body); err != nil {
					t.Fatalf("embedded identity operation schema: %v", err)
				}
			})
		}
	}
}

func TestManifestV3IdentityProviderAcceptsPackagePathSchemas(t *testing.T) {
	manifest := completeExecutableIdentityManifest("risk", "risk.evaluate")
	operation := &manifest.Identity.Providers[0].Operations[0]
	operation.InputSchema = "schemas/identity-input.json"
	operation.OutputSchema = "schemas/identity-output.json"
	if err := Validate(manifest); err != nil {
		t.Fatalf("package-path schemas: %v", err)
	}
}

func TestManifestV3SessionPolicyRequiresExecutableSameManifestProvider(t *testing.T) {
	valid := completeExecutableIdentityManifest("session", "session.evaluate")
	valid.Identity.SessionPolicy = valid.Identity.Providers[0].ID
	if err := Validate(valid); err != nil {
		t.Fatalf("valid session policy provider: %v", err)
	}

	tests := []struct {
		name  string
		build func() Manifest
	}{
		{
			name: "missing provider",
			build: func() Manifest {
				manifest := completeV3Manifest()
				manifest.Identity.SessionPolicy = "demo.v3.identity.session"
				return manifest
			},
		},
		{
			name: "wrong provider kind",
			build: func() Manifest {
				manifest := completeExecutableIdentityManifest("risk", "risk.evaluate")
				manifest.Identity.SessionPolicy = manifest.Identity.Providers[0].ID
				return manifest
			},
		},
		{
			name: "inspect-only session provider",
			build: func() Manifest {
				manifest := completeV3Manifest()
				manifest.Identity.Providers = []ManifestIdentityProvider{{
					ID: "demo.v3.identity.session", ContractVersion: "demo.v3.identity.session@1",
					Kind: "session", Handler: "identity.session",
				}}
				manifest.Identity.SessionPolicy = manifest.Identity.Providers[0].ID
				return manifest
			},
		},
		{
			name: "different executable session provider",
			build: func() Manifest {
				manifest := completeExecutableIdentityManifest("session", "session.evaluate")
				manifest.Identity.SessionPolicy = "demo.v3.identity.missing"
				return manifest
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.build()); err == nil {
				t.Fatal("unbound session policy was accepted")
			}
		})
	}
}

func TestManifestV3IdentityProviderRejectsUnsafeExecutableContracts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{name: "operation from another kind", mutate: func(value *Manifest) { value.Identity.Providers[0].Operations[0].Name = "login.start" }},
		{name: "session revoke is Host owned", mutate: func(value *Manifest) {
			value.Identity.Providers[0].Kind = "session"
			value.Identity.Providers[0].Operations[0].Name = "session.revoke"
		}},
		{name: "duplicate operation", mutate: func(value *Manifest) {
			value.Identity.Providers[0].Operations = append(value.Identity.Providers[0].Operations, value.Identity.Providers[0].Operations[0])
		}},
		{name: "security operation omit", mutate: func(value *Manifest) { value.Identity.Providers[0].Operations[0].FailurePolicy = "omit" }},
		{name: "presentation operation fail closed", mutate: func(value *Manifest) {
			value.Identity.Providers[0].Kind = "profile"
			value.Identity.Providers[0].Operations[0].Name = "sections.list"
			value.Identity.Providers[0].Operations[0].FailurePolicy = "fail_closed"
		}},
		{name: "account read omit", mutate: func(value *Manifest) {
			value.Identity.Providers[0].Kind = "profile"
			value.Identity.Providers[0].Operations[0].Name = "account.read"
			value.Identity.Providers[0].Operations[0].FailurePolicy = "omit"
		}},
		{name: "missing input schema", mutate: func(value *Manifest) { value.Identity.Providers[0].Operations[0].InputSchema = "" }},
		{name: "timeout overflow", mutate: func(value *Manifest) {
			value.Identity.Providers[0].Operations[0].TimeoutMS = ManifestIdentityProviderMaximumTimeoutMS + 1
		}},
		{name: "protocol v1", mutate: func(value *Manifest) { value.Backend.ProtocolVersion = 1 }},
		{name: "missing output schema file", mutate: func(value *Manifest) { value.PackageFiles = value.PackageFiles[:len(value.PackageFiles)-1] }},
		{name: "schema version drift", mutate: func(value *Manifest) { value.PackageFiles[len(value.PackageFiles)-1].Version = "2" }},
		{name: "schema kind drift", mutate: func(value *Manifest) { value.PackageFiles[len(value.PackageFiles)-1].Kind = "asset" }},
		{name: "operation overflow", mutate: func(value *Manifest) {
			operation := value.Identity.Providers[0].Operations[0]
			value.Identity.Providers[0].Operations = make([]ManifestIdentityProviderOperation, ManifestIdentityProviderMaximumOperations+1)
			for index := range value.Identity.Providers[0].Operations {
				value.Identity.Providers[0].Operations[index] = operation
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeExecutableIdentityManifest("risk", "risk.evaluate")
			test.mutate(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("unsafe executable identity contract was accepted")
			}
		})
	}
}

func TestManifestV3IdentityProviderSchemaRejectsRemovedOperation(t *testing.T) {
	manifest := completeExecutableIdentityManifest("session", "session.evaluate")
	manifest.Identity.Providers[0].Operations[0].Name = "session.revoke"
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("embedded schema accepted Host-owned session.revoke")
	}
}

func TestManifestV3IdentityProviderSchemaRejectsContractDrift(t *testing.T) {
	tests := []struct {
		name   string
		build  func() Manifest
		mutate func(*Manifest)
	}{
		{
			name:  "operation from another kind",
			build: func() Manifest { return completeExecutableIdentityManifest("auth", "login.start") },
			mutate: func(value *Manifest) {
				value.Identity.Providers[0].Operations[0].Name = "risk.evaluate"
			},
		},
		{
			name:  "fixed presentation policy",
			build: func() Manifest { return completeExecutableIdentityManifest("profile", "sections.list") },
			mutate: func(value *Manifest) {
				value.Identity.Providers[0].Operations[0].FailurePolicy = "fail_closed"
			},
		},
		{
			name: "executable provider without backend",
			build: func() Manifest {
				value := minimalInspectOnlyIdentityManifest()
				value.Identity.Providers[0].Operations = []ManifestIdentityProviderOperation{{
					Name: "login.start", InputSchema: "demo.identity.inspect.input@1",
					OutputSchema: "demo.identity.inspect.output@1",
					TimeoutMS:    ManifestIdentityProviderDefaultTimeoutMS, FailurePolicy: IdentityProviderFailureFailClosed,
				}}
				return value
			},
			mutate: func(*Manifest) {},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := test.build()
			test.mutate(&manifest)
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateV3JSONSchema(body); err == nil {
				t.Fatal("embedded schema accepted identity contract drift")
			}
			if _, err := LoadRootBytes(body, FileMapFS{}); err == nil {
				t.Fatal("package loader accepted identity contract drift")
			}
		})
	}
}

func completeExecutableIdentityManifest(kind, name string) Manifest {
	manifest := completeV3Manifest()
	policy, ok := identityProviderOperationPolicy(kind, name)
	if !ok {
		policy = IdentityProviderFailureFailClosed
	}
	manifest.Identity.Providers = []ManifestIdentityProvider{{
		ID: "demo.v3.identity.provider", ContractVersion: "demo.v3.identity.provider@1",
		Kind: kind, Handler: "demo.v3.identity.provider", Operations: []ManifestIdentityProviderOperation{{
			Name: name, InputSchema: "demo.v3.identity.input@1", OutputSchema: "demo.v3.identity.output@1",
			TimeoutMS: ManifestIdentityProviderDefaultTimeoutMS, FailurePolicy: policy,
		}},
	}}
	manifest.PackageFiles = append(manifest.PackageFiles,
		ManifestPackageFile{ID: "demo.v3.identity.input", Kind: "schema", Path: "schemas/identity-input.json", Digest: v3FixtureDigest(), Version: "1"},
		ManifestPackageFile{ID: "demo.v3.identity.output", Kind: "schema", Path: "schemas/identity-output.json", Digest: v3FixtureDigest(), Version: "1"},
	)
	return manifest
}

func minimalInspectOnlyIdentityManifest() Manifest {
	base := completeV3Manifest()
	return Manifest{
		ManifestVersion: ManifestVersionV3,
		ID:              "demo.identity.inspect",
		Name:            base.Name,
		Description:     base.Description,
		URL:             base.URL,
		Author:          base.Author,
		Version:         base.Version,
		Type:            TypePlugin,
		SForumVersion:   base.SForumVersion,
		Identity: &ManifestIdentity{
			ContractVersion: "demo.identity.inspect.contract@1",
			Providers: []ManifestIdentityProvider{{
				ID: "demo.identity.inspect.provider", ContractVersion: "demo.identity.inspect.provider@1",
				Kind: "auth", Handler: "legacy.auth",
			}},
		},
	}
}
