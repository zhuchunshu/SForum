package dependencyinspector

import "testing"

func BenchmarkDependencyInspectorSnapshot(b *testing.B) {
	ins := New()
	for i := 0; i < 64; i++ {
		id := "ext." + string(rune('a'+i%26)) + string(rune('0'+i/26))
		ins.UpsertNode(Node{ExtensionID: id, Version: "1.0.0", Enabled: true})
	}
	edges := make([]Edge, 0, 64)
	for i := 0; i < 63; i++ {
		from := "ext." + string(rune('a'+i%26)) + string(rune('0'+i/26))
		to := "ext." + string(rune('a'+(i+1)%26)) + string(rune('0'+(i+1)/26))
		edges = append(edges, Edge{From: from, To: to, Kind: "required"})
	}
	ins.SetEdges(edges)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ins.Snapshot()
	}
}
