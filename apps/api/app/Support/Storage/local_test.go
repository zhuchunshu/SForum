package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalAdapterPutOpenStatAndDelete(t *testing.T) {
	adapter, err := NewLocalAdapter(t.TempDir(), "https://cdn.example.com/uploads")
	if err != nil {
		t.Fatalf("new local adapter: %v", err)
	}

	ctx := context.Background()
	if err := adapter.Put(ctx, "2026/07/file.txt", PutInput{
		Reader:      strings.NewReader("hello"),
		Size:        5,
		ContentType: "text/plain",
	}); err != nil {
		t.Fatalf("put object: %v", err)
	}

	ok, err := adapter.Exists(ctx, "2026/07/file.txt")
	if err != nil {
		t.Fatalf("exists object: %v", err)
	}
	if !ok {
		t.Fatal("expected object to exist")
	}

	info, err := adapter.Stat(ctx, "2026/07/file.txt")
	if err != nil {
		t.Fatalf("stat object: %v", err)
	}
	if info.Size != 5 {
		t.Fatalf("expected size 5, got %d", info.Size)
	}
	if got := adapter.PublicURL("2026/07/file.txt"); got != "https://cdn.example.com/uploads/2026/07/file.txt" {
		t.Fatalf("unexpected public url: %q", got)
	}

	reader, err := adapter.Open(ctx, "2026/07/file.txt")
	if err != nil {
		t.Fatalf("open object: %v", err)
	}
	body, err := io.ReadAll(reader)
	if closeErr := reader.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("expected hello, got %q", string(body))
	}

	if err := adapter.Delete(ctx, "2026/07/file.txt"); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	ok, err = adapter.Exists(ctx, "2026/07/file.txt")
	if err != nil {
		t.Fatalf("exists deleted object: %v", err)
	}
	if ok {
		t.Fatal("expected deleted object to be absent")
	}
}

func TestLocalAdapterRejectsUnsafeObjectKeys(t *testing.T) {
	adapter, err := NewLocalAdapter(t.TempDir(), "")
	if err != nil {
		t.Fatalf("new local adapter: %v", err)
	}

	cases := []string{
		"../outside.txt",
		"safe/../outside.txt",
		"safe/./outside.txt",
		"/absolute.txt",
	}
	for _, key := range cases {
		err := adapter.Put(context.Background(), key, PutInput{Reader: strings.NewReader("x")})
		if !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("expected invalid key for %q, got %v", key, err)
		}
	}
}

func TestNewLocalAdapterRequiresRoot(t *testing.T) {
	if _, err := NewLocalAdapter("", ""); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}
