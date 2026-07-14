package extensionmanifest

import "testing"

func TestManifestV3NormalizesHookExecutionAndFailureDefaults(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Hooks[0].Execution = ""
	manifest.Hooks[0].FailurePolicy = ""
	manifest.Hooks[0].TimeoutMS = 0
	normalized := Normalize(manifest)
	hook := normalized.Hooks[0]
	if hook.Execution != "sync" || hook.FailurePolicy != "fail_closed" || hook.TimeoutMS != 2000 {
		t.Fatalf("sync defaults = %#v", hook)
	}
	if err := Validate(normalized); err != nil {
		t.Fatalf("normalized hook contract: %v", err)
	}

	manifest.Hooks[0].Kind = "observe"
	manifest.Hooks[0].ResultSchema = ""
	manifest.Hooks[0].Execution = ""
	manifest.Hooks[0].FailurePolicy = ""
	manifest.Hooks[0].TimeoutMS = 0
	normalized = Normalize(manifest)
	hook = normalized.Hooks[0]
	if hook.Execution != "async" || hook.FailurePolicy != "fail_open" || hook.TimeoutMS != 5000 {
		t.Fatalf("async defaults = %#v", hook)
	}
}

func TestManifestV3AllowsDependencyTargetedHookAndRejectsUnsafeModes(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Hooks = []ManifestHook{{
		ID: "demo.v3.hook.consumer", ContractVersion: "provider.hook.transform@1",
		Name: "provider.content.transform", TargetID: "provider.hook.transform",
		Kind: "filter", Handler: "hook.consume", InputSchema: "provider.content@1",
		ResultSchema: "provider.content.result@1", Execution: "sync",
		FailurePolicy: "fail_closed", TimeoutMS: 1000, MutableFields: []string{"title"},
	}}
	manifest.Dependencies = append(manifest.Dependencies, ManifestDependency{ID: "provider", Version: "^1.0.0", Kind: "required"})
	if err := Validate(manifest); err != nil {
		t.Fatalf("dependency-targeted hook should validate: %v", err)
	}

	manifest.Hooks[0].Execution = "async"
	if err := Validate(manifest); err == nil {
		t.Fatal("async filter must be rejected because its patch cannot remain on the authoritative write path")
	}
	manifest.Hooks[0].Execution = "sync"
	manifest.Hooks[0].ResultSchema = ""
	if err := Validate(manifest); err == nil {
		t.Fatal("action/filter hook without typed result identity must be rejected")
	}
	manifest.Hooks[0].Kind = "action"
	manifest.Hooks[0].ResultSchema = "provider.content.result@1"
	manifest.Hooks[0].Execution = "async"
	manifest.Hooks[0].FailurePolicy = "fail_closed"
	if err := Validate(manifest); err == nil {
		t.Fatal("async fail_closed must be rejected because queued deliveries cannot roll back atomically")
	}
}

func TestManifestV3AllowsPassiveHookDefinitionButListenerNeedsHandler(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Hooks[0].Handler = ""
	if err := Validate(manifest); err != nil {
		t.Fatalf("passive hook definition should validate: %v", err)
	}
	manifest.Hooks[0].TargetID = "provider.hook"
	manifest.Hooks[0].Name = "provider.changed"
	if err := Validate(manifest); err == nil {
		t.Fatal("targeted listener without handler must be rejected")
	}
}

func TestManifestV3RejectsHookTimeoutAboveHostDeadline(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Hooks[0].TimeoutMS = HookMaximumTimeoutMS + 1
	if err := Validate(manifest); err == nil {
		t.Fatal("hook timeout above Protocol V2 Host deadline must be rejected")
	}
}
