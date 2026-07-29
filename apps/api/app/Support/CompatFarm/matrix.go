// Package compatfarm loads the V3 P12 multi-version compatibility matrix.
package compatfarm

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the matrix document contract.
const SchemaVersion = "sforum.compat-farm@1"

// Matrix is the root farm document.
type Matrix struct {
	Version int    `yaml:"version" json:"version"`
	Schema  string `yaml:"schema" json:"schema"`
	Cells   []Cell `yaml:"cells" json:"cells"`
}

// Cell is one compatibility farm cell.
type Cell struct {
	ID       string `yaml:"id" json:"id"`
	SForum   string `yaml:"sforum" json:"sforum"`
	Protocol string `yaml:"protocol" json:"protocol"`
	Manifest string `yaml:"manifest" json:"manifest"`
	Database string `yaml:"database" json:"database"`
	Browser  string `yaml:"browser" json:"browser"`
	Status   string `yaml:"status" json:"status"` // required|deprecated
	Command  string `yaml:"command" json:"command,omitempty"`
}

// LoadMatrix reads a YAML matrix file.
func LoadMatrix(path string) (Matrix, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Matrix{}, err
	}
	var matrix Matrix
	if err := yaml.Unmarshal(raw, &matrix); err != nil {
		return Matrix{}, err
	}
	if matrix.Schema != "" && matrix.Schema != SchemaVersion {
		return Matrix{}, fmt.Errorf("compat farm schema %q want %s", matrix.Schema, SchemaVersion)
	}
	if matrix.Version < 1 || len(matrix.Cells) == 0 {
		return Matrix{}, fmt.Errorf("compat farm matrix is empty")
	}
	seen := map[string]struct{}{}
	for i := range matrix.Cells {
		cell := &matrix.Cells[i]
		cell.ID = strings.TrimSpace(cell.ID)
		cell.Status = strings.ToLower(strings.TrimSpace(cell.Status))
		if cell.ID == "" || (cell.Status != "required" && cell.Status != "deprecated") {
			return Matrix{}, fmt.Errorf("invalid cell at index %d", i)
		}
		if _, dup := seen[cell.ID]; dup {
			return Matrix{}, fmt.Errorf("duplicate cell id %s", cell.ID)
		}
		seen[cell.ID] = struct{}{}
	}
	return matrix, nil
}

// RequiredCells returns cells that must pass on every CI run.
func (m Matrix) RequiredCells() []Cell {
	out := make([]Cell, 0, len(m.Cells))
	for _, cell := range m.Cells {
		if cell.Status == "required" {
			out = append(out, cell)
		}
	}
	return out
}

// DeprecatedCells returns deprecated fixture cells.
func (m Matrix) DeprecatedCells() []Cell {
	out := make([]Cell, 0)
	for _, cell := range m.Cells {
		if cell.Status == "deprecated" {
			out = append(out, cell)
		}
	}
	return out
}
