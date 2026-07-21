package compatfarm

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	apilts "github.com/zhuchunshu/sforum/apps/api/app/Support/APILTS"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Support/CompatFarm -> repo root
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func TestLoadMatrixAndDeprecatedShimTelemetry(t *testing.T) {
	root := repoRoot(t)
	matrixPath := filepath.Join(root, "tests", "compat", "matrix.yaml")
	matrix, err := LoadMatrix(matrixPath)
	if err != nil {
		t.Fatal(err)
	}
	if matrix.Schema != SchemaVersion || len(matrix.RequiredCells()) < 1 {
		t.Fatalf("matrix = %#v", matrix)
	}
	deprecated := matrix.DeprecatedCells()
	if len(deprecated) == 0 {
		t.Fatal("expected deprecated protocol-v1 cell")
	}
	lts := apilts.New()
	for _, cell := range deprecated {
		if cell.ExpectsShimTelemetry {
			lts.RecordShimCall(apilts.ProtocolV1ContractID)
		}
	}
	snap := lts.Snapshot()
	found := false
	for _, row := range snap.ShimUsage {
		if row.ContractID == apilts.ProtocolV1ContractID && row.Calls > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("shim telemetry missing: %#v", snap.ShimUsage)
	}
}

func TestRunMatrixGatePassesRequiredAndDeprecated(t *testing.T) {
	root := repoRoot(t)
	matrixPath := filepath.Join(root, "tests", "compat", "matrix.yaml")
	result, err := RunMatrix(context.Background(), matrixPath, RunOptions{
		RepoRoot: root,
		LTS:      apilts.New(),
		Now:      func() time.Time { return time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("farm not ok: %#v", result.Cells)
	}
	if len(result.Cells) < 3 {
		t.Fatalf("expected >=3 cells, got %d", len(result.Cells))
	}
	for _, cell := range result.Cells {
		if cell.Outcome != OutcomePass {
			t.Fatalf("cell %s outcome=%s msg=%s", cell.ID, cell.Outcome, cell.Message)
		}
		if cell.ID == "deprecated-protocol-v1-shim" && cell.ShimCalls == 0 {
			t.Fatal("deprecated cell must prove shim telemetry")
		}
	}
}

func TestRunMatrixMissingFileFailsGate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-matrix.yaml")
	result, err := RunMatrix(context.Background(), path, RunOptions{RepoRoot: repoRoot(t)})
	if err == nil {
		t.Fatal("expected error for missing matrix")
	}
	if result.OK {
		t.Fatal("missing matrix must not be OK")
	}
	if len(result.Cells) == 0 || result.Cells[0].Outcome != OutcomeMissing {
		t.Fatalf("want missing outcome: %#v", result.Cells)
	}
}

func TestRunMatrixRejectsSkipAsGateFailure(t *testing.T) {
	// 未知 protocol/manifest 组合必须 skip 并失败门禁，不得静默通过。
	dir := t.TempDir()
	path := filepath.Join(dir, "matrix.yaml")
	body := `version: 1
schema: sforum.compat-farm@1
cells:
  - id: unknown-combo
    sforum: current
    protocol: v9
    manifest: v99
    database: none
    browser: none
    status: required
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := RunMatrix(context.Background(), path, RunOptions{RepoRoot: repoRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatal("skipped required cell must fail gate")
	}
	if result.Cells[0].Outcome != OutcomeSkip {
		t.Fatalf("outcome = %s", result.Cells[0].Outcome)
	}
}
