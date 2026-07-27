// Package dependencyinspector exposes Host dependency graph inspection for V3 P12.
package dependencyinspector

import (
	"sort"
	"strings"
	"sync"
)

// SchemaVersion is the dependency inspector snapshot contract.
const SchemaVersion = "sforum.dependency-inspector@1"

// Edge is one required/optional/conflict dependency edge.
type Edge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"` // required|optional|conflict|provides
	Constraint string `json:"constraint,omitempty"`
}

// Node is one extension package node.
type Node struct {
	ExtensionID   string `json:"extensionId"`
	Version       string `json:"version,omitempty"`
	PackageDigest string `json:"packageDigest,omitempty"`
	Enabled       bool   `json:"enabled,omitempty"`
}

// Snapshot is the operator-facing dependency view.
type Snapshot struct {
	SchemaVersion string `json:"schemaVersion"`
	Nodes         []Node `json:"nodes"`
	Edges         []Edge `json:"edges"`
	// Cycles lists simple cycle paths if any.
	Cycles [][]string `json:"cycles,omitempty"`
}

// Inspector is a process-local dependency graph.
type Inspector struct {
	mu    sync.Mutex
	nodes map[string]Node
	edges []Edge
}

// New builds an empty inspector.
func New() *Inspector {
	return &Inspector{nodes: make(map[string]Node)}
}

// UpsertNode adds or replaces a package node.
func (i *Inspector) UpsertNode(node Node) {
	if i == nil {
		return
	}
	node.ExtensionID = strings.ToLower(strings.TrimSpace(node.ExtensionID))
	if node.ExtensionID == "" {
		return
	}
	i.mu.Lock()
	i.nodes[node.ExtensionID] = node
	i.mu.Unlock()
}

// SetEdges replaces all edges (Host rebuilds from Manifest graph).
func (i *Inspector) SetEdges(edges []Edge) {
	if i == nil {
		return
	}
	cloned := make([]Edge, 0, len(edges))
	for _, edge := range edges {
		edge.From = strings.ToLower(strings.TrimSpace(edge.From))
		edge.To = strings.ToLower(strings.TrimSpace(edge.To))
		edge.Kind = strings.ToLower(strings.TrimSpace(edge.Kind))
		if edge.From == "" || edge.To == "" || edge.Kind == "" {
			continue
		}
		cloned = append(cloned, edge)
	}
	i.mu.Lock()
	i.edges = cloned
	i.mu.Unlock()
}

// Snapshot returns nodes, edges, and detected required-dependency cycles.
func (i *Inspector) Snapshot() Snapshot {
	if i == nil {
		return Snapshot{SchemaVersion: SchemaVersion}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	nodes := make([]Node, 0, len(i.nodes))
	for _, node := range i.nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(a, b int) bool { return nodes[a].ExtensionID < nodes[b].ExtensionID })
	edges := append([]Edge(nil), i.edges...)
	sort.Slice(edges, func(a, b int) bool {
		if edges[a].From != edges[b].From {
			return edges[a].From < edges[b].From
		}
		return edges[a].To < edges[b].To
	})
	return Snapshot{
		SchemaVersion: SchemaVersion,
		Nodes:         nodes,
		Edges:         edges,
		Cycles:        detectRequiredCycles(edges),
	}
}

func detectRequiredCycles(edges []Edge) [][]string {
	adj := map[string][]string{}
	for _, edge := range edges {
		if edge.Kind != "required" {
			continue
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}
	var cycles [][]string
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var dfs func(string)
	dfs = func(node string) {
		color[node] = gray
		stack = append(stack, node)
		for _, next := range adj[node] {
			switch color[next] {
			case white:
				dfs(next)
			case gray:
				// cycle: stack from next to end
				start := 0
				for i, id := range stack {
					if id == next {
						start = i
						break
					}
				}
				cycle := append([]string(nil), stack[start:]...)
				cycle = append(cycle, next)
				cycles = append(cycles, cycle)
			}
		}
		stack = stack[:len(stack)-1]
		color[node] = black
	}
	nodes := make([]string, 0, len(adj))
	for node := range adj {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if color[node] == white {
			dfs(node)
		}
	}
	return cycles
}
