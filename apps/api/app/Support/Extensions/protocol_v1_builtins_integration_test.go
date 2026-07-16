package extensionsruntime_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

type protocolV1BuiltinSettings map[string]string

func (s protocolV1BuiltinSettings) ListSettings(context.Context, string) (map[string]string, error) {
	return s, nil
}

func TestProtocolV1BuiltInCompatibilityPackages(t *testing.T) {
	repositoryRoot := protocolV1RepositoryRoot(t)
	tests := []struct {
		name         string
		packageName  string
		manifestName string
		buildTags    string
		setup        func(*testing.T) (protocolV1BuiltinSettings, func(*testing.T, context.Context, *extensionsruntime.ProtocolStarter, extensions.Extension))
	}{
		{
			name: "smtp", packageName: "sforum-smtp", manifestName: extensions.ManifestFileName,
			setup: protocolV1SMTPFixture,
		},
		{
			name: "storage-fs", packageName: "sforum-storage-fs", manifestName: extensions.ManifestFileName,
			setup: protocolV1StorageFixture,
		},
		{
			name: "content-policy rollback", packageName: "sforum-content-policy",
			manifestName: "sforum.extension.v1.json", buildTags: "protocol_v1",
			setup: protocolV1ContentPolicyFixture,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			extension := buildProtocolV1Builtin(t, repositoryRoot, test.packageName, test.manifestName, test.buildTags)
			settings, exercise := test.setup(t)
			starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{Settings: settings})
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			target, err := starter.Start(ctx, extension)
			if err != nil {
				t.Fatalf("start built-in package: %v", err)
			}
			started := true
			t.Cleanup(func() {
				if started {
					_ = starter.Stop(context.Background(), extension)
				}
			})
			if target.BaseURL != "" {
				t.Fatalf("provider package exposed unexpected route target %q", target.BaseURL)
			}
			identity := extensionsruntime.RuntimeInstanceIdentity{
				ExtensionID: extension.ID, InstanceID: target.InstanceID,
			}

			exercise(t, ctx, starter, extension)
			telemetry := starter.ProtocolTelemetry(extension.ID)
			if telemetry.ProtocolVersion != 1 || telemetry.Transport != "net/rpc" || !telemetry.Deprecated || telemetry.StartCount != 1 || telemetry.CallCount == 0 {
				t.Fatalf("unexpected v1 telemetry: %#v", telemetry)
			}
			if err := starter.StopRetainedInstance(context.Background(), identity); !errors.Is(err, extensionsruntime.ErrProtocolInstancePublished) {
				t.Fatalf("active Protocol V1 retained-stop error = %v", err)
			}
			published, lease, err := starter.PublishInstanceSet(context.Background(), nil)
			if err != nil || len(published) != 0 || lease == nil {
				t.Fatalf("remove Protocol V1 from complete set = %#v lease=%v err=%v", published, lease, err)
			}
			lease.Release()
			if err := starter.StopRetainedInstance(context.Background(), identity); err != nil {
				t.Fatalf("stop retained built-in package: %v", err)
			}
			started = false
			if _, err := starter.ProviderProbe(ctx, extension.ID, extensionsruntime.ProviderProbeRequest{}); !errors.Is(err, extensions.ErrRuntimeUnavailable) {
				t.Fatalf("post-stop provider probe error = %v", err)
			}
		})
	}
}

func buildProtocolV1Builtin(t *testing.T, repositoryRoot, packageName, manifestName, buildTags string) extensions.Extension {
	t.Helper()
	sourceRoot := filepath.Join(repositoryRoot, "extensions", "builtin", "plugins", packageName)
	manifestBody, err := os.ReadFile(filepath.Join(sourceRoot, manifestName))
	if err != nil {
		t.Fatalf("read %s manifest: %v", packageName, err)
	}
	packageFiles := protocolV1ManifestFiles(t, sourceRoot, manifestBody)
	manifest, err := extensionmanifest.LoadPackageFS(packageFiles)
	if err != nil {
		t.Fatalf("load %s manifest: %v", packageName, err)
	}
	if manifest.Backend.ProtocolVersion != 1 || manifest.Backend.RPC != "hashicorp-go-plugin" {
		t.Fatalf("%s is not a protocol v1 package: %#v", packageName, manifest.Backend)
	}

	packageRoot := filepath.Join(t.TempDir(), packageName)
	binary := filepath.Join(packageRoot, filepath.FromSlash(manifest.Backend.Entry))
	for relative, body := range packageFiles {
		target := filepath.Join(packageRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create temporary package path %s: %v", relative, err)
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			t.Fatalf("write temporary package file %s: %v", relative, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatalf("create temporary backend path: %v", err)
	}
	arguments := []string{"build", "-trimpath", "-buildvcs=false", "-o", binary}
	if buildTags != "" {
		arguments = append(arguments, "-tags", buildTags)
	}
	arguments = append(arguments, ".")
	command := exec.Command("go", arguments...)
	command.Dir = filepath.Join(sourceRoot, "backend")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build %s protocol v1 binary: %v\n%s", packageName, err, output)
	}
	if info, err := os.Stat(binary); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		t.Fatalf("%s binary is not executable: %v", packageName, err)
	}
	return extensions.Extension{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: manifest.Type,
		Status: extensions.StatusEnabled, Source: extensions.SourceBuiltin,
		Manifest: manifest, PackagePath: packageRoot,
	}
}

func protocolV1ManifestFiles(t *testing.T, sourceRoot string, rootBody []byte) extensionmanifest.FileMapFS {
	t.Helper()
	files := extensionmanifest.FileMapFS{extensions.ManifestFileName: rootBody}
	manifestRoot := filepath.Join(sourceRoot, "manifest")
	if err := filepath.Walk(manifestRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = body
		return nil
	}); err != nil {
		t.Fatalf("load manifest partials: %v", err)
	}
	return files
}

func protocolV1SMTPFixture(t *testing.T) (protocolV1BuiltinSettings, func(*testing.T, context.Context, *extensionsruntime.ProtocolStarter, extensions.Extension)) {
	t.Helper()
	host, port, wait := startSMTPProbeServer(t)
	settings := protocolV1BuiltinSettings{
		"host": host, "port": strconv.Itoa(port), "encryption": "none",
		"from_address": "noreply@example.com", "from_name": "SForum",
	}
	return settings, func(t *testing.T, ctx context.Context, starter *extensionsruntime.ProtocolStarter, extension extensions.Extension) {
		t.Helper()
		probe, err := starter.ProviderProbe(ctx, extension.ID, extensionsruntime.ProviderProbeRequest{Slot: "mail.provider"})
		if err != nil || !probe.OK || probe.Reason != "smtp.connection_ok" {
			t.Fatalf("SMTP provider probe = %#v err=%v", probe, err)
		}
		wait(t)
	}
}

func protocolV1StorageFixture(t *testing.T) (protocolV1BuiltinSettings, func(*testing.T, context.Context, *extensionsruntime.ProtocolStarter, extensions.Extension)) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "objects")
	settings := protocolV1BuiltinSettings{"root_path": root, "public_base_url": "https://cdn.example.test/files"}
	return settings, func(t *testing.T, ctx context.Context, starter *extensionsruntime.ProtocolStarter, extension extensions.Extension) {
		t.Helper()
		probe, err := starter.StorageProbe(ctx, extension.ID, extensionsruntime.StorageProbeRequest{})
		if err != nil || !probe.OK || probe.Reason != "storage.ok" {
			t.Fatalf("storage provider probe = %#v err=%v", probe, err)
		}
		payload := []byte("protocol-v1-storage-round-trip")
		begin, err := starter.StoragePutBegin(ctx, extension.ID, extensionsruntime.StoragePutBeginRequest{
			Key: "compatibility/p3.txt", ContentType: "text/plain", Size: int64(len(payload)),
		})
		if err != nil || !begin.OK {
			t.Fatalf("storage put begin = %#v err=%v", begin, err)
		}
		written, err := starter.StoragePutChunk(ctx, extension.ID, extensionsruntime.StoragePutChunkRequest{SessionID: begin.SessionID, Data: payload, Final: true})
		if err != nil || !written.OK {
			t.Fatalf("storage put = %#v err=%v", written, err)
		}
		opened, err := starter.StorageOpen(ctx, extension.ID, extensionsruntime.StorageOpenRequest{Key: "compatibility/p3.txt"})
		if err != nil || !opened.OK || opened.Size != int64(len(payload)) {
			t.Fatalf("storage open = %#v err=%v", opened, err)
		}
		var result bytes.Buffer
		for {
			chunk, err := starter.StorageGetChunk(ctx, extension.ID, extensionsruntime.StorageGetChunkRequest{SessionID: opened.SessionID, MaxBytes: 7})
			if err != nil || !chunk.OK {
				t.Fatalf("storage get chunk = %#v err=%v", chunk, err)
			}
			_, _ = result.Write(chunk.Data)
			if chunk.EOF {
				break
			}
		}
		if !bytes.Equal(result.Bytes(), payload) {
			t.Fatalf("storage payload = %q, want %q", result.Bytes(), payload)
		}
		if closed, err := starter.StorageClose(ctx, extension.ID, extensionsruntime.StorageCloseRequest{SessionID: opened.SessionID}); err != nil || !closed.OK {
			t.Fatalf("storage close = %#v err=%v", closed, err)
		}
		if deleted, err := starter.StorageDelete(ctx, extension.ID, extensionsruntime.StorageObjectRequest{Key: "compatibility/p3.txt"}); err != nil || !deleted.OK {
			t.Fatalf("storage delete = %#v err=%v", deleted, err)
		}
	}
}

func protocolV1ContentPolicyFixture(*testing.T) (protocolV1BuiltinSettings, func(*testing.T, context.Context, *extensionsruntime.ProtocolStarter, extensions.Extension)) {
	settings := protocolV1BuiltinSettings{
		"enabled": "true", "keywords": "blocked", "mode": "reject",
		"match_title": "true", "match_content": "true",
	}
	return settings, func(t *testing.T, ctx context.Context, starter *extensionsruntime.ProtocolStarter, extension extensions.Extension) {
		t.Helper()
		result := starter.InvokeHook(ctx, extension, extensionsruntime.HookInput{
			Name: "comment.before_create", Kind: "filter", Timeout: 2 * time.Second,
			Payload: map[string]any{"content": "a blocked reply"},
		})
		if result.OK || result.Reason != "content_policy.keyword_blocked" {
			t.Fatalf("content-policy v1 decision = %#v", result)
		}
	}
}

func startSMTPProbeServer(t *testing.T) (string, int, func(*testing.T)) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for SMTP probe: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		if _, err := io.WriteString(connection, "220 localhost ESMTP ready\r\n"); err != nil {
			done <- err
			return
		}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- err
				return
			}
			command := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(command, "EHLO "), strings.HasPrefix(command, "HELO "):
				_, err = io.WriteString(connection, "250-localhost\r\n250 OK\r\n")
			case command == "QUIT":
				_, err = io.WriteString(connection, "221 bye\r\n")
				if err == nil {
					done <- nil
				} else {
					done <- err
				}
				return
			default:
				err = fmt.Errorf("unexpected SMTP command %q", command)
			}
			if err != nil {
				done <- err
				return
			}
		}
	}()
	address := listener.Addr().(*net.TCPAddr)
	wait := func(t *testing.T) {
		t.Helper()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("SMTP probe server: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("SMTP probe server did not finish")
		}
	}
	return address.IP.String(), address.Port, wait
}

func protocolV1RepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve protocol v1 fixture path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
}
