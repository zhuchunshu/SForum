package pages

import (
	"fmt"
	"strings"
)

type contributionSource uint8

const (
	contributionSourceExtension contributionSource = iota
	contributionSourceTheme
)

// prepareContributions 校验并规范化插件贡献列表（不修改 Registry）。
func prepareContributions(extensionID string, items []PageContribution) ([]PageContribution, error) {
	return prepareContributionsFor(extensionID, items, contributionSourceExtension)
}

func prepareThemeContributions(extensionID string, items []PageContribution) ([]PageContribution, error) {
	return prepareContributionsFor(extensionID, items, contributionSourceTheme)
}

func prepareContributionsFor(extensionID string, items []PageContribution, source contributionSource) ([]PageContribution, error) {
	prepared := make([]PageContribution, 0, len(items))
	seenAddSigs := map[string]struct{}{}
	for _, item := range items {
		item.ExtensionID = extensionID
		access, err := NormalizeAccess(string(item.Access))
		if err != nil {
			return nil, err
		}
		item.Access = access
		if access == AccessPermission && strings.TrimSpace(item.Permission) == "" {
			return nil, fmt.Errorf("%w: permission key required for access=permission", ErrInvalidAccess)
		}
		if err := validateContribution(item); err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.DataRoute) != "" {
			if err := ValidateDataRoute(item.DataRoute); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidContribution, err)
			}
			if item.DataSource == "" {
				item.DataSource = "plugin"
			}
		}
		switch item.Action {
		case ActionReplace:
			if err := validateReplacementTarget(&item, source); err != nil {
				return nil, err
			}
		case ActionAdd:
			sig, err := prepareAddedRoute(&item, seenAddSigs)
			if err != nil {
				return nil, err
			}
			seenAddSigs[sig] = struct{}{}
		default:
			return nil, fmt.Errorf("%w: action %q", ErrInvalidContribution, item.Action)
		}
		prepared = append(prepared, item)
	}
	return prepared, nil
}

func validateReplacementTarget(item *PageContribution, source contributionSource) error {
	target := strings.TrimSpace(item.Target)
	page, ok := Find(target)
	if !ok {
		return fmt.Errorf("%w: target %q", ErrUnknownPage, target)
	}
	if source == contributionSourceTheme {
		if !page.Themeable {
			return fmt.Errorf("%w: %s", ErrNotThemeable, target)
		}
	} else if !page.Replaceable {
		return fmt.Errorf("%w: %s", ErrNotReplaceable, target)
	}
	if strings.TrimSpace(item.Contract) == "" {
		return fmt.Errorf("%w: replace requires contract", ErrInvalidContribution)
	}
	if page.ContractVersion != "" && item.Contract != page.ContractVersion {
		return fmt.Errorf("%w: contribution contract %q != core %q", ErrContractMismatch, item.Contract, page.ContractVersion)
	}
	return nil
}

func prepareAddedRoute(item *PageContribution, seen map[string]struct{}) (string, error) {
	path := normalizePublicPath(item.Path)
	item.Path = path
	if IsReservedPath(path) {
		return "", fmt.Errorf("%w: %s", ErrReservedPath, path)
	}
	if _, ok := MatchPath(path); ok {
		return "", fmt.Errorf("%w: path %s collides with core page", ErrConflictProvider, path)
	}
	sig, err := CanonicalRouteSignature(path)
	if err != nil {
		return "", err
	}
	item.RouteSignature = sig
	for _, page := range Catalog() {
		if page.PathPattern == "" {
			continue
		}
		coreSig, err := CanonicalRouteSignature(page.PathPattern)
		if err == nil && signaturesConflict(sig, coreSig) {
			return "", fmt.Errorf("%w: path signature %s collides with core page %s", ErrConflictProvider, sig, page.ID)
		}
	}
	if _, ok := seen[sig]; ok {
		return "", fmt.Errorf("%w: duplicate add signature %s", ErrInvalidContribution, sig)
	}
	if _, err := CompileRoute(path, *item); err != nil {
		return "", err
	}
	return sig, nil
}
