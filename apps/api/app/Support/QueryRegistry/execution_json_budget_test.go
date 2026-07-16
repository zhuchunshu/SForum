package queryregistry

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

type expandingQueryJSON string

func (expandingQueryJSON) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strings.Repeat("x", 4096) + `"`), nil
}

type addressableExpandingQueryJSON string

var addressableExpandingQueryJSONCalls atomic.Int32

func (*addressableExpandingQueryJSON) MarshalJSON() ([]byte, error) {
	addressableExpandingQueryJSONCalls.Add(1)
	return []byte(`"` + strings.Repeat("x", 4096) + `"`), nil
}

func TestCloneRowsPreflightRejectsCyclesAndCustomMarshalers(t *testing.T) {
	cyclic := QueryRow{"id": "1"}
	cyclic["self"] = cyclic
	if _, _, err := cloneRowsBounded([]QueryRow{cyclic}, 4096); !errors.Is(err, ErrResultTooLarge) {
		t.Fatalf("cyclic result=%v", err)
	}
	if _, _, err := cloneRowsBounded([]QueryRow{{"id": expandingQueryJSON("small")}}, 1024); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("custom marshaler result=%v", err)
	}
	addressableExpandingQueryJSONCalls.Store(0)
	values := []addressableExpandingQueryJSON{"small"}
	if _, _, err := cloneRowsBounded([]QueryRow{{"id": "1", "values": values}}, 1024); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("addressable custom marshaler result=%v", err)
	}
	if calls := addressableExpandingQueryJSONCalls.Load(); calls != 0 {
		t.Fatalf("addressable custom marshaler ran before Host budget rejection: calls=%d", calls)
	}
	if _, _, err := cloneRowsBounded([]QueryRow{{"id": "1", "value": strings.Repeat("<", 300)}}, 1024); !errors.Is(err, ErrResultTooLarge) {
		t.Fatalf("escaped expansion result=%v", err)
	}
}
