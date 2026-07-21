package marketplace

// cloneIndex deep-copies Index including Entries, Dependencies, and Notices.
// Callers must not share nested slices with the internal snapshot.
func cloneIndex(index Index) Index {
	out := index
	if index.Entries != nil {
		out.Entries = make([]Entry, len(index.Entries))
		for i := range index.Entries {
			out.Entries[i] = cloneEntry(index.Entries[i])
		}
	}
	return out
}

func cloneEntry(entry Entry) Entry {
	out := entry
	if entry.Dependencies != nil {
		out.Dependencies = append([]DependencyConstraint(nil), entry.Dependencies...)
	}
	if entry.Notices != nil {
		out.Notices = append([]Notice(nil), entry.Notices...)
	}
	return out
}

func cloneEntries(entries []Entry) []Entry {
	if entries == nil {
		return nil
	}
	out := make([]Entry, len(entries))
	for i := range entries {
		out[i] = cloneEntry(entries[i])
	}
	return out
}

func cloneResolveResult(result ResolveResult) ResolveResult {
	out := result
	if result.Order != nil {
		out.Order = append([]PlanStep(nil), result.Order...)
	}
	out.Report.Warnings = append([]string(nil), result.Report.Warnings...)
	out.Report.BlockedBy = append([]string(nil), result.Report.BlockedBy...)
	return out
}

func cloneInstallPlan(plan InstallPlan) InstallPlan {
	out := plan
	out.ResolveResult = cloneResolveResult(plan.ResolveResult)
	return out
}
