//go:build unix

package extensions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestPublicFrontendFIFOAssetFailsWithoutBlocking(t *testing.T) {
	extension := publicFrontendFixture(t)
	reader := &fakeFrontendExtensionReader{item: extension}
	trust := NewExecutableTrustService(reader, &memoryExecutableTrustStore{})
	service := newAdmittedPublicFrontendService(reader, trust)
	grantPublicFrontend(t, trust, extension)
	publishTrustedPublicAssets(t, service, extension)
	descriptor, err := service.PublicComponent(t.Context(), extension.ID, extension.Manifest.Components[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(extension.PackagePath, extension.Manifest.PackageFiles[0].Path)
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(target, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, callErr := service.PublicAsset(
			context.Background(), extension.ID, extension.PackageDigest, descriptor.Entry.Digest, descriptor.Entry.Handle,
		)
		done <- callErr
	}()

	select {
	case callErr := <-done:
		if !errors.Is(callErr, ErrPublicFrontendUnavailable) {
			t.Fatalf("FIFO asset error=%v", callErr)
		}
	case <-time.After(500 * time.Millisecond):
		// 回归时先唤醒阻塞 reader，避免失败用例遗留挂起 goroutine。
		writer, openErr := os.OpenFile(target, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if openErr == nil {
			_ = writer.Close()
		}
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
		}
		t.Fatal("FIFO asset read blocked")
	}
}
