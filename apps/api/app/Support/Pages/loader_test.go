package pages

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateDataRoute(t *testing.T) {
	if err := ValidateDataRoute("https://evil.com/x"); err == nil {
		t.Fatal("absolute url")
	}
	if err := ValidateDataRoute("//evil.com"); err == nil {
		t.Fatal("protocol relative")
	}
	if err := ValidateDataRoute("../x"); err == nil {
		t.Fatal("traversal")
	}
	if err := ValidateDataRoute("/docs/data"); err != nil {
		t.Fatal(err)
	}
}

func TestLoaderSuccess(t *testing.T) {
	l := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("must not forward cookie")
		}
		if r.Header.Get("X-SForum-Actor-ID-Hint") != "7" {
			t.Fatalf("actor hint: %s", r.Header.Get("X-SForum-Actor-ID-Hint"))
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("must not forward auth headers")
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"title":"ok"}`)),
		}, nil
	}))
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p",
		Route:       "/docs/data",
		ActorID:     7,
		TargetBase:  "http://127.0.0.1:9999",
	})
	if res.Error != "" || string(res.Data) != `{"title":"ok"}` {
		t.Fatalf("%#v", res)
	}
}

func TestLoaderRejectsLargeAndSensitive(t *testing.T) {
	l := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"password":"x"}`)),
		}, nil
	}))
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "http://127.0.0.1:1",
	})
	if res.Error == "" || !res.Fallback {
		t.Fatalf("expected sensitive reject: %#v", res)
	}
}

func TestLoader401(t *testing.T) {
	l := NewPageDataLoader(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	}))
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "http://localhost:1",
	})
	if res.Status != 401 {
		t.Fatalf("%#v", res)
	}
}

func TestLoaderRejectsNonLoopback(t *testing.T) {
	l := NewPageDataLoader(nil)
	res := l.Fetch(context.Background(), LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "https://evil.example",
	})
	if !res.Fallback {
		t.Fatalf("%#v", res)
	}
}
