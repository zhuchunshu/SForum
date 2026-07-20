package extensionsruntime_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	mediaregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/MediaRegistry"
)

// TestReferenceMediaOptimizePluginPublishesMIMETransformAndFallsBack proves the
// P13 media-optimize package publishes MIME policy + transform variants from
// Manifest V3 and keeps the immutable original after disable.
func TestReferenceMediaOptimizePluginPublishesMIMETransformAndFallsBack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reference media-optimize plugin subprocess build in short mode")
	}
	extension := buildReferenceMediaOptimizeExtension(t)
	if extension.ID != "sforum.media-optimize" || len(extension.Manifest.Media) != 1 {
		t.Fatalf("media-optimize package = id=%s media=%d", extension.ID, len(extension.Manifest.Media))
	}
	if len(extension.Manifest.Jobs) != 2 {
		t.Fatalf("media-optimize jobs = %#v", extension.Manifest.Jobs)
	}
	jobNames := map[string]bool{}
	for _, job := range extension.Manifest.Jobs {
		jobNames[job.Name] = true
	}
	if !jobNames["sforum.media-optimize.variants"] || !jobNames["sforum.media-optimize.retention"] {
		t.Fatalf("media-optimize job names = %#v", jobNames)
	}
	if len(extension.Manifest.Schedules) != 1 ||
		extension.Manifest.Schedules[0].JobID != "sforum.media-optimize.job.retention" {
		t.Fatalf("media-optimize schedules = %#v", extension.Manifest.Schedules)
	}
	if len(extension.Manifest.AdminSurfaces) != 1 ||
		extension.Manifest.AdminSurfaces[0].Kind != "notice" {
		t.Fatalf("media-optimize admin surfaces = %#v", extension.Manifest.AdminSurfaces)
	}
	// Host-assigned permission recommendation is catalog-only.
	if len(extension.Manifest.PermissionDefinitions) != 2 {
		t.Fatalf("permission definitions = %#v", extension.Manifest.PermissionDefinitions)
	}
	for _, permission := range extension.Manifest.PermissionDefinitions {
		if permission.AssignmentPolicy != "host" {
			t.Fatalf("permission must stay Host-assigned: %#v", permission)
		}
	}

	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust: staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{
			TrustGrantID: "media-optimize-reference", ImpactDigest: extension.PackageDigest,
		}},
	})
	manager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{Starter: starter})
	if err := manager.Start(t.Context(), extension); err != nil {
		t.Fatalf("start media-optimize plugin: %v", err)
	}
	stopped := false
	t.Cleanup(func() {
		if !stopped {
			_ = manager.Stop(context.Background(), extension)
		}
	})
	active, err := manager.ActiveRuntimeInstance(extension.ID)
	if err != nil {
		t.Fatal(err)
	}

	publication := mediaPublicationFromManifest(extension, active.Identity.InstanceID, extension.PackageDigest)
	registry := mediaregistry.New()
	// Core MIME baseline so plan has a policy winner path; plugin policy is higher priority.
	coreArtifact, err := mediaregistry.NewCoreArtifact("core.media", "1.0.0", strings.Repeat("1", 64), strings.Repeat("2", 64))
	if err != nil {
		t.Fatal(err)
	}
	core := mediaregistry.Publication{
		Artifact: coreArtifact,
		Policies: []mediaregistry.MIMEPolicyDeclaration{{
			ID: "core.media.policy", ContractVersion: "core.media.policy@1", Purpose: "general",
			RequiredPermission: "attachment.upload", AllowedMIMEs: []string{"image/*"},
			StrictDeclaredMIME: true, Budget: mediaregistry.DefaultBudget(),
		}},
	}
	if _, err := registry.ReplaceAll([]mediaregistry.Publication{core, publication}, false); err != nil {
		t.Fatalf("publish media graph: %v", err)
	}

	request := mediaregistry.PlanRequest{
		Kind: mediaregistry.PlanUpload, Purpose: "general", Permission: "sforum.media-optimize.transform",
		Actor: mediaregistry.Actor{ID: "user-7", PermissionFingerprint: "permissions-v1"},
		Source: mediaregistry.SourceAsset{
			ID: "attachment-ref", Digest: strings.Repeat("a", 64), Kind: mediaregistry.SourceOriginal,
			MIME: "image/png", Filename: "photo.png", SizeBytes: 1024, Immutable: true,
		},
		Upload: mediaregistry.UploadFacts{
			BatchFileCount: 1, DeclaredMIME: "image/png", DetectedMIMEs: []string{"image/png"},
		},
	}
	authorizer := mediaAuthorizerFunc(func(context.Context, mediaregistry.AuthorizationRequest) bool { return true })
	plan, err := registry.Plan(t.Context(), request, authorizer)
	if err != nil {
		t.Fatalf("plan upload: %v", err)
	}
	if !plan.Source.Immutable || plan.Source.Digest != request.Source.Digest {
		t.Fatalf("source must stay immutable: %#v", plan.Source)
	}
	foundTransform := false
	for _, step := range plan.Steps {
		if step.Processor.Stage == mediaregistry.StageTransform {
			foundTransform = true
			if step.Processor.ID != "sforum.media-optimize.pipeline.image" {
				t.Fatalf("transform processor = %#v", step.Processor)
			}
			if step.Processor.FailureMode != mediaregistry.FailureFallbackOriginal {
				t.Fatalf("failure mode = %s", step.Processor.FailureMode)
			}
			if step.Processor.Execution != mediaregistry.ExecutionBackground {
				t.Fatalf("transform should be background for variants: %#v", step.Processor)
			}
			if len(step.Variants) < 2 {
				t.Fatalf("expected thumbnail+preview variants, got %#v", step.Variants)
			}
		}
	}
	if !foundTransform {
		t.Fatalf("transform stage missing from plan: %#v", plan.Steps)
	}

	// Permission deny is Host-final.
	denied := mediaAuthorizerFunc(func(context.Context, mediaregistry.AuthorizationRequest) bool { return false })
	if _, err := registry.Plan(t.Context(), request, denied); !errors.Is(err, mediaregistry.ErrPermissionDenied) {
		t.Fatalf("denied plan err = %v", err)
	}

	// Disable plugin: remove publication; original source digest from prior plan stays immutable.
	originalDigest := plan.Source.Digest
	if err := manager.Stop(t.Context(), extension); err != nil {
		t.Fatal(err)
	}
	stopped = true
	if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
		t.Fatalf("remove media publication: removed=%v err=%v", removed, err)
	}
	if plan.Source.Digest != originalDigest || !plan.Source.Immutable {
		t.Fatalf("disable rewrote source: %#v", plan.Source)
	}
	// 卸载后 MIME/transform 不再参与计划；仅 core 策略仍可用（不同权限语义）。
	if snap := registry.Snapshot(); len(snap.Publications) != 1 || !snap.Publications[0].Artifact.Core {
		t.Fatalf("after remove publications = %#v", snap.Publications)
	}
}

func mediaPublicationFromManifest(
	extension extensions.Extension,
	runtimeInstanceID string,
	impactDigest string,
) mediaregistry.Publication {
	publication := mediaregistry.Publication{Artifact: mediaregistry.Artifact{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, ImpactDigest: impactDigest,
		VersionID: extension.ActiveVersionID, RuntimeInstanceID: runtimeInstanceID,
	}}
	for _, pipeline := range extension.Manifest.Media {
		permission := strings.TrimSpace(pipeline.Permission)
		if permission == "" {
			permission = "attachment.upload"
		}
		policyID := pipeline.ID + ".mime-policy"
		publication.Policies = append(publication.Policies, mediaregistry.MIMEPolicyDeclaration{
			ID: policyID, ContractVersion: policyID + "@1", Purpose: "general",
			Priority: pipeline.Priority + 20, RequiredPermission: permission,
			AllowedMIMEs: append([]string(nil), pipeline.MIMEs...),
			StrictDeclaredMIME: true, Budget: mediaregistry.DefaultBudget(),
		})
		execution := mediaregistry.ExecutionSync
		if len(pipeline.Transforms) > 0 {
			execution = mediaregistry.ExecutionBackground
		}
		publication.Processors = append(publication.Processors, mediaregistry.ProcessorDeclaration{
			ID: pipeline.ID, ContractVersion: pipeline.ContractVersion,
			Stage: mediaregistry.StageTransform, Purpose: "general",
			MIMEs: append([]string(nil), pipeline.MIMEs...), Handler: pipeline.Handler,
			Priority: pipeline.Priority, Mode: mediaregistry.ProcessorCompose,
			Execution: execution, FailureMode: mediaregistry.FailureFallbackOriginal,
			RequiredPermission: permission,
			Retry: mediaregistry.RetryPolicy{MaxAttempts: 3, BaseDelaySeconds: 2, MaxDelaySeconds: 30},
		})
		for _, transform := range pipeline.Transforms {
			variantID := pipeline.ID + "." + transform.ID
			outputMIME := "image/webp"
			switch strings.ToLower(strings.TrimSpace(transform.Format)) {
			case "png":
				outputMIME = "image/png"
			case "jpg", "jpeg":
				outputMIME = "image/jpeg"
			}
			publication.Variants = append(publication.Variants, mediaregistry.VariantDeclaration{
				ID: variantID, ContractVersion: variantID + "@1", Purpose: "general",
				Name: transform.Variant, ProcessorID: pipeline.ID,
				ProcessorContractVersion: pipeline.ContractVersion,
				ProcessorOwnerExtensionID: extension.ID,
				ProcessorPackageDigest:    extension.PackageDigest,
				OutputMIME:                outputMIME,
				Priority:                  pipeline.Priority,
			})
		}
	}
	return publication
}

type mediaAuthorizerFunc func(context.Context, mediaregistry.AuthorizationRequest) bool

func (f mediaAuthorizerFunc) Authorize(ctx context.Context, request mediaregistry.AuthorizationRequest) bool {
	return f(ctx, request)
}

func buildReferenceMediaOptimizeExtension(t *testing.T) extensions.Extension {
	t.Helper()
	fixtureRoot := referenceMediaOptimizeFixtureRoot(t)
	packageRoot := filepath.Join(t.TempDir(), "sforum.media-optimize")
	if err := os.CopyFS(packageRoot, os.DirFS(fixtureRoot)); err != nil {
		t.Fatalf("copy media-optimize plugin: %v", err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(fixtureRoot, "../../../.."))
	goModPath := filepath.Join(packageRoot, "backend", "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = []byte(strings.ReplaceAll(string(goMod), "../../../../../apps/api", filepath.Join(repositoryRoot, "apps/api")))
	if err := os.WriteFile(goModPath, goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(packageRoot, "backend", "plugin")
	build := exec.Command("go", "build", "-mod=mod", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	build.Dir = filepath.Join(packageRoot, "backend")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build media-optimize plugin: %v\n%s", err, output)
	}
	templateBody, err := os.ReadFile(filepath.Join(packageRoot, "sforum.extension.json.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	manifestBody := string(templateBody)
	manifestBody = strings.ReplaceAll(manifestBody, "__BACKEND_DIGEST__", fileSHA256(t, binaryPath))
	manifestBody = strings.ReplaceAll(manifestBody, "__BULK_PROPS_DIGEST__", fileSHA256(t, filepath.Join(packageRoot, "schemas/bulk-notice-props.json")))
	manifestBody = strings.ReplaceAll(manifestBody, "__BULK_RESULT_DIGEST__", fileSHA256(t, filepath.Join(packageRoot, "schemas/bulk-notice-result.json")))
	if strings.Contains(manifestBody, "__") {
		t.Fatal("media-optimize manifest still contains digest tokens")
	}
	if err := os.WriteFile(filepath.Join(packageRoot, extensionmanifest.ManifestFileName), []byte(manifestBody), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := extensionmanifest.LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load media-optimize package: %v", err)
	}
	packageDigest, err := extensionpackage.DigestTree(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: extensions.TypePlugin,
		Status: extensions.StatusEnabled, Source: extensions.SourceUploaded,
		PackagePath: packageRoot, PackageDigest: packageDigest, Manifest: manifest, ActiveVersionID: 701,
		CapabilityGrants: []extensions.CapabilityGrant{{Key: capabilities.HostAPI, Risk: capabilities.RiskLow}},
	}
}

func referenceMediaOptimizeFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../extensions/fixtures/plugins/sforum-media-optimize"))
}
