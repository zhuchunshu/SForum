package extensionsruntime

import (
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type RouteGateway struct{}

func NewRouteGateway() *RouteGateway {
	return &RouteGateway{}
}

func (g *RouteGateway) Proxy(input *ProxyInput) error {
	target, err := url.Parse(input.TargetBase)
	if err != nil {
		return err
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)

	input.Request.CopyTo(request)
	request.SetRequestURI(input.TargetPath)
	request.URI().SetScheme(target.Scheme)
	request.URI().SetHost(target.Host)
	// 先剥离客户端可伪造的身份/鉴权头，再只写入宿主权威值，避免匿名路由伪造 Actor-ID。
	for _, header := range []string{
		"Cookie",
		"Authorization",
		"Proxy-Authorization",
		"X-SForum-Actor-ID",
		"X-SForum-Extension-ID",
		"X-SForum-Locale",
		"X-Api-Key",
		"X-Auth-Token",
	} {
		request.Header.Del(header)
	}
	request.Header.Set("X-SForum-Extension-ID", input.ExtensionID)
	if input.ActorID != "" {
		request.Header.Set("X-SForum-Actor-ID", input.ActorID)
	}
	if input.Locale != "" {
		request.Header.Set("X-SForum-Locale", input.Locale)
	}
	for _, header := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade"} {
		request.Header.Del(header)
	}
	client := fasthttp.HostClient{Addr: target.Host}
	if strings.EqualFold(target.Scheme, "https") {
		client.IsTLS = true
	}
	return client.DoTimeout(request, input.Response, timeout)
}
