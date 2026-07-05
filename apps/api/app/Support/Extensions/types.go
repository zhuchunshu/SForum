package extensionsruntime

import (
	"time"

	"github.com/valyala/fasthttp"
)

type ProxyInput struct {
	Request     *fasthttp.Request
	Response    *fasthttp.Response
	ExtensionID string
	ActorID     string
	Locale      string
	TargetBase  string
	TargetPath  string
	Timeout     time.Duration
}

type RouteTarget struct {
	BaseURL string
}
