package extensions

import "testing"

func TestFrontendRequiresWebRelease(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{FrontendTrustNone, false},
		{FrontendTrustRequired, false},
		{FrontendTrustInvalidated, false},
		{FrontendTrustTrusted, true},
		{FrontendTrustSourceTrusted, true},
		{FrontendTrustRevocationPending, true},
	}
	for _, test := range tests {
		if got := frontendRequiresWebRelease(FrontendStatus{TrustState: test.state}); got != test.want {
			t.Fatalf("state %s: expected %v, got %v", test.state, test.want, got)
		}
	}
}
