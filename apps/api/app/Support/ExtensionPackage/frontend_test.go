package extensionpackage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const validTestIntegrity = "sha512-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="

func TestInspectAdminFrontendAcceptsLockedHuJSONAndReturnsStableSummary(t *testing.T) {
	fixture := newFrontendFixture(t)
	sentinel := filepath.Join(fixture.packageRoot, "script-ran")
	fixture.packageJSON = strings.Replace(
		fixture.packageJSON,
		`"scripts": {}`,
		`"scripts": {"inspect": "touch `+sentinel+`"}`,
		1,
	)
	fixture.write(t)

	summary, err := InspectAdminFrontend(fixture.input())
	if err != nil {
		t.Fatalf("inspect valid admin frontend: %v", err)
	}
	if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inspection executed plugin code, stat error: %v", err)
	}

	wantDirect := []Dependency{
		{Name: "@nuxt/ui", Version: "^4.9.0"},
		{Name: "@scope/alpha", Version: "~1.2.0"},
		{Name: "@sforum/admin-sdk", Version: "^1.0.0"},
		{Name: "builder", Version: "1.x"},
		{Name: "nuxt", Version: "^4.4.0"},
		{Name: "optional-pkg", Version: "^0.5.0"},
		{Name: "vue", Version: "^3.5.0"},
		{Name: "vue-router", Version: "^5.0.0"},
		{Name: "zeta", Version: "^2.0.0"},
	}
	wantResolved := []Dependency{
		{Name: "@scope/alpha", Version: "1.2.4", Integrity: validTestIntegrity},
		{Name: "@scope/nested", Version: "3.0.1", Integrity: validTestIntegrity},
		{Name: "builder", Version: "1.9.0", Integrity: validTestIntegrity},
		{Name: "optional-pkg", Version: "0.5.3", Integrity: validTestIntegrity},
		{Name: "zeta", Version: "2.1.0", Integrity: validTestIntegrity},
	}
	if !reflect.DeepEqual(summary.Direct, wantDirect) {
		t.Fatalf("unexpected direct dependencies:\nwant %#v\n got %#v", wantDirect, summary.Direct)
	}
	if !reflect.DeepEqual(summary.Resolved, wantResolved) {
		t.Fatalf("unexpected resolved dependencies:\nwant %#v\n got %#v", wantResolved, summary.Resolved)
	}
	lockBody, err := os.ReadFile(filepath.Join(fixture.frontendRoot(), "bun.lock"))
	if err != nil {
		t.Fatalf("read fixture lock: %v", err)
	}
	digest := sha256.Sum256(lockBody)
	if summary.LockDigest != hex.EncodeToString(digest[:]) {
		t.Fatalf("lock digest must use original HuJSON bytes: want=%s got=%s", hex.EncodeToString(digest[:]), summary.LockDigest)
	}
}

func TestInspectAdminFrontendRejectsMissingAndEscapingInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*frontendFixture)
	}{
		{
			name: "frontend root escapes package",
			mutate: func(f *frontendFixture) {
				f.root = "../outside"
			},
		},
		{
			name: "component escapes frontend root",
			mutate: func(f *frontendFixture) {
				f.components["job-cell"] = "../outside.vue"
			},
		},
		{
			name: "locale escapes frontend root",
			mutate: func(f *frontendFixture) {
				f.locales["zh-CN"] = "../zh-CN.json"
			},
		},
		{
			name: "missing package json",
			mutate: func(f *frontendFixture) {
				f.omit["package.json"] = true
			},
		},
		{
			name: "missing bun lock",
			mutate: func(f *frontendFixture) {
				f.omit["bun.lock"] = true
			},
		},
		{
			name: "missing component",
			mutate: func(f *frontendFixture) {
				f.omit["components/JobCell.vue"] = true
			},
		},
		{
			name: "missing simplified chinese locale",
			mutate: func(f *frontendFixture) {
				delete(f.locales, "zh-CN")
			},
		},
		{
			name: "missing english locale",
			mutate: func(f *frontendFixture) {
				delete(f.locales, "en-US")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFrontendFixture(t)
			test.mutate(fixture)
			fixture.write(t)
			_, err := InspectAdminFrontend(fixture.input())
			if !errors.Is(err, ErrInvalidAdminFrontend) {
				t.Fatalf("expected ErrInvalidAdminFrontend, got %v", err)
			}
		})
	}
}

func TestInspectAdminFrontendRejectsSymlinkAnywhereInAdminTree(t *testing.T) {
	fixture := newFrontendFixture(t)
	fixture.write(t)
	if err := os.Symlink("JobCell.vue", filepath.Join(fixture.frontendRoot(), "components", "Alias.vue")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := InspectAdminFrontend(fixture.input())
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("expected ErrSymlink, got %v", err)
	}
}

func TestInspectAdminFrontendValidatesLocaleStringLeaves(t *testing.T) {
	valid := []string{
		`{}`,
		`{"table":{"latency":"Latency","nested":{"unit":"ms"}}}`,
	}
	for index, body := range valid {
		t.Run("valid-"+string(rune('a'+index)), func(t *testing.T) {
			fixture := newFrontendFixture(t)
			fixture.localeBodies["en-US"] = body
			fixture.write(t)
			if _, err := InspectAdminFrontend(fixture.input()); err != nil {
				t.Fatalf("valid locale rejected: %v", err)
			}
		})
	}

	invalid := []string{
		`[]`,
		`{"value":1}`,
		`{"value":true}`,
		`{"value":null}`,
		`{"value":["text"]}`,
		`{"value":{}} trailing`,
	}
	for index, body := range invalid {
		t.Run("invalid-"+string(rune('a'+index)), func(t *testing.T) {
			fixture := newFrontendFixture(t)
			fixture.localeBodies["en-US"] = body
			fixture.write(t)
			_, err := InspectAdminFrontend(fixture.input())
			if !errors.Is(err, ErrInvalidAdminFrontend) {
				t.Fatalf("expected invalid locale error, got %v", err)
			}
		})
	}
}

func TestInspectAdminFrontendRejectsLocalDependencyProtocolsInEveryMap(t *testing.T) {
	for _, field := range []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"} {
		for _, protocol := range []string{"workspace:*", "file:../shared", "link:../shared"} {
			name := field + "-" + strings.TrimSuffix(protocol, ":*")
			t.Run(name, func(t *testing.T) {
				fixture := newFrontendFixture(t)
				fixture.packageJSON = addPackageDependency(t, fixture.packageJSON, field, "unsafe-package", protocol)
				fixture.lock = addWorkspaceDependency(t, fixture.lock, field, "unsafe-package", protocol)
				fixture.write(t)
				_, err := InspectAdminFrontend(fixture.input())
				if !errors.Is(err, ErrInvalidAdminFrontend) {
					t.Fatalf("expected forbidden protocol error, got %v", err)
				}
			})
		}
	}
}

func TestInspectAdminFrontendValidatesHostPeerOwnershipAndRanges(t *testing.T) {
	for _, field := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		t.Run("private-"+field, func(t *testing.T) {
			fixture := newFrontendFixture(t)
			fixture.packageJSON = movePackageDependency(t, fixture.packageJSON, "vue", "peerDependencies", field)
			fixture.lock = moveWorkspaceDependency(t, fixture.lock, "vue", "peerDependencies", field)
			fixture.write(t)
			_, err := InspectAdminFrontend(fixture.input())
			if !errors.Is(err, ErrInvalidAdminFrontend) {
				t.Fatalf("expected private host dependency error, got %v", err)
			}
		})
	}

	for _, constraint := range []string{"^3.5.0", ">=3.5.0 <4.0.0", "~3.5.30 || ^4.0.0"} {
		t.Run("compatible-"+constraint, func(t *testing.T) {
			fixture := newFrontendFixture(t)
			fixture.packageJSON = replaceDependencyVersion(t, fixture.packageJSON, "peerDependencies", "vue", constraint)
			fixture.lock = replaceWorkspaceDependencyVersion(t, fixture.lock, "peerDependencies", "vue", constraint)
			fixture.write(t)
			if _, err := InspectAdminFrontend(fixture.input()); err != nil {
				t.Fatalf("compatible host peer rejected: %v", err)
			}
		})
	}

	for _, constraint := range []string{"^2.7.0", ">=4.0.0", "not-a-range"} {
		t.Run("incompatible-"+constraint, func(t *testing.T) {
			fixture := newFrontendFixture(t)
			fixture.packageJSON = replaceDependencyVersion(t, fixture.packageJSON, "peerDependencies", "vue", constraint)
			fixture.lock = replaceWorkspaceDependencyVersion(t, fixture.lock, "peerDependencies", "vue", constraint)
			fixture.write(t)
			_, err := InspectAdminFrontend(fixture.input())
			if !errors.Is(err, ErrInvalidAdminFrontend) {
				t.Fatalf("expected incompatible host peer error, got %v", err)
			}
		})
	}
}

func TestInspectAdminFrontendRequiresCompleteHostPeerCatalog(t *testing.T) {
	fixture := newFrontendFixture(t)
	delete(fixture.hostPeers, "vue")
	fixture.packageJSON = removeDependencyFromJSON(t, fixture.packageJSON, "peerDependencies", "vue", "^3.5.0")
	fixture.lock = removeDependencyFromJSON(t, fixture.lock, "peerDependencies", "vue", "^3.5.0")
	fixture.write(t)

	_, err := InspectAdminFrontend(fixture.input())
	if !errors.Is(err, ErrInvalidAdminFrontend) {
		t.Fatalf("expected incomplete host peer catalog error, got %v", err)
	}
}

func TestInspectAdminFrontendRejectsInvalidFrozenLock(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*frontendFixture)
	}{
		{name: "unsupported version", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `"lockfileVersion": 1`, `"lockfileVersion": 2`, 1)
		}},
		{name: "unsupported config version", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `"configVersion": 1`, `"configVersion": 2`, 1)
		}},
		{name: "missing config version", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `"configVersion": 1,`, ``, 1)
		}},
		{name: "missing root workspace", mutate: func(f *frontendFixture) { f.lock = strings.Replace(f.lock, `"": {`, `"plugin": {`, 1) }},
		{name: "direct declaration mismatch", mutate: func(f *frontendFixture) { f.lock = strings.Replace(f.lock, `"zeta": "^2.0.0"`, `"zeta": "^3.0.0"`, 1) }},
		{name: "malformed package tuple", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, zetaLockTuple(validTestIntegrity), `{"resolution":"zeta@2.1.0"}`, 1)
		}},
		{name: "unpinned registry resolution", mutate: func(f *frontendFixture) { f.lock = strings.Replace(f.lock, `zeta@2.1.0`, `zeta@^2.0.0`, 1) }},
		{name: "missing registry integrity", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, zetaLockTuple(validTestIntegrity), `["zeta@2.1.0", "", {"dependencies":{"@scope/nested":"^3.0.0"}}]`, 1)
		}},
		{name: "invalid registry integrity", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, zetaLockTuple(validTestIntegrity), zetaLockTuple("sha512-not-base64"), 1)
		}},
		{name: "missing direct resolution", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `"zeta": `+zetaLockTuple(validTestIntegrity)+`,`, ``, 1)
		}},
		{name: "direct resolution outside declared range", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `zeta@2.1.0`, `zeta@1.9.0`, 1)
		}},
		{name: "extra local workspace", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `"workspaces": {`, `"workspaces": {"packages/local":{"name":"local"},`, 1)
		}},
		{name: "local tuple source", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, zetaLockTuple(validTestIntegrity), `["zeta@2.1.0", "file:../local", {"dependencies":{"@scope/nested":"^3.0.0"}}]`, 1)
		}},
		{name: "remote tuple source", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, zetaLockTuple(validTestIntegrity), `["zeta@2.1.0", "https://example.com/zeta.tgz", {"dependencies":{"@scope/nested":"^3.0.0"}}, "`+validTestIntegrity+`"]`, 1)
		}},
		{name: "transitive local dependency metadata", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, zetaLockTuple(validTestIntegrity), `["zeta@2.1.0", "", {"dependencies":{"@scope/nested":"^3.0.0","unsafe":"link:../local"}}, "`+validTestIntegrity+`"]`, 1)
		}},
		{name: "missing transitive resolution", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `"zeta/@scope/nested": ["@scope/nested@3.0.1", "", {}, "`+validTestIntegrity+`"],`, ``, 1)
		}},
		{name: "transitive resolution outside declared range", mutate: func(f *frontendFixture) {
			f.lock = strings.Replace(f.lock, `@scope/nested@3.0.1`, `@scope/nested@2.9.0`, 1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFrontendFixture(t)
			test.mutate(fixture)
			fixture.write(t)
			_, err := InspectAdminFrontend(fixture.input())
			if !errors.Is(err, ErrInvalidAdminFrontend) {
				t.Fatalf("expected invalid lock error, got %v", err)
			}
		})
	}
}

func TestResolveBunDependencySupportsScopedParentsAndRootHoisting(t *testing.T) {
	packages := map[string]lockedPackage{
		"@scope/parent": {
			Dependency: Dependency{Name: "@scope/parent", Version: "1.0.0"},
		},
		"@scope/parent/@scope/child": {
			Dependency: Dependency{Name: "@scope/child", Version: "2.0.0"},
		},
		"hoisted": {
			Dependency: Dependency{Name: "hoisted", Version: "3.0.0"},
		},
	}

	tests := []struct {
		name       string
		dependency string
		version    string
	}{
		{name: "scoped nested dependency", dependency: "@scope/child", version: "2.0.0"},
		{name: "root hoisted dependency", dependency: "hoisted", version: "3.0.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved, exists, err := resolveBunDependency(packages, "@scope/parent", test.dependency)
			if err != nil || !exists {
				t.Fatalf("resolve dependency %s: exists=%t err=%v", test.dependency, exists, err)
			}
			if resolved.Dependency.Name != test.dependency || resolved.Dependency.Version != test.version {
				t.Fatalf("unexpected resolution: %#v", resolved.Dependency)
			}
		})
	}
}

func TestInspectAdminFrontendRejectsHostBuildAndServerInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*frontendFixture)
	}{
		{name: "workspaces", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"workspaces":["packages/*"]`)
		}},
		{name: "trusted dependencies", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"trustedDependencies":["native-addon"]`)
		}},
		{name: "patched dependencies", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"patchedDependencies":{"pkg@1.0.0":"patches/pkg.patch"}`)
		}},
		{name: "nuxt package config", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"nuxt":{"modules":["unsafe"]}`)
		}},
		{name: "vite package config", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"vite":{"plugins":["unsafe"]}`)
		}},
		{name: "nitro package config", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"nitro":{"plugins":["unsafe"]}`)
		}},
		{name: "server package config", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"server":{"routes":["unsafe"]}`)
		}},
		{name: "route rules", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"routeRules":{"/**":{"headers":{"content-security-policy":"*"}}}`)
		}},
		{name: "csp", mutate: func(f *frontendFixture) {
			f.packageJSON = addTopLevelPackageField(f.packageJSON, `"csp":{"connect-src":["*"]}`)
		}},
		{name: "install hook", mutate: func(f *frontendFixture) {
			f.packageJSON = replacePackageScripts(f.packageJSON, `{"postinstall":"node install.mjs"}`)
		}},
		{name: "build hook", mutate: func(f *frontendFixture) {
			f.packageJSON = replacePackageScripts(f.packageJSON, `{"build":"nuxt build"}`)
		}},
		{name: "nuxt config file", mutate: func(f *frontendFixture) { f.extraFiles["nuxt.config.ts"] = `export default {}` }},
		{name: "vite config file", mutate: func(f *frontendFixture) { f.extraFiles["config/vite.config.mjs"] = `export default {}` }},
		{name: "nitro config file", mutate: func(f *frontendFixture) { f.extraFiles["nitro.config.ts"] = `export default {}` }},
		{name: "bun config file", mutate: func(f *frontendFixture) { f.extraFiles["bunfig.toml"] = `[install]` }},
		{name: "npm registry config", mutate: func(f *frontendFixture) { f.extraFiles[".npmrc"] = `registry=https://example.com/` }},
		{name: "server route", mutate: func(f *frontendFixture) { f.extraFiles["server/api/unsafe.get.ts"] = `export default () => ({})` }},
		{name: "nuxt plugin", mutate: func(f *frontendFixture) {
			f.extraFiles["plugins/unsafe.ts"] = `export default defineNuxtPlugin(() => {})`
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFrontendFixture(t)
			test.mutate(fixture)
			fixture.write(t)
			_, err := InspectAdminFrontend(fixture.input())
			if !errors.Is(err, ErrInvalidAdminFrontend) {
				t.Fatalf("expected forbidden host input error, got %v", err)
			}
		})
	}
}

type frontendFixture struct {
	packageRoot  string
	root         string
	components   map[string]string
	locales      map[string]string
	hostPeers    HostPeers
	packageJSON  string
	lock         string
	localeBodies map[string]string
	extraFiles   map[string]string
	omit         map[string]bool
}

func newFrontendFixture(t *testing.T) *frontendFixture {
	t.Helper()
	fixture := &frontendFixture{
		packageRoot: t.TempDir(),
		root:        "frontend/admin",
		components:  map[string]string{"job-cell": "components/JobCell.vue"},
		locales:     map[string]string{"zh-CN": "locales/zh-CN.json", "en-US": "locales/en-US.json"},
		hostPeers: HostPeers{
			"vue":               "3.5.39",
			"nuxt":              "4.4.8",
			"@nuxt/ui":          "4.9.0",
			"vue-router":        "5.1.0",
			"@sforum/admin-sdk": "1.0.0",
		},
		packageJSON: `{
  "name": "fixture-admin",
  "private": true,
  "scripts": {},
  "dependencies": {
    "zeta": "^2.0.0",
    "@scope/alpha": "~1.2.0"
  },
  "devDependencies": {
    "builder": "1.x"
  },
  "optionalDependencies": {
    "optional-pkg": "^0.5.0"
  },
  "peerDependencies": {
    "vue": "^3.5.0",
    "nuxt": "^4.4.0",
    "@nuxt/ui": "^4.9.0",
    "vue-router": "^5.0.0",
    "@sforum/admin-sdk": "^1.0.0"
  }
}`,
		lock: `{
  // Bun's text lock is HuJSON, not strict JSON.
  "lockfileVersion": 1,
  "configVersion": 1,
  "workspaces": {
    "": {
      "name": "fixture-admin",
      "dependencies": {
        "zeta": "^2.0.0",
        "@scope/alpha": "~1.2.0",
      },
      "devDependencies": {
        "builder": "1.x",
      },
      "optionalDependencies": {
        "optional-pkg": "^0.5.0",
      },
      "peerDependencies": {
        "vue": "^3.5.0",
        "nuxt": "^4.4.0",
        "@nuxt/ui": "^4.9.0",
        "vue-router": "^5.0.0",
        "@sforum/admin-sdk": "^1.0.0",
      },
    },
  },
  "packages": {
	    "@scope/alpha": ["@scope/alpha@1.2.4", "", {}, "sha512-alpha"],
	    "zeta/@scope/nested": ["@scope/nested@3.0.1", "", {}, "sha512-nested"],
    "optional-pkg": ["optional-pkg@0.5.3", "", {}, "sha512-optional"],
    "builder": ["builder@1.9.0", "", {}, "sha512-builder"],
	    "zeta": ["zeta@2.1.0", "", {"dependencies":{"@scope/nested":"^3.0.0"}}, "sha512-zeta"],
  },
}`,
		localeBodies: map[string]string{
			"zh-CN": `{"table":{"latency":"延迟"}}`,
			"en-US": `{"table":{"latency":"Latency"}}`,
		},
		extraFiles: map[string]string{},
		omit:       map[string]bool{},
	}
	fixture.lock = strings.NewReplacer(
		"sha512-alpha", validTestIntegrity,
		"sha512-nested", validTestIntegrity,
		"sha512-builder", validTestIntegrity,
		"sha512-optional", validTestIntegrity,
		"sha512-zeta", validTestIntegrity,
	).Replace(fixture.lock)
	return fixture
}

func zetaLockTuple(integrity string) string {
	return `["zeta@2.1.0", "", {"dependencies":{"@scope/nested":"^3.0.0"}}, "` + integrity + `"]`
}

func (f *frontendFixture) frontendRoot() string {
	return filepath.Join(f.packageRoot, filepath.FromSlash(f.root))
}

func (f *frontendFixture) input() FrontendInspectInput {
	return FrontendInspectInput{
		PackageRoot: f.packageRoot,
		Root:        f.root,
		Components:  f.components,
		Locales:     f.locales,
		HostPeers:   f.hostPeers,
	}
}

func (f *frontendFixture) write(t *testing.T) {
	t.Helper()
	root := f.frontendRoot()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create frontend root: %v", err)
	}
	files := map[string]string{
		"package.json":           f.packageJSON,
		"bun.lock":               f.lock,
		"components/JobCell.vue": `<template><span>ready</span></template>`,
		"locales/zh-CN.json":     f.localeBodies["zh-CN"],
		"locales/en-US.json":     f.localeBodies["en-US"],
	}
	for path, body := range f.extraFiles {
		files[path] = body
	}
	for relative, body := range files {
		if f.omit[relative] {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create parent for %s: %v", relative, err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", relative, err)
		}
	}
}

func addTopLevelPackageField(body string, field string) string {
	return strings.Replace(body, "{", "{\n  "+field+",", 1)
}

func replacePackageScripts(body string, scripts string) string {
	return strings.Replace(body, `"scripts": {}`, `"scripts": `+scripts, 1)
}

func addPackageDependency(t *testing.T, body string, field string, name string, version string) string {
	t.Helper()
	needle := `"` + field + `": {`
	if !strings.Contains(body, needle) {
		t.Fatalf("fixture package field %s missing", field)
	}
	return strings.Replace(body, needle, needle+`"`+name+`":"`+version+`",`, 1)
}

func addWorkspaceDependency(t *testing.T, body string, field string, name string, version string) string {
	t.Helper()
	needle := `"` + field + `": {`
	if !strings.Contains(body, needle) {
		t.Fatalf("fixture lock field %s missing", field)
	}
	return strings.Replace(body, needle, needle+`"`+name+`":"`+version+`",`, 1)
}

func movePackageDependency(t *testing.T, body string, name string, from string, to string) string {
	t.Helper()
	version := dependencyVersionFromJSON(t, body, from, name)
	body = removeDependencyFromJSON(t, body, from, name, version)
	return addPackageDependency(t, body, to, name, version)
}

func moveWorkspaceDependency(t *testing.T, body string, name string, from string, to string) string {
	t.Helper()
	version := dependencyVersionFromJSON(t, body, from, name)
	body = removeDependencyFromJSON(t, body, from, name, version)
	return addWorkspaceDependency(t, body, to, name, version)
}

func replaceDependencyVersion(t *testing.T, body string, field string, name string, version string) string {
	t.Helper()
	current := dependencyVersionFromJSON(t, body, field, name)
	return strings.Replace(body, `"`+name+`": "`+current+`"`, `"`+name+`": "`+version+`"`, 1)
}

func replaceWorkspaceDependencyVersion(t *testing.T, body string, field string, name string, version string) string {
	t.Helper()
	return replaceDependencyVersion(t, body, field, name, version)
}

func dependencyVersionFromJSON(t *testing.T, body string, field string, name string) string {
	t.Helper()
	fieldAt := strings.Index(body, `"`+field+`": {`)
	if fieldAt < 0 {
		t.Fatalf("fixture field %s missing", field)
	}
	remainder := body[fieldAt:]
	nameAt := strings.Index(remainder, `"`+name+`": "`)
	if nameAt < 0 {
		t.Fatalf("fixture dependency %s missing from %s", name, field)
	}
	value := remainder[nameAt+len(name)+5:]
	end := strings.Index(value, `"`)
	if end < 0 {
		t.Fatalf("fixture dependency %s has no closing quote", name)
	}
	return value[:end]
}

func removeDependencyFromJSON(t *testing.T, body string, field string, name string, version string) string {
	t.Helper()
	withComma := `"` + name + `": "` + version + `",`
	withoutComma := `"` + name + `": "` + version + `"`
	if strings.Contains(body, withComma) {
		return strings.Replace(body, withComma, "", 1)
	}
	if strings.Contains(body, withoutComma) {
		return strings.Replace(body, withoutComma, "", 1)
	}
	t.Fatalf("fixture dependency %s missing", name)
	return body
}
