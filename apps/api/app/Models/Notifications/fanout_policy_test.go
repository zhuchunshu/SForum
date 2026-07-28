package notifications

import (
	"slices"
	"testing"
)

func TestCoreDeliveryChannelsUseLayeredResolver(t *testing.T) {
	want := []string{"in_app", "email", "web_push"}
	if got := coreDeliveryChannels(); !slices.Equal(got, want) {
		t.Fatalf("core delivery channels=%v want=%v", got, want)
	}
}
