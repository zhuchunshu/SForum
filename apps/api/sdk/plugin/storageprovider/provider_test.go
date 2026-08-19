package storageprovider

import (
	"bytes"
	"io"
	"math"
	"testing"
	"time"
)

type memoryStorageBackend struct {
	objects map[string][]byte
}

func (b *memoryStorageBackend) Probe() error { return nil }

func (b *memoryStorageBackend) Put(key, _ string, reader io.Reader) error {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	b.objects[key] = payload
	return nil
}

func (b *memoryStorageBackend) Open(key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.objects[key])), nil
}

func (b *memoryStorageBackend) Delete(key string) error {
	delete(b.objects, key)
	return nil
}

func (b *memoryStorageBackend) Stat(key string) (StorageObjectInfo, error) {
	payload, ok := b.objects[key]
	return StorageObjectInfo{Exists: ok, Size: int64(len(payload))}, nil
}

func (b *memoryStorageBackend) Exists(key string) (bool, error) {
	_, ok := b.objects[key]
	return ok, nil
}

func (b *memoryStorageBackend) PublicURL(key string) string { return "https://files.example/" + key }

func (b *memoryStorageBackend) SignedURL(key string, _ time.Duration) (string, error) {
	return b.PublicURL(key), nil
}

func TestStorageProviderRoundTrip(t *testing.T) {
	backend := &memoryStorageBackend{objects: map[string][]byte{}}
	provider := NewStorageProvider("storage.test", func() (StorageBackend, error) { return backend, nil })
	payload := []byte("chunked-plugin-storage")

	begin, err := provider.StoragePutBegin(StoragePutBeginRequest{Key: "2026/demo.txt", Size: int64(len(payload))})
	if err != nil || !begin.OK {
		t.Fatalf("put begin = %#v err=%v", begin, err)
	}
	for offset := 0; offset < len(payload); offset += 5 {
		end := offset + 5
		if end > len(payload) {
			end = len(payload)
		}
		result, err := provider.StoragePutChunk(StoragePutChunkRequest{SessionID: begin.SessionID, Data: payload[offset:end], Final: end == len(payload)})
		if err != nil || !result.OK {
			t.Fatalf("put chunk = %#v err=%v", result, err)
		}
	}
	if !bytes.Equal(backend.objects["2026/demo.txt"], payload) {
		t.Fatalf("stored payload = %q", backend.objects["2026/demo.txt"])
	}

	opened, err := provider.StorageOpen(StorageOpenRequest{Key: "2026/demo.txt"})
	if err != nil || !opened.OK || opened.Size != int64(len(payload)) {
		t.Fatalf("open = %#v err=%v", opened, err)
	}
	var received []byte
	for {
		chunk, err := provider.StorageGetChunk(StorageGetChunkRequest{SessionID: opened.SessionID, MaxBytes: 4})
		if err != nil || !chunk.OK {
			t.Fatalf("get chunk = %#v err=%v", chunk, err)
		}
		received = append(received, chunk.Data...)
		if chunk.EOF {
			break
		}
	}
	if !bytes.Equal(received, payload) {
		t.Fatalf("read payload = %q", received)
	}
}

func TestStorageProviderRejectsOversizedChunk(t *testing.T) {
	provider := NewStorageProvider("storage.test", func() (StorageBackend, error) {
		return &memoryStorageBackend{objects: map[string][]byte{}}, nil
	})
	begin, err := provider.StoragePutBegin(StoragePutBeginRequest{Key: "a.txt", Size: 1})
	if err != nil || !begin.OK {
		t.Fatalf("put begin = %#v err=%v", begin, err)
	}
	result, err := provider.StoragePutChunk(StoragePutChunkRequest{SessionID: begin.SessionID, Data: []byte("too large"), Final: true})
	if err != nil || result.OK || result.Reason != "storage.test.size_mismatch" {
		t.Fatalf("result = %#v err=%v", result, err)
	}
}

func TestMultiStorageProviderKeepsInstancesIsolated(t *testing.T) {
	provider := NewMultiStorageProvider("storage.test", func(settings map[string]string) (StorageBackend, error) {
		return &memoryStorageBackend{objects: map[string][]byte{"marker": []byte(settings["marker"])}}, nil
	})
	for id, marker := range map[string]string{"primary": "aws", "archive": "r2"} {
		result, err := provider.StorageConfigureInstance(StorageConfigureInstanceRequest{
			InstanceID: id,
			Settings:   map[string]string{"marker": marker},
		})
		if err != nil || !result.OK {
			t.Fatalf("configure %s = %#v err=%v", id, result, err)
		}
	}

	for id, want := range map[string]string{"primary": "aws", "archive": "r2"} {
		opened, err := provider.StorageOpen(StorageOpenRequest{InstanceID: id, Key: "marker"})
		if err != nil || !opened.OK {
			t.Fatalf("open %s = %#v err=%v", id, opened, err)
		}
		chunk, err := provider.StorageGetChunk(StorageGetChunkRequest{SessionID: opened.SessionID, MaxBytes: 16})
		if err != nil || !chunk.OK || string(chunk.Data) != want {
			t.Fatalf("read %s = %#v err=%v", id, chunk, err)
		}
	}

	removed, err := provider.StorageRemoveInstance(StorageRemoveInstanceRequest{InstanceID: "primary"})
	if err != nil || !removed.OK {
		t.Fatalf("remove = %#v err=%v", removed, err)
	}
	missing, err := provider.StorageOpen(StorageOpenRequest{InstanceID: "primary", Key: "marker"})
	if err != nil || missing.OK || missing.Reason != "storage.test.config" {
		t.Fatalf("removed instance open = %#v err=%v", missing, err)
	}
}

func TestNormalizeStorageChunkSizeBoundsBeforeIntegerConversion(t *testing.T) {
	tests := []struct {
		name      string
		requested int64
		want      int
	}{
		{name: "negative", requested: -1, want: storageProviderChunkLimit},
		{name: "zero", requested: 0, want: storageProviderChunkLimit},
		{name: "requested", requested: 4096, want: 4096},
		{name: "over limit", requested: storageProviderChunkLimit + 1, want: storageProviderChunkLimit},
		{name: "maximum int64", requested: math.MaxInt64, want: storageProviderChunkLimit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeStorageChunkSize(test.requested); got != test.want {
				t.Fatalf("normalizeStorageChunkSize(%d) = %d, want %d", test.requested, got, test.want)
			}
		})
	}
}
