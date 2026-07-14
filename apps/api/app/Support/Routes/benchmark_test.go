package routes

import "testing"

// BenchmarkRegistryResolveP0Catalog is the P6 in-memory counterpart to the
// P0 BenchmarkRouteGatewayV1Baseline loopback proxy benchmark.
func BenchmarkRegistryResolveP0Catalog(b *testing.B) {
	registry := NewRegistry()
	if _, err := registry.Publish(Publication{Core: p0CoreCatalog(b)}); err != nil {
		b.Fatal(err)
	}
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "static", method: "GET", path: "/api/v1/admin/pages/added"},
		{name: "parameter", method: "GET", path: "/api/v1/topics/42"},
		{name: "catch_all", method: "PURGE", path: "/api/v1/extensions/benchmark.plugin/cache"},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, err := registry.Resolve(test.method, test.path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
