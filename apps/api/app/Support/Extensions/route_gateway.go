package extensionsruntime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RouteGateway struct {
	client *http.Client
}

func NewRouteGateway() *RouteGateway {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// 插件路由只允许直连握手校验过的 loopback target，不继承部署代理。
	transport.Proxy = nil
	return &RouteGateway{client: &http.Client{Transport: transport}}
}

func (g *RouteGateway) Proxy(input *ProxyInput) error {
	if g == nil || g.client == nil || input == nil || input.Request == nil || input.Response == nil {
		return fmt.Errorf("extension route proxy input is invalid")
	}
	target, err := url.Parse(input.TargetBase)
	if err != nil {
		return err
	}
	relative, err := url.Parse(input.TargetPath)
	if err != nil || relative.IsAbs() || relative.Host != "" {
		return fmt.Errorf("extension route target path is invalid")
	}
	target.Path = relative.Path
	target.RawPath = relative.RawPath
	target.RawQuery = relative.RawQuery
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, string(input.Request.Header.Method()), target.String(), bytes.NewReader(input.Request.Body()))
	if err != nil {
		return err
	}
	input.Request.Header.VisitAll(func(key, value []byte) {
		name := string(key)
		if routeRequestHeaderAllowed(name) {
			request.Header.Add(name, string(value))
		}
	})
	request.Header.Set("X-SForum-Extension-ID", input.ExtensionID)
	if input.ActorID != "" {
		request.Header.Set("X-SForum-Actor-ID", input.ActorID)
	}
	if input.Locale != "" {
		request.Header.Set("X-SForum-Locale", input.Locale)
	}
	response, err := g.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	input.Response.Reset()
	input.Response.SetStatusCode(response.StatusCode)
	for name, values := range response.Header {
		if !routeResponseHeaderAllowed(name) {
			continue
		}
		for _, value := range values {
			input.Response.Header.Add(name, value)
		}
	}
	input.Response.SetBodyRaw(body)
	return nil
}

func routeRequestHeaderAllowed(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "host", "content-length", "cookie", "authorization", "proxy-authorization",
		"x-sforum-actor-id", "x-sforum-extension-id", "x-sforum-locale", "x-api-key", "x-auth-token",
		"connection", "keep-alive", "proxy-authenticate", "te", "trailer", "transfer-encoding", "upgrade":
		return false
	default:
		return true
	}
}

func routeResponseHeaderAllowed(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "content-length", "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return false
	default:
		return true
	}
}
