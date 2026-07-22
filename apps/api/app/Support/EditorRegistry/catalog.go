package editorregistry

import (
	"sort"
	"strings"
)

// CatalogSchemaVersion is the Host-owned editor catalog contract for SFEditor.
const CatalogSchemaVersion = "sforum.editor-catalog@1"

// CatalogModule is one package-digest-bound prebuilt editor L2 module plus the
// declarations that load through it. Toolbar chrome is listed separately when
// it references a command shipped by this module.
type CatalogModule struct {
	ExtensionID      string         `json:"extensionId"`
	ExtensionVersion string         `json:"extensionVersion"`
	PackageDigest    string         `json:"packageDigest"`
	L2Module         string         `json:"l2Module"`
	L2Digest         string         `json:"l2Digest"`
	// AssetPath is the Host package-serve path under exact package digest.
	AssetPath string         `json:"assetPath"`
	Nodes     []Contribution `json:"nodes,omitempty"`
	Marks     []Contribution `json:"marks,omitempty"`
	Commands  []Contribution `json:"commands,omitempty"`
	Toolbars  []Contribution `json:"toolbars,omitempty"`
}

// Catalog is the inspectable, deterministic editor surface for trusted L2 load.
type Catalog struct {
	SchemaVersion string          `json:"schemaVersion"`
	Revision      uint64          `json:"revision"`
	Digest        string          `json:"digest"`
	SafeMode      bool            `json:"safeMode,omitempty"`
	Modules       []CatalogModule `json:"modules"`
	Toolbars      []Contribution  `json:"toolbars"`
}

// BuildCatalog projects the active Editor Registry graph into a loadable catalog.
// Safe Mode still returns core modules only (third-party already filtered).
func (r *Registry) BuildCatalog() Catalog {
	if r == nil {
		return Catalog{SchemaVersion: CatalogSchemaVersion}
	}
	snapshot := r.Snapshot()
	modules := map[string]*CatalogModule{}
	var toolbars []Contribution
	for _, contribution := range snapshot.Editor {
		switch contribution.Kind {
		case KindToolbar:
			toolbars = append(toolbars, contribution)
			continue
		case KindNode, KindMark, KindCommand:
		default:
			continue
		}
		if contribution.L2Module == "" || contribution.L2Digest == "" {
			// Command without dedicated module rides on a node/mark module;
			// still expose command metadata via toolbar resolution later.
			if contribution.Kind == KindCommand {
				// Attach command to any module owned by the same package that
				// already exists; otherwise create a metadata-only bucket.
				key := contribution.Artifact.PackageDigest + "\x00" + contribution.Artifact.ExtensionID
				module := modules[key]
				if module == nil {
					module = &CatalogModule{
						ExtensionID: contribution.Artifact.ExtensionID,
						ExtensionVersion: contribution.Artifact.ExtensionVersion,
						PackageDigest: contribution.Artifact.PackageDigest,
					}
					modules[key] = module
				}
				module.Commands = append(module.Commands, contribution)
			}
			continue
		}
		key := contribution.Artifact.PackageDigest + "\x00" + contribution.L2Module + "\x00" + contribution.L2Digest
		module := modules[key]
		if module == nil {
			module = &CatalogModule{
				ExtensionID: contribution.Artifact.ExtensionID,
				ExtensionVersion: contribution.Artifact.ExtensionVersion,
				PackageDigest: contribution.Artifact.PackageDigest,
				L2Module: contribution.L2Module,
				L2Digest: contribution.L2Digest,
				AssetPath: editorPackageAssetPath(contribution.Artifact.ExtensionID, contribution.Artifact.PackageDigest, contribution.L2Module),
			}
			modules[key] = module
		}
		switch contribution.Kind {
		case KindNode:
			module.Nodes = append(module.Nodes, contribution)
		case KindMark:
			module.Marks = append(module.Marks, contribution)
		case KindCommand:
			module.Commands = append(module.Commands, contribution)
		}
	}
	// Attach toolbars to modules whose package owns the referenced command.
	commandOwners := map[string]string{}
	for _, module := range modules {
		for _, command := range module.Commands {
			commandOwners[command.ID] = module.ExtensionID + "\x00" + module.PackageDigest + "\x00" + module.L2Module + "\x00" + module.L2Digest
		}
	}
	for _, toolbar := range toolbars {
		ownerKey := commandOwners[toolbar.CommandID]
		if ownerKey == "" {
			continue
		}
		// Prefer exact module key when L2 present.
		for key, module := range modules {
			if module.ExtensionID == toolbar.Artifact.ExtensionID &&
				module.PackageDigest == toolbar.Artifact.PackageDigest {
				// Match command owner when available.
				if ownerKey == module.ExtensionID+"\x00"+module.PackageDigest+"\x00"+module.L2Module+"\x00"+module.L2Digest ||
					strings.HasPrefix(ownerKey, module.ExtensionID+"\x00"+module.PackageDigest) {
					module.Toolbars = append(module.Toolbars, toolbar)
					_ = key
					break
				}
			}
		}
	}
	result := Catalog{
		SchemaVersion: CatalogSchemaVersion,
		Revision:      snapshot.Revision,
		Digest:        snapshot.Digest,
		SafeMode:      snapshot.SafeMode,
		Toolbars:      toolbars,
	}
	for _, module := range modules {
		sort.Slice(module.Nodes, func(i, j int) bool { return contributionBefore(module.Nodes[i], module.Nodes[j]) })
		sort.Slice(module.Marks, func(i, j int) bool { return contributionBefore(module.Marks[i], module.Marks[j]) })
		sort.Slice(module.Commands, func(i, j int) bool { return contributionBefore(module.Commands[i], module.Commands[j]) })
		sort.Slice(module.Toolbars, func(i, j int) bool { return contributionBefore(module.Toolbars[i], module.Toolbars[j]) })
		result.Modules = append(result.Modules, *module)
	}
	sort.Slice(result.Modules, func(i, j int) bool {
		if result.Modules[i].ExtensionID != result.Modules[j].ExtensionID {
			return result.Modules[i].ExtensionID < result.Modules[j].ExtensionID
		}
		if result.Modules[i].L2Module != result.Modules[j].L2Module {
			return result.Modules[i].L2Module < result.Modules[j].L2Module
		}
		return result.Modules[i].L2Digest < result.Modules[j].L2Digest
	})
	sort.Slice(result.Toolbars, func(i, j int) bool {
		return contributionBefore(result.Toolbars[i], result.Toolbars[j])
	})
	return result
}

func editorPackageAssetPath(extensionID, packageDigest, modulePath string) string {
	return "/_sforum/assets/extensions/" + extensionID + "/" + packageDigest + "/" + strings.TrimPrefix(modulePath, "/")
}
