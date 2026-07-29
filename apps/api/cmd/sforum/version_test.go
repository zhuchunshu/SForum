package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommandPrintsBuildSummary(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		cmd := newRootCommand()
		cmd.SetArgs(args)
		var output bytes.Buffer
		cmd.SetOut(&output)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute %v: %v", args, err)
		}
		if got := strings.TrimSpace(output.String()); !strings.HasPrefix(got, "SForum ") || strings.HasPrefix(got, "sforum version") {
			t.Fatalf("unexpected version output for %v: %q", args, got)
		}
	}
}
