package extensionsruntime

import (
	"context"
	"time"

	"github.com/valyala/fasthttp"
)

type ProxyInput struct {
	Context     context.Context
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
	BaseURL    string
	InstanceID string
}
