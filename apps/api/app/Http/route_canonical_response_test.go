package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestWriteRouteDispatchResponseEnforcesHostCanonicalLink(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		writeRouteDispatchResponse(c, routes.DispatchResponse{
			Status: stdhttp.StatusOK,
			Headers: stdhttp.Header{
				fiber.HeaderLink: {"<https://evil.example/>; rel=\"canonical\""},
			},
			CanonicalPath: "/topics/中文",
		}, nil)
		return nil
	})

	response, err := app.Test(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if got := response.Header.Values(fiber.HeaderLink); len(got) != 1 || got[0] != "</topics/%E4%B8%AD%E6%96%87>; rel=\"canonical\"" {
		t.Fatalf("Link = %#v", got)
	}
}

func TestRouteCanonicalLinkPathRejectsExternalQueryFragmentAndControls(t *testing.T) {
	for _, path := range []string{"", "relative", "//evil.example/path", "https://evil.example/path", "/path?query=1", "/path#fragment", "/bad\r\nvalue"} {
		if value, ok := routeCanonicalLinkPath(path); ok || value != "" {
			t.Fatalf("canonical path %q = %q, %v", path, value, ok)
		}
	}
	for _, test := range []struct{ path, want string }{
		{path: "/topics/中文", want: "/topics/%E4%B8%AD%E6%96%87"},
		{path: "/topics/%E4%B8%AD%E6%96%87", want: "/topics/%E4%B8%AD%E6%96%87"},
	} {
		if got, ok := routeCanonicalLinkPath(test.path); !ok || got != test.want {
			t.Fatalf("canonical path %q = %q, %v", test.path, got, ok)
		}
	}
}
