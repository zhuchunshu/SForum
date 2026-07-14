package extensionmanifest

import (
	"sort"
	"strings"
)

const (
	DatabaseGrantOwnSchema    = "own_schema"
	DatabaseGrantCoreViews    = "core_views"
	DatabaseGrantHostCommands = "host_commands"
	DatabaseGrantRawCore      = "raw_core"
	DatabaseGrantKernel       = "kernel"
)

var databaseGrantTierOrder = []string{
	DatabaseGrantOwnSchema,
	DatabaseGrantCoreViews,
	DatabaseGrantHostCommands,
	DatabaseGrantRawCore,
	DatabaseGrantKernel,
}

// DatabaseGrants returns the canonical exact grant set. Legacy authority is
// cumulative so a higher tier includes every preceding tier.
func DatabaseGrants(database *ManifestDatabase) []string {
	if database == nil {
		return nil
	}
	if len(database.Grants) > 0 {
		grants := append([]string(nil), database.Grants...)
		for index := range grants {
			grants[index] = strings.ToLower(strings.TrimSpace(grants[index]))
		}
		sortDatabaseGrants(grants)
		return grants
	}
	authority := strings.ToLower(strings.TrimSpace(database.Authority))
	for index, grant := range databaseGrantTierOrder {
		if authority == grant {
			return append([]string(nil), databaseGrantTierOrder[:index+1]...)
		}
	}
	return nil
}

func HasDatabaseGrant(database *ManifestDatabase, grant string) bool {
	grant = strings.ToLower(strings.TrimSpace(grant))
	for _, candidate := range DatabaseGrants(database) {
		if candidate == grant {
			return true
		}
	}
	return false
}

func normalizeDatabaseGrants(database *ManifestDatabase) {
	database.Authority = strings.ToLower(strings.TrimSpace(database.Authority))
	for index := range database.Grants {
		database.Grants[index] = strings.ToLower(strings.TrimSpace(database.Grants[index]))
	}
	sortDatabaseGrants(database.Grants)
	if database.Authority == "" || len(database.Grants) > 0 {
		return
	}
	grants := DatabaseGrants(database)
	if len(grants) == 0 {
		return
	}
	database.Grants = grants
	database.Authority = ""
}

func sortDatabaseGrants(grants []string) {
	rank := func(value string) int {
		for index, grant := range databaseGrantTierOrder {
			if value == grant {
				return index
			}
		}
		return len(databaseGrantTierOrder)
	}
	sort.SliceStable(grants, func(i, j int) bool {
		left, right := rank(grants[i]), rank(grants[j])
		if left != right {
			return left < right
		}
		return grants[i] < grants[j]
	})
}
