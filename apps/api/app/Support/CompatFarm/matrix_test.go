package compatfarm

import (
	"path/filepath"
	"runtime"
	"testing"

	apilts "github.com/zhuchunshu/sforum/apps/api/app/Support/APILTS"
)

func TestLoadMatrixAndDeprecatedShimTelemetry(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Support/CompatFarm -> repo root tests/compat/matrix.yaml
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
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
	// LTS shim telemetry must be recordable for deprecated fixtures before removal.
	lts := apilts.New()
	for _, cell := range deprecated {
		if cell.ExpectsShimTelemetry {
			lts.RecordShimCall("sforum.protocol.v1")
		}
	}
	snap := lts.Snapshot()
	found := false
	for _, row := range snap.ShimUsage {
		if row.ContractID == "sforum.protocol.v1" && row.Calls > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("shim telemetry missing: %#v", snap.ShimUsage)
	}
}
