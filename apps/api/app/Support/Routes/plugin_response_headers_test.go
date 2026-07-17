package routes

import (
	"net/http"
	"reflect"
	"testing"
)

func TestFilterPluginResponseHeadersReservesHostAuthority(t *testing.T) {
	source := http.Header{
		"Set-Cookie":              {"session=forged"},
		"Idempotency-Replayed":    {"true"},
		"X-SForum-Route-Revision": {"forged"},
		"Proxy-Connection":        {"keep-alive"},
		"Connection":              {"X-Hop, X-Second"},
		"connection":              {"X-Third"},
		"X-Hop":                   {"forged"},
		"x-second":                {"forged"},
		"X-Third":                 {"forged"},
		"Link":                    {`</evil>; rel="canonical"`, `</asset>; rel="preload"`},
		"Location":                {"/plugin-terminal"},
		"Cache-Control":           {"private", "max-age=60"},
		"X-Plugin-Metadata":       {"kept"},
	}
	want := http.Header{
		"Location":          {"/plugin-terminal"},
		"Cache-Control":     {"private", "max-age=60"},
		"X-Plugin-Metadata": {"kept"},
	}
	if got := FilterPluginResponseHeaders(source); !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered headers = %#v, want %#v", got, want)
	}
}

func TestFilterPluginResponseHeadersRejectsReservedNamesCaseInsensitively(t *testing.T) {
	for _, name := range []string{
		"Content-Length", "CONNECTION", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
		"Proxy-Connection", "Set-Cookie", "lInK", "Idempotency-Replayed", "TE", "Trailer",
		"Transfer-Encoding", "Upgrade", "X-SForum-Artifact-Digest",
	} {
		t.Run(name, func(t *testing.T) {
			if got := FilterPluginResponseHeaders(http.Header{name: {"forged"}}); len(got) != 0 {
				t.Fatalf("reserved header %q survived: %#v", name, got)
			}
		})
	}
}
