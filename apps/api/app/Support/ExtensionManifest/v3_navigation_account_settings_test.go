package extensionmanifest

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// accountSettingsNavigationManifest 构造一个最小的 V3 manifest，其 navigation 声明指向
// Host 拥有的个人设置侧栏（account-settings）。
func accountSettingsNavigationManifest(visibility string) Manifest {
	manifest := versionedTestManifest(ManifestVersionV3)
	manifest.ID = "demo.account"
	manifest.Permissions = []string{"demo.account.apps"}
	manifest.Navigation = []ManifestNavigation{{
		ID: "demo.account.navigation.apps", ContractVersion: "demo.account.navigation.apps@1",
		Kind: "account-settings", Action: "add", Label: "Apps",
		Href: "/settings/demo-apps", Permission: "demo.account.apps",
		Visibility: visibility, OwnerResource: "demo.account.resource", Order: 50,
	}}
	return manifest
}

func marshalManifest(t *testing.T, manifest Manifest) []byte {
	t.Helper()
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// account-settings 必须与 visibility / ownerResource 一起通过 JSON Schema 与语义校验，
// 且新字段能被 Go 结构体完整往返。
func TestManifestV3AccountSettingsNavigationDeclarationPassesContract(t *testing.T) {
	manifest := accountSettingsNavigationManifest("authenticated")
	body := marshalManifest(t, manifest)
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("account-settings declaration rejected by JSON Schema: %v", err)
	}
	var decoded Manifest
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := Validate(decoded); err != nil {
		t.Fatalf("account-settings declaration rejected by semantic validation: %v", err)
	}
	if len(decoded.Navigation) != 1 {
		t.Fatalf("navigation declarations = %#v", decoded.Navigation)
	}
	item := decoded.Navigation[0]
	if item.Kind != "account-settings" || item.Visibility != "authenticated" ||
		item.OwnerResource != "demo.account.resource" || item.Href != "/settings/demo-apps" {
		t.Fatalf("account-settings declaration did not round-trip: %#v", item)
	}
}

// visibility 在 Schema 中是可选的：默认 authenticated 由生命周期发布层抬升，
// manifest 契约不得代劳，否则「省略 visibility」这一合法写法会被误拒。
func TestManifestV3AccountSettingsNavigationAllowsOmittedVisibility(t *testing.T) {
	body := marshalManifest(t, accountSettingsNavigationManifest(""))
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("omitted visibility must stay legal: %v", err)
	}
	var decoded Manifest
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := Validate(decoded); err != nil {
		t.Fatalf("omitted visibility rejected by semantic validation: %v", err)
	}
	if decoded.Navigation[0].Visibility != "" {
		t.Fatalf("manifest layer must not default visibility: %#v", decoded.Navigation[0])
	}
}

// 回归：新增 kind 不得把白名单放宽成「任意字符串」。
func TestManifestV3NavigationRejectsUnknownKind(t *testing.T) {
	manifest := accountSettingsNavigationManifest("authenticated")
	manifest.Navigation[0].Kind = "personal-settings"
	body := marshalManifest(t, manifest)
	err := ValidateV3JSONSchema(body)
	if err == nil || !strings.Contains(err.Error(), "account-settings") {
		t.Fatalf("unknown kind must be rejected with the accepted enum, got: %v", err)
	}
	var decoded Manifest
	if unmarshalErr := json.Unmarshal(body, &decoded); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if err := Validate(decoded); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("Go navigation whitelist must still fail closed, got: %v", err)
	}
}

// 回归：visibility 只接受三个合法值。
func TestManifestV3NavigationRejectsUnknownVisibility(t *testing.T) {
	body := marshalManifest(t, accountSettingsNavigationManifest("owner"))
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("visibility outside public|anonymous|authenticated must be rejected")
	}
}

// 回归：只为 navigation item 打开 visibility / ownerResource 两个口子，
// additionalProperties: false 仍然拒绝其他未知字段。
func TestManifestV3NavigationItemStillRejectsUnknownProperties(t *testing.T) {
	body := marshalManifest(t, accountSettingsNavigationManifest("authenticated"))
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	navigation, ok := object["navigation"].([]any)
	if !ok || len(navigation) != 1 {
		t.Fatalf("navigation payload = %#v", object["navigation"])
	}
	navigation[0].(map[string]any)["bogusField"] = true
	drifted, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(drifted); err == nil {
		t.Fatal("unknown navigation property must still be rejected")
	}
}

// 零行为变化：既有 6 个 kind 与新增 kind 一起仍全部通过契约。
func TestManifestV3NavigationAcceptsEveryDeclaredKind(t *testing.T) {
	for _, kind := range []string{"menu", "item", "breadcrumb", "header", "footer", "sidebar", "account-settings"} {
		t.Run(kind, func(t *testing.T) {
			manifest := accountSettingsNavigationManifest("authenticated")
			manifest.Navigation[0].Kind = kind
			body := marshalManifest(t, manifest)
			if err := ValidateV3JSONSchema(body); err != nil {
				t.Fatalf("%s rejected by JSON Schema: %v", kind, err)
			}
			var decoded Manifest
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal(err)
			}
			if err := Validate(decoded); err != nil {
				t.Fatalf("%s rejected by semantic validation: %v", kind, err)
			}
		})
	}
}
