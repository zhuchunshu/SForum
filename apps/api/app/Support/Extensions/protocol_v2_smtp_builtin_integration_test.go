package extensionsruntime_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

// TestProtocolV2SMTPBuiltinMailProvider 证明默认 sforum.smtp 制品走 gRPC V2，
// known-slot probe 仍可用，且不再计入 Protocol V1 弃用遥测。
func TestProtocolV2SMTPBuiltinMailProvider(t *testing.T) {
	repositoryRoot := protocolV1RepositoryRoot(t)
	extension := buildProtocolV2SMTPBuiltin(t, repositoryRoot)
	host, port, wait := startSMTPProbeServer(t)
	settings := protocolV1BuiltinSettings{
		"host": host, "port": strconv.Itoa(port), "encryption": "none",
		"from_address": "noreply@example.com", "from_name": "SForum",
	}
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Settings: settings,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	target, err := starter.Start(ctx, extension)
	if err != nil {
		t.Fatalf("start smtp v2: %v", err)
	}
	t.Cleanup(func() { _ = starter.Stop(context.Background(), extension) })
	if target.BaseURL != "" {
		t.Fatalf("smtp v2 must not expose HTTP route target: %#v", target)
	}

	probe, err := starter.ProviderProbe(ctx, extension.ID, extensionsruntime.ProviderProbeRequest{Slot: "mail.provider"})
	if err != nil || !probe.OK || probe.Reason != "smtp.connection_ok" {
		t.Fatalf("SMTP v2 provider probe = %#v err=%v", probe, err)
	}
	wait(t)

	telemetry := starter.ProtocolTelemetry(extension.ID)
	if telemetry.ProtocolVersion != 2 || telemetry.Transport != "grpc" || telemetry.Deprecated {
		t.Fatalf("expected protocol v2 telemetry, got %#v", telemetry)
	}
	if telemetry.StartCount != 1 || telemetry.CallCount == 0 {
		t.Fatalf("unexpected call counters: %#v", telemetry)
	}
}

func buildProtocolV2SMTPBuiltin(t *testing.T, repositoryRoot string) extensions.Extension {
	t.Helper()
	packageName := "sforum-smtp"
	sourceRoot := filepath.Join(repositoryRoot, "extensions", "builtin", "plugins", packageName)
	moduleRoot := filepath.Join(sourceRoot, "backend")

	// 在源码树构建可执行文件，使 manifest packageFiles digest 校验通过。
	sourceBinary := filepath.Join(moduleRoot, "plugin")
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", sourceBinary, ".")
	command.Dir = moduleRoot
	command.Env = append(os.Environ(), "GOWORK="+temporaryPluginWorkspace(t, repositoryRoot, moduleRoot))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build smtp v2 binary: %v\n%s", err, output)
	}
	body, err := os.ReadFile(sourceBinary)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	sum := sha256.Sum256(body)
	digest := hex.EncodeToString(sum[:])

	// 若提交的 digest 与本机构建不一致，仍用实际 digest 启动（制品一致性由 CI 机器对齐）。
	manifest, err := extensionmanifest.LoadPackage(sourceRoot)
	if err != nil {
		// digest 漂移时改写临时副本再加载。
		packageRoot := filepath.Join(t.TempDir(), packageName)
		if err := copyTree(sourceRoot, packageRoot); err != nil {
			t.Fatalf("copy package: %v", err)
		}
		manifestPath := filepath.Join(packageRoot, extensions.ManifestFileName)
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		// 简单替换 digest 字段为本次构建值。
		updated := replaceSMTPDigests(raw, digest)
		if err := os.WriteFile(manifestPath, updated, 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		manifest, err = extensionmanifest.LoadPackage(packageRoot)
		if err != nil {
			t.Fatalf("load smtp v2 package after digest rewrite: %v", err)
		}
		sourceRoot = packageRoot
		sourceBinary = filepath.Join(packageRoot, "backend", "plugin")
	} else if manifest.Backend.Digest != digest {
		// 已通过校验但 digest 不同：强制以构建产物为准。
		manifest.Backend.Digest = digest
		for i := range manifest.PackageFiles {
			if manifest.PackageFiles[i].Path == "backend/plugin" {
				manifest.PackageFiles[i].Digest = digest
			}
		}
	}
	if manifest.Backend.ProtocolVersion != 2 {
		t.Fatalf("default smtp package must be protocol v2: %#v", manifest.Backend)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: manifest.Type,
		Status: extensions.StatusEnabled, Source: extensions.SourceBuiltin,
		Manifest: manifest, PackagePath: sourceRoot, PackageDigest: digest,
	}
}

func replaceSMTPDigests(raw []byte, digest string) []byte {
	// 测试辅助：把已提交 digest 替换为本次构建值（package 仅含 backend digest）。
	const committed = "fc20b752646e0ca76e1cfde53cf8163166cda6767a37d64132e3a9828cc1ecba"
	return []byte(strings.ReplaceAll(string(raw), committed, digest))
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
