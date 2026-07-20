package dependencyinspector

import "testing"

func TestSnapshotAndCycleDetection(t *testing.T) {
	ins := New()
	ins.UpsertNode(Node{ExtensionID: "a", Version: "1.0.0", Enabled: true})
	ins.UpsertNode(Node{ExtensionID: "b", Version: "1.0.0", Enabled: true})
	ins.UpsertNode(Node{ExtensionID: "c", Version: "1.0.0", Enabled: false})
	ins.SetEdges([]Edge{
		{From: "a", To: "b", Kind: "required", Constraint: ">=1.0.0"},
		{From: "b", To: "a", Kind: "required"},
		{From: "a", To: "c", Kind: "optional"},
	})
	snap := ins.Snapshot()
	if len(snap.Nodes) != 3 || len(snap.Edges) != 3 {
		t.Fatalf("snap = %#v", snap)
	}
	if len(snap.Cycles) == 0 {
		t.Fatal("expected required cycle a↔b")
	}
}
