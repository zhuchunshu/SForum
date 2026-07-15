package pagescontroller

import (
	nethttp "net/http"
	"strings"
	"testing"
)

func TestPageResolveResponsesArePrivateAcrossAuthenticatedUsers(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	paths := []string{
		"/api/v1/pages/resolve?id=forum.home",
		"/api/v1/pages/resolve-path?path=/demo-members",
	}

	for _, userID := range []int64{1, 2} {
		cookie := loginPagesUser(t, app, userID)
		for _, path := range paths {
			response := performPages(t, app, nethttp.MethodGet, path, nil, cookie)
			if response.StatusCode != nethttp.StatusOK {
				t.Fatalf("user %d path %s status = %d", userID, path, response.StatusCode)
			}
			if got := response.Header.Get("Cache-Control"); got != "private, no-store" {
				t.Fatalf("user %d path %s Cache-Control = %q", userID, path, got)
			}
			assertVaryFields(t, response.Header.Get("Vary"), "Cookie", "Authorization", "Accept-Language")
		}
	}
}

func assertVaryFields(t *testing.T, value string, expected ...string) {
	t.Helper()
	fields := make(map[string]struct{})
	for _, field := range strings.Split(value, ",") {
		fields[strings.ToLower(strings.TrimSpace(field))] = struct{}{}
	}
	for _, field := range expected {
		if _, ok := fields[strings.ToLower(field)]; !ok {
			t.Fatalf("Vary = %q, missing %s", value, field)
		}
	}
}
