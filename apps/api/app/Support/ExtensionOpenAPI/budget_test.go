package extensionopenapi

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceBudgetRejectsDocumentCountTotalBytesAndReferences(t *testing.T) {
	count := &resourceBudget{}
	for range maxDocuments {
		if err := count.reserveDocument(1); err != nil {
			t.Fatal(err)
		}
	}
	if err := count.reserveDocument(1); !errors.Is(err, ErrResourceBudget) {
		t.Fatalf("document count error = %v", err)
	}

	total := &resourceBudget{}
	for range maxAggregateInputBytes / maxDocumentBytes {
		if err := total.reserveDocument(maxDocumentBytes); err != nil {
			t.Fatal(err)
		}
	}
	if err := total.reserveDocument(1); !errors.Is(err, ErrResourceBudget) {
		t.Fatalf("total byte error = %v", err)
	}

	references := &resourceBudget{}
	if err := references.reserveReferences(maxReferences); err != nil {
		t.Fatal(err)
	}
	if err := references.reserveReferences(1); !errors.Is(err, ErrResourceBudget) {
		t.Fatalf("reference count error = %v", err)
	}
}

func TestReadLimitedRegularFileStatsAndLimitsBeforeAllocation(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "oversized.yaml")
	handle, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Truncate(maxDocumentBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	budget := &resourceBudget{}
	if _, err := readLimitedRegularFile(target, maxDocumentBytes, budget); !errors.Is(err, ErrResourceBudget) {
		t.Fatalf("oversized stat error = %v", err)
	}
	if budget.documents != 0 || budget.bytes != 0 {
		t.Fatalf("oversized file consumed budget after stat rejection: %#v", budget)
	}
}

func TestBuildRejectsOversizedOpenAPIDocumentBeforeRead(t *testing.T) {
	options := defaultFixtureOptions("budget.document")
	base := fixtureDocument(options)
	options.document = base + "\n# " + strings.Repeat("x", int(maxDocumentBytes)-len(base))
	fixture := buildFixture(t, options)
	if _, err := Build(BuildInput{Artifacts: []Artifact{fixture}}); !errors.Is(err, ErrResourceBudget) {
		t.Fatalf("oversized aggregate error = %v", err)
	}
}

func TestDecodeRejectsExcessiveRecursiveDepth(t *testing.T) {
	body := strings.Repeat(`{"nested":`, maxDocumentDepth+2) + `true` + strings.Repeat(`}`, maxDocumentDepth+2)
	if _, err := decodeYAMLOrJSON([]byte(body)); !errors.Is(err, ErrResourceBudget) {
		t.Fatalf("depth error = %v", err)
	}
}
