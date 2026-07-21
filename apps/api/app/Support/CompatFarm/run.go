// Package compatfarm loads and executes the V3 P12 multi-version compatibility matrix.
package compatfarm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	apilts "github.com/zhuchunshu/sforum/apps/api/app/Support/APILTS"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

// CellOutcome is the machine status of one executed cell.
type CellOutcome string

const (
	OutcomePass    CellOutcome = "pass"
	OutcomeFail    CellOutcome = "fail"
	OutcomeSkip    CellOutcome = "skip"
	OutcomeMissing CellOutcome = "missing"
)

// CellResult is one cell's execution report.
type CellResult struct {
	ID       string      `json:"id"`
	Status   string      `json:"status"` // required|deprecated from matrix
	Outcome  CellOutcome `json:"outcome"`
	Duration time.Duration `json:"duration"`
	Message  string      `json:"message,omitempty"`
	// ShimCalls is set when expects_shim_telemetry was verified.
	ShimCalls uint64 `json:"shimCalls,omitempty"`
}

// RunResult is the full farm report.
type RunResult struct {
	SchemaVersion string       `json:"schemaVersion"`
	MatrixPath    string       `json:"matrixPath"`
	Cells         []CellResult `json:"cells"`
	// OK is true only when every required cell passed and no required cell was
	// skipped/missing/failed. Deprecated cells that fail also fail the gate.
	OK bool `json:"ok"`
}

// RunOptions controls farm execution.
type RunOptions struct {
	// RepoRoot is the monorepo root (contains tests/compat and extensions/).
	RepoRoot string
	// LTS is optional; when nil a fresh process registry is used for shim proof.
	LTS *apilts.Registry
	// BuiltinPluginPath overrides the default builtin plugin used for contracts.
	BuiltinPluginPath string
	// Now is injectable for tests.
	Now func() time.Time
}

// RunMatrix loads path and executes every cell. Missing matrix, empty required
// set, skip, or fail makes OK false (CI must fail).
func RunMatrix(ctx context.Context, matrixPath string, opts RunOptions) (RunResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	matrixPath = strings.TrimSpace(matrixPath)
	if matrixPath == "" {
		return RunResult{OK: false}, fmt.Errorf("compat farm: matrix path is required")
	}
	if _, err := os.Stat(matrixPath); err != nil {
		return RunResult{
			SchemaVersion: SchemaVersion,
			MatrixPath:    matrixPath,
			OK:            false,
			Cells: []CellResult{{
				ID: "_matrix", Outcome: OutcomeMissing,
				Message: fmt.Sprintf("matrix file missing: %v", err),
			}},
		}, fmt.Errorf("compat farm: matrix missing: %w", err)
	}
	matrix, err := LoadMatrix(matrixPath)
	if err != nil {
		return RunResult{SchemaVersion: SchemaVersion, MatrixPath: matrixPath, OK: false}, err
	}
	if len(matrix.RequiredCells()) == 0 {
		return RunResult{
			SchemaVersion: SchemaVersion, MatrixPath: matrixPath, OK: false,
			Cells: []CellResult{{
				ID: "_required", Outcome: OutcomeFail,
				Message: "matrix has zero required cells",
			}},
		}, fmt.Errorf("compat farm: zero required cells")
	}

	root := strings.TrimSpace(opts.RepoRoot)
	if root == "" {
		root = guessRepoRoot(matrixPath)
	}
	lts := opts.LTS
	if lts == nil {
		lts = apilts.New()
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	result := RunResult{
		SchemaVersion: SchemaVersion,
		MatrixPath:    matrixPath,
		OK:            true,
	}
	for _, cell := range matrix.Cells {
		if err := ctx.Err(); err != nil {
			result.OK = false
			result.Cells = append(result.Cells, CellResult{
				ID: cell.ID, Status: cell.Status, Outcome: OutcomeFail,
				Message: err.Error(),
			})
			return result, err
		}
		started := now()
		cr := executeCell(ctx, cell, root, lts, opts.BuiltinPluginPath)
		cr.Duration = now().Sub(started)
		if cr.Outcome != OutcomePass {
			result.OK = false
		}
		// 门禁：required 与 deprecated 均不得 skip/missing；skip 一律失败。
		if cr.Outcome == OutcomeSkip || cr.Outcome == OutcomeMissing {
			result.OK = false
		}
		result.Cells = append(result.Cells, cr)
	}
	return result, nil
}

func executeCell(
	ctx context.Context,
	cell Cell,
	repoRoot string,
	lts *apilts.Registry,
	builtinOverride string,
) CellResult {
	base := CellResult{ID: cell.ID, Status: cell.Status}
	protocol := strings.ToLower(strings.TrimSpace(cell.Protocol))
	manifest := strings.ToLower(strings.TrimSpace(cell.Manifest))

	switch {
	case cell.Command == "extension-test-builtin" || cell.ID == "manifest-v3-contract":
		return runManifestSDKContract(ctx, base, repoRoot, builtinOverride)
	case protocol == "v1" && cell.ExpectsShimTelemetry:
		return runProtocolV1ShimTelemetry(base, lts)
	case protocol == "v2" && (manifest == "v3" || manifest == "3"):
		return runCurrentHostProtocolV2ManifestV3(base, lts, repoRoot, builtinOverride)
	default:
		base.Outcome = OutcomeSkip
		base.Message = fmt.Sprintf("no executor for protocol=%s manifest=%s command=%s", cell.Protocol, cell.Manifest, cell.Command)
		return base
	}
}

// runCurrentHostProtocolV2ManifestV3 proves current Host contracts + Manifest V3 load.
func runCurrentHostProtocolV2ManifestV3(
	base CellResult,
	lts *apilts.Registry,
	repoRoot string,
	builtinOverride string,
) CellResult {
	// LTS 种子必须声明 current Host / protocol v2 / manifest v3。
	snap := lts.Snapshot()
	need := map[string]string{
		"sforum.host.v2":      "current",
		"sforum.protocol.v2":  "current",
		"sforum.manifest.v3":  "current",
	}
	found := map[string]string{}
	for _, c := range snap.Contracts {
		if want, ok := need[c.ID]; ok {
			found[c.ID] = c.Status
			if c.Status != want {
				base.Outcome = OutcomeFail
				base.Message = fmt.Sprintf("LTS contract %s status=%s want=%s", c.ID, c.Status, want)
				return base
			}
		}
	}
	for id := range need {
		if _, ok := found[id]; !ok {
			base.Outcome = OutcomeFail
			base.Message = "missing LTS contract " + id
			return base
		}
	}

	// Manifest V3 + SDK contract on a real builtin package.
	pluginRoot, err := resolveBuiltinPlugin(repoRoot, builtinOverride)
	if err != nil {
		base.Outcome = OutcomeFail
		base.Message = err.Error()
		return base
	}
	manifest, err := extensionmanifest.LoadPackage(pluginRoot)
	if err != nil {
		base.Outcome = OutcomeFail
		base.Message = "load manifest: " + err.Error()
		return base
	}
	if extensionmanifest.EffectiveManifestVersion(manifest) < 3 {
		base.Outcome = OutcomeFail
		base.Message = fmt.Sprintf("manifest version %d want >= 3", extensionmanifest.EffectiveManifestVersion(manifest))
		return base
	}
	if err := extensionmanifest.Validate(manifest); err != nil {
		base.Outcome = OutcomeFail
		base.Message = "validate manifest: " + err.Error()
		return base
	}
	report, err := pluginsdk.LoadAndTest(pluginRoot, pluginsdk.Options{SkipBackendBinary: true})
	if err != nil {
		base.Outcome = OutcomeFail
		base.Message = "sdk LoadAndTest: " + err.Error()
		return base
	}
	if !report.OK {
		base.Outcome = OutcomeFail
		base.Message = fmt.Sprintf("sdk contract failed errors=%d", report.Errors)
		return base
	}
	base.Outcome = OutcomePass
	base.Message = fmt.Sprintf("host.v2+protocol.v2+manifest.v3 ok package=%s", manifest.ID)
	return base
}

func runProtocolV1ShimTelemetry(base CellResult, lts *apilts.Registry) CellResult {
	// 验证 deprecated Protocol V1 合同存在、shim 启用，且 RecordShimCall 可观测。
	snap := lts.Snapshot()
	var contract *apilts.Contract
	for i := range snap.Contracts {
		if snap.Contracts[i].ID == apilts.ProtocolV1ContractID {
			contract = &snap.Contracts[i]
			break
		}
	}
	if contract == nil {
		base.Outcome = OutcomeFail
		base.Message = "protocol v1 LTS contract missing"
		return base
	}
	if contract.Status != "deprecated" || !contract.ShimEnabled {
		base.Outcome = OutcomeFail
		base.Message = fmt.Sprintf("protocol v1 status=%s shim=%v", contract.Status, contract.ShimEnabled)
		return base
	}
	before := lts.ShimCalls(apilts.ProtocolV1ContractID)
	lts.RecordShimCall(apilts.ProtocolV1ContractID)
	after := lts.ShimCalls(apilts.ProtocolV1ContractID)
	if after <= before {
		base.Outcome = OutcomeFail
		base.Message = "shim telemetry did not increment"
		return base
	}
	// 再次从 Snapshot 证明 telemetry 进入 ShimUsage（门禁删除前必须可观测）。
	snap = lts.Snapshot()
	var usage uint64
	for _, row := range snap.ShimUsage {
		if row.ContractID == apilts.ProtocolV1ContractID {
			usage = row.Calls
		}
	}
	if usage == 0 {
		base.Outcome = OutcomeFail
		base.Message = "shim usage missing from LTS snapshot"
		return base
	}
	base.Outcome = OutcomePass
	base.ShimCalls = usage
	base.Message = fmt.Sprintf("protocol v1 shim telemetry calls=%d", usage)
	return base
}

func runManifestSDKContract(
	ctx context.Context,
	base CellResult,
	repoRoot string,
	builtinOverride string,
) CellResult {
	_ = ctx
	pluginRoot, err := resolveBuiltinPlugin(repoRoot, builtinOverride)
	if err != nil {
		base.Outcome = OutcomeFail
		base.Message = err.Error()
		return base
	}
	report, err := pluginsdk.LoadAndTest(pluginRoot, pluginsdk.Options{SkipBackendBinary: true})
	if err != nil {
		base.Outcome = OutcomeFail
		base.Message = err.Error()
		return base
	}
	if !report.OK {
		base.Outcome = OutcomeFail
		base.Message = fmt.Sprintf("extension test failed errors=%d warnings=%d", report.Errors, report.Warnings)
		return base
	}
	if extensionmanifest.EffectiveManifestVersion(report.Manifest) < 3 {
		base.Outcome = OutcomeFail
		base.Message = "builtin package is not Manifest V3"
		return base
	}
	base.Outcome = OutcomePass
	base.Message = fmt.Sprintf("manifest/sdk contract ok id=%s", report.Manifest.ID)
	return base
}

func resolveBuiltinPlugin(repoRoot, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		info, err := os.Stat(override)
		if err != nil || !info.IsDir() {
			return "", fmt.Errorf("builtin plugin path invalid: %s", override)
		}
		return override, nil
	}
	// 优先站点搜索内置插件（Manifest V3 + 真实贡献）。
	candidates := []string{
		filepath.Join(repoRoot, "extensions", "builtin", "plugins", "sforum-search-site"),
		filepath.Join(repoRoot, "extensions", "builtin", "plugins", "sforum-smtp"),
		filepath.Join(repoRoot, "extensions", "builtin", "plugins", "sforum-storage-fs"),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("no builtin plugin found under %s/extensions/builtin/plugins", repoRoot)
}

func guessRepoRoot(matrixPath string) string {
	// tests/compat/matrix.yaml -> repo root
	dir := filepath.Dir(matrixPath)
	if filepath.Base(dir) == "compat" {
		parent := filepath.Dir(dir)
		if filepath.Base(parent) == "tests" {
			return filepath.Dir(parent)
		}
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
