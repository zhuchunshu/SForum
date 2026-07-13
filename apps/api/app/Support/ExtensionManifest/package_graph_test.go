package extensionmanifest

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestResolvePackageGraphDeterministicOrder(t *testing.T) {
	base := graphManifest("graph.base")
	consumer := graphManifest("graph.consumer")
	consumer.Dependencies = []ManifestDependency{{ID: base.ID, Version: "^1.0.0", Kind: "required"}}
	provider := graphManifest("graph.provider")
	provider.Dependencies = []ManifestDependency{{Capability: "platform.search", Version: "2.0.0", Kind: "provides"}}
	virtual := graphManifest("graph.virtual")
	virtual.Dependencies = []ManifestDependency{{Capability: "platform.search", Version: "^2.0.0", Kind: "required"}}

	graph, err := ResolvePackageGraph([]Manifest{virtual, provider, consumer, base})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"graph.base", "graph.consumer", "graph.provider", "graph.virtual"}
	if !reflect.DeepEqual(graph.Order, want) {
		t.Fatalf("order = %#v, want %#v", graph.Order, want)
	}
	if !reflect.DeepEqual(graph.Providers["platform.search"], []string{"graph.provider"}) {
		t.Fatalf("providers = %#v", graph.Providers)
	}
}

func TestResolvePackageGraphOptionalDependencyCanBeAbsentOrIncompatible(t *testing.T) {
	consumer := graphManifest("graph.optional")
	consumer.Dependencies = []ManifestDependency{{ID: "graph.missing", Version: "^2.0.0", Kind: "optional"}}
	if _, err := ResolvePackageGraph([]Manifest{consumer}); err != nil {
		t.Fatalf("missing optional dependency should not fail: %v", err)
	}
	old := graphManifest("graph.missing")
	old.Version = "1.0.0"
	if _, err := ResolvePackageGraph([]Manifest{consumer, old}); err != nil {
		t.Fatalf("incompatible optional dependency should be ignored: %v", err)
	}
}

func TestResolvePackageGraphFailures(t *testing.T) {
	tests := []struct {
		name      string
		manifests func() []Manifest
		want      error
	}{
		{name: "missing", want: ErrDependencyMissing, manifests: func() []Manifest {
			consumer := graphManifest("graph.consumer")
			consumer.Dependencies = []ManifestDependency{{ID: "graph.missing", Version: "^1.0.0", Kind: "required"}}
			return []Manifest{consumer}
		}},
		{name: "version", want: ErrDependencyVersion, manifests: func() []Manifest {
			base := graphManifest("graph.base")
			consumer := graphManifest("graph.consumer")
			consumer.Dependencies = []ManifestDependency{{ID: base.ID, Version: "^2.0.0", Kind: "required"}}
			return []Manifest{base, consumer}
		}},
		{name: "conflict", want: ErrDependencyConflict, manifests: func() []Manifest {
			base := graphManifest("graph.base")
			consumer := graphManifest("graph.consumer")
			consumer.Dependencies = []ManifestDependency{{ID: base.ID, Version: "^1.0.0", Kind: "conflict"}}
			return []Manifest{base, consumer}
		}},
		{name: "cycle", want: ErrDependencyCycle, manifests: func() []Manifest {
			left := graphManifest("graph.left")
			right := graphManifest("graph.right")
			left.Dependencies = []ManifestDependency{{ID: right.ID, Version: "^1.0.0", Kind: "required"}}
			right.Dependencies = []ManifestDependency{{ID: left.ID, Version: "^1.0.0", Kind: "required"}}
			return []Manifest{left, right}
		}},
		{name: "ambiguous capability", want: ErrDependencyAmbiguous, manifests: func() []Manifest {
			first := graphManifest("graph.first")
			first.Dependencies = []ManifestDependency{{Capability: "platform.search", Version: "1.0.0", Kind: "provides"}}
			second := graphManifest("graph.second")
			second.Dependencies = []ManifestDependency{{Capability: "platform.search", Version: "1.1.0", Kind: "provides"}}
			consumer := graphManifest("graph.consumer")
			consumer.Dependencies = []ManifestDependency{{Capability: "platform.search", Version: "^1.0.0", Kind: "required"}}
			return []Manifest{first, second, consumer}
		}},
		{name: "duplicate extension", want: ErrDuplicateExtension, manifests: func() []Manifest {
			return []Manifest{graphManifest("graph.same"), graphManifest("graph.same")}
		}},
		{name: "ambiguous route replacement", want: ErrManifestSetConflict, manifests: func() []Manifest {
			return []Manifest{graphReplaceManifest("graph.replace-one"), graphReplaceManifest("graph.replace-two")}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolvePackageGraph(test.manifests())
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func graphManifest(id string) Manifest {
	manifest := versionedTestManifest(ManifestVersionV3)
	manifest.ID = id
	manifest.Name = id
	manifest.Version = "1.0.0"
	return manifest
}

func graphReplaceManifest(id string) Manifest {
	digest := strings.Repeat("c", 64)
	manifest := graphManifest(id)
	manifest.Backend = ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1, Digest: digest}
	manifest.PackageFiles = []ManifestPackageFile{{ID: id + ".file.backend", Kind: "executable", Path: "backend/plugin", Digest: digest}}
	manifest.Routes = []ManifestRoute{{
		ID: id + ".route.replace", ContractVersion: id + ".route.replace@1",
		Action: RouteActionReplace, TargetID: "core.route.forum.home", Path: "/api/v1/topics", Methods: []string{"POST"},
		Guard: GuardCorePublic, Mode: RouteModeHTTP, Handler: "route.replace",
		RequestSchema: id + ".route.request@1", ResponseSchema: id + ".route.response@1", Fallback: "closed",
	}}
	return manifest
}
