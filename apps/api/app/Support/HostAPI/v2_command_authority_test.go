package hostapi

import (
	"reflect"
	"testing"
)

func TestProtocolV2CommandDatabaseGrantsNormalizeLegacyAndAdditiveAuthority(t *testing.T) {
	tests := []struct {
		name     string
		document string
		want     []string
		valid    bool
	}{
		{name: "additive exact", document: `{"grants":["core_views","host_commands"]}`, want: []string{"core_views", "host_commands"}, valid: true},
		{name: "legacy cumulative", document: `{"authority":"raw_core"}`, want: []string{"own_schema", "core_views", "host_commands", "raw_core"}, valid: true},
		{name: "unknown", document: `{"grants":["host_commands","unknown"]}`},
		{name: "duplicate", document: `{"grants":["host_commands","host_commands"]}`},
		{name: "empty", document: `{}`},
		{name: "null", document: `null`},
		{name: "malformed", document: `{`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grants, err := protocolV2CommandDatabaseGrants([]byte(test.document))
			if test.valid && (err != nil || !reflect.DeepEqual(grants, test.want)) {
				t.Fatalf("grants = %#v, %v", grants, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("invalid grants accepted: %#v", grants)
			}
		})
	}
}
