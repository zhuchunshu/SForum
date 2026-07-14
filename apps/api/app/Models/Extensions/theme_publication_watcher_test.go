package extensions

import (
	"errors"
	"strings"
	"testing"
)

func TestThemeRuntimeFailureReasonPreservesUTF8WithinDatabaseBound(t *testing.T) {
	reason := themeRuntimeFailureReason(errors.New(strings.Repeat("主题失败", 1000)))
	if reason == "" || len([]byte(reason)) > 2048 || !strings.HasPrefix(reason, "主题失败") {
		t.Fatalf("bounded reason bytes=%d value=%q", len([]byte(reason)), reason)
	}
}
