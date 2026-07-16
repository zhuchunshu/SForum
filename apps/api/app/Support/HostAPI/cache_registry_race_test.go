package hostapi

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestHostCacheConcurrentOperationsAndInspector(t *testing.T) {
	fixture := newHostCacheTestFixture(t, "race-host.cache", "public", "")
	inspector := NewHostCacheInspector(256)
	service, err := NewHostCacheService(fixture.registry, fixture.backend, nil,
		hostCacheTestServiceOptions(WithHostCacheTraceSink(inspector))...)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 64
	start := make(chan struct{})
	errorsCh := make(chan error, workers)
	var group sync.WaitGroup
	for index := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			key := fmt.Sprintf("entry-%d", index)
			if _, setErr := service.Set(context.Background(), HostCacheSetRequest{
				HostCacheRequestBase: fixture.base, Key: key, Schema: fixture.schema,
				Value: []byte(key), TTL: time.Minute, Tags: []string{fixture.cache.Tags[0]},
			}); setErr != nil {
				errorsCh <- setErr
				return
			}
			result, getErr := service.Get(context.Background(), HostCacheGetRequest{
				HostCacheRequestBase: fixture.base, Key: key, Schema: fixture.schema,
			})
			if getErr != nil || !result.Found || string(result.Value) != key {
				errorsCh <- fmt.Errorf("get %s result=%#v err=%v", key, result, getErr)
				return
			}
			if _, incrementErr := service.Increment(context.Background(), HostCacheIncrementRequest{
				HostCacheRequestBase: fixture.base, Key: "shared", TTL: time.Minute,
			}); incrementErr != nil {
				errorsCh <- incrementErr
			}
		}()
	}
	close(start)
	group.Wait()
	close(errorsCh)
	for operationErr := range errorsCh {
		t.Error(operationErr)
	}
	value, err := service.Increment(context.Background(), HostCacheIncrementRequest{
		HostCacheRequestBase: fixture.base, Key: "shared", TTL: time.Minute,
	})
	if err != nil || value != workers+1 {
		t.Fatalf("shared counter=%d err=%v", value, err)
	}
	entries := inspector.Snapshot()
	if len(entries) == 0 || len(entries) > 256 {
		t.Fatalf("inspector size = %d", len(entries))
	}
	for index := 1; index < len(entries); index++ {
		if entries[index].Sequence <= entries[index-1].Sequence {
			t.Fatalf("non-monotonic inspector sequence at %d: %#v", index, entries)
		}
	}
}
