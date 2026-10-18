package extensionopenapi

import (
	"strings"
	"testing"
)

// 保存设置等生命周期操作会对同一批不可变制品连续编译多个候选聚合。loader 的
// 存在意义是让「同一个制品」在一次同步调用里只被加载并校验一次；这些用例锁定
// 复用行为，同时锁定校验本身没有被跳过。
func TestArtifactLoaderReusesVerifiedArtifact(t *testing.T) {
	artifact := buildFixture(t, defaultFixtureOptions("acme.loader"))

	loader := NewArtifactLoader()
	first, err := loadArtifact(artifact, &resourceBudget{}, loader)
	if err != nil {
		t.Fatalf("cold load: %v", err)
	}
	second, err := loadArtifact(artifact, &resourceBudget{}, loader)
	if err != nil {
		t.Fatalf("cached load: %v", err)
	}
	if first != second {
		t.Fatal("loader did not reuse the artifact verified for the same identity")
	}
	if second.input.ExtensionID != artifact.ExtensionID || second.input.Version != artifact.Version ||
		second.input.PackageDigest != artifact.PackageDigest || second.input.Root != artifact.Root {
		t.Fatalf("cached artifact lost its identity: %#v", second.input)
	}
}

// 冷启动（缓存未命中）仍必须完整校验包摘要：篡改后的期望摘要不能被缓存掩盖。
func TestArtifactLoaderStillVerifiesDigestOnMiss(t *testing.T) {
	artifact := buildFixture(t, defaultFixtureOptions("acme.loader.miss"))
	loader := NewArtifactLoader()

	if _, err := loadArtifact(artifact, &resourceBudget{}, loader); err != nil {
		t.Fatalf("cold load: %v", err)
	}

	tampered := artifact
	tampered.PackageDigest = strings.Repeat("0", 64)
	if _, err := loadArtifact(tampered, &resourceBudget{}, loader); err == nil {
		t.Fatal("loader served a result for an unverified package digest")
	}
}

// 身份字段必须全部参与缓存键：版本变化时若命中旧条目，聚合会把错误的
// extensionVersion 写进 fragment source identity。
func TestArtifactLoaderSeparatesIdentity(t *testing.T) {
	loader := NewArtifactLoader()
	base := Artifact{
		Root: "/packages/acme", ExtensionID: "acme.loader.identity", Version: "1.0.0",
		PackageDigest: strings.Repeat("a", 64),
	}
	variants := map[string]Artifact{
		"baseline": base,
		"version":  func() Artifact { v := base; v.Version = "1.0.1"; return v }(),
		"digest":   func() Artifact { v := base; v.PackageDigest = strings.Repeat("b", 64); return v }(),
		"root":     func() Artifact { v := base; v.Root = "/packages/other"; return v }(),
		"id":       func() Artifact { v := base; v.ExtensionID = "acme.other"; return v }(),
	}
	seen := make(map[string]string, len(variants))
	for name, variant := range variants {
		key := loader.key(variant)
		if previous, duplicate := seen[key]; duplicate {
			t.Fatalf("%s collides with %s: cache key %q ignores an identity field", name, previous, key)
		}
		seen[key] = name
	}
}

// nil loader 保持旧的「每次构建各自校验一遍」语义。
func TestArtifactLoaderNilKeepsPerBuildVerification(t *testing.T) {
	artifact := buildFixture(t, defaultFixtureOptions("acme.loader.nil"))

	first, err := loadArtifact(artifact, &resourceBudget{}, nil)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	second, err := loadArtifact(artifact, &resourceBudget{}, nil)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if first == second {
		t.Fatal("nil loader must keep verifying the package on every build")
	}
}

// BuildInput.Loader 必须把同一批制品透传到 catalog 与 aggregate 两次编译。
func TestBuildInputLoaderIsSharedAcrossBuilds(t *testing.T) {
	artifact := buildFixture(t, defaultFixtureOptions("acme.loader.build"))
	loader := NewArtifactLoader()

	if _, err := buildSchemaAggregate(BuildInput{Artifacts: []Artifact{artifact}, Loader: loader}); err != nil {
		t.Fatalf("catalog build: %v", err)
	}
	loaded, ok := loader.get(artifact)
	if !ok || loaded == nil {
		t.Fatal("catalog build did not populate the shared loader")
	}
	if _, err := buildAggregate(BuildInput{Artifacts: []Artifact{artifact}, Loader: loader}, true); err != nil {
		t.Fatalf("aggregate build: %v", err)
	}
	reused, ok := loader.get(artifact)
	if !ok || reused != loaded {
		t.Fatal("aggregate build did not reuse the artifact loaded by the catalog build")
	}
}
