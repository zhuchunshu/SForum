// Package componentcatalog owns the neutral Host catalog of stable public/admin component ids.
package componentcatalog

import (
	"fmt"
	"regexp"
	"strings"
)

type Kind string

const (
	KindPage      Kind = "page"
	KindComponent Kind = "component"
)

type Owner string

const (
	OwnerPublic Owner = "public"
	OwnerAdmin  Owner = "admin"
)

// CoreComponent is one immutable Host target exposed to Manifest V3 composition.
type CoreComponent struct {
	ID              string  `json:"id"`
	ContractVersion string  `json:"contractVersion"`
	Kind            Kind    `json:"kind"`
	Owners          []Owner `json:"owners"`
	Route           string  `json:"route,omitempty"`
	Source          string  `json:"source"`
}

var (
	coreComponentIDPattern       = regexp.MustCompile(`^core\.component\.[a-z0-9][a-z0-9._-]*$`)
	coreComponentContractPattern = regexp.MustCompile(`^sforum\.component\.[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
)

// CoreComponentCatalog returns a caller-owned copy of the reviewed Host catalog.
func CoreComponentCatalog() []CoreComponent {
	result := append([]CoreComponent(nil), generatedCoreComponentCatalog[:]...)
	for index := range result {
		result[index].Owners = append([]Owner(nil), result[index].Owners...)
	}
	return result
}

// FindCoreComponent resolves an exact stable Host id and returns a detached value.
func FindCoreComponent(id string) (CoreComponent, bool) {
	id = strings.TrimSpace(id)
	for _, item := range generatedCoreComponentCatalog {
		if item.ID != id {
			continue
		}
		item.Owners = append([]Owner(nil), item.Owners...)
		return item, true
	}
	return CoreComponent{}, false
}

// OwnedBy reports whether this target is available on the requested Host surface.
func (component CoreComponent) OwnedBy(owner Owner) bool {
	for _, candidate := range component.Owners {
		if candidate == owner {
			return true
		}
	}
	return false
}

func validateCoreComponentCatalog(catalog []CoreComponent) error {
	ids := make(map[string]struct{}, len(catalog))
	contracts := make(map[string]struct{}, len(catalog))
	sources := make(map[string]struct{}, len(catalog))
	for _, item := range catalog {
		if !coreComponentIDPattern.MatchString(item.ID) {
			return fmt.Errorf("components: invalid core id %q", item.ID)
		}
		if _, exists := ids[item.ID]; exists {
			return fmt.Errorf("components: duplicate core id %q", item.ID)
		}
		ids[item.ID] = struct{}{}
		if !coreComponentContractPattern.MatchString(item.ContractVersion) {
			return fmt.Errorf("components: invalid contract %q", item.ContractVersion)
		}
		if _, exists := contracts[item.ContractVersion]; exists {
			return fmt.Errorf("components: duplicate contract %q", item.ContractVersion)
		}
		contracts[item.ContractVersion] = struct{}{}
		if !strings.HasPrefix(item.Source, "apps/web/app/") || !strings.HasSuffix(item.Source, ".vue") {
			return fmt.Errorf("components: invalid source %q", item.Source)
		}
		if _, exists := sources[item.Source]; exists {
			return fmt.Errorf("components: duplicate source %q", item.Source)
		}
		sources[item.Source] = struct{}{}
		if item.Kind != KindPage && item.Kind != KindComponent {
			return fmt.Errorf("components: %s has invalid kind %q", item.ID, item.Kind)
		}
		if len(item.Owners) == 0 || len(item.Owners) > 2 {
			return fmt.Errorf("components: %s needs explicit ownership", item.ID)
		}
		previous := -1
		for _, owner := range item.Owners {
			rank := -1
			switch owner {
			case OwnerPublic:
				rank = 0
			case OwnerAdmin:
				rank = 1
			default:
				return fmt.Errorf("components: %s has invalid owner %q", item.ID, owner)
			}
			if rank <= previous {
				return fmt.Errorf("components: %s owners are duplicate or non-canonical", item.ID)
			}
			previous = rank
		}
		if item.Kind == KindPage && (len(item.Owners) != 1 || item.Route == "") {
			return fmt.Errorf("components: page %s needs one owner and one route", item.ID)
		}
		if item.Kind == KindComponent && item.Route != "" {
			return fmt.Errorf("components: component %s cannot own a page route", item.ID)
		}
	}
	return nil
}
