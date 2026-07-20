package extensionmanifest

const (
	RouteActionAdd              = "add"
	RouteActionAlias            = "alias"
	RouteActionRedirect         = "redirect"
	RouteActionRewrite          = "rewrite"
	RouteActionBefore           = "before"
	RouteActionAfter            = "after"
	RouteActionFilter           = "filter"
	RouteActionWrap             = "wrap"
	RouteActionReplace          = "replace"
	RouteActionGlobalMiddleware = "global_middleware"
	RouteRedirectStatusDefault  = 308

	RouteModeHTTP      = "http"
	RouteModeSSE       = "sse"
	RouteModeWebSocket = "websocket"
	RouteModeStream    = "stream"
	RouteModeMultipart = "multipart"

	GuardCorePublic     = "core.guard.public"
	GuardCoreLogin      = "core.guard.login"
	GuardCorePermission = "core.guard.permission"
	GuardCoreGuest      = "core.guard.guest"
	GuardCoreRaw        = "core.guard.raw_request"
	GuardCoreInherit    = "core.guard.inherit"

	ComponentActionAdd          = "add"
	ComponentActionBefore       = "before"
	ComponentActionAfter        = "after"
	ComponentActionWrap         = "wrap"
	ComponentActionReplace      = "replace"
	ComponentActionHide         = "hide"
	ComponentActionFilterProps  = "filter_props"
	ComponentActionFilterResult = "filter_result"
)

type ManifestGuard struct {
	ID              string   `json:"id"`
	ContractVersion string   `json:"contractVersion"`
	Kind            string   `json:"kind"`
	Entry           string   `json:"entry,omitempty"`
	Digest          string   `json:"digest,omitempty"`
	Permissions     []string `json:"permissions,omitempty"`
}

type ManifestSchedule struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	JobID           string `json:"jobId"`
	Cron            string `json:"cron"`
	Timezone        string `json:"timezone,omitempty"`
}

type ManifestComponent struct {
	ID                    string `json:"id"`
	ContractVersion       string `json:"contractVersion"`
	Action                string `json:"action"`
	TargetID              string `json:"targetId,omitempty"`
	TargetContractVersion string `json:"targetContractVersion,omitempty"`
	Priority              int    `json:"priority,omitempty"`
	Permission            string `json:"permission,omitempty"`
	SSRTemplate           string `json:"ssrTemplate,omitempty"`
	L2Component           string `json:"l2Component,omitempty"`
	PropsSchema           string `json:"propsSchema,omitempty"`
	ResultSchema          string `json:"resultSchema,omitempty"`
	ThemeOverrideKey      string `json:"themeOverrideKey,omitempty"`
}

type ManifestTemplate struct {
	ID               string `json:"id"`
	ContractVersion  string `json:"contractVersion"`
	Action           string `json:"action"`
	TargetID         string `json:"targetId,omitempty"`
	Path             string `json:"path"`
	Digest           string `json:"digest"`
	ViewModelSchema  string `json:"viewModelSchema"`
	ThemeOverrideKey string `json:"themeOverrideKey,omitempty"`
}

type ManifestAsset struct {
	Handle          string   `json:"handle"`
	ContractVersion string   `json:"contractVersion"`
	Type            string   `json:"type"`
	Path            string   `json:"path"`
	Digest          string   `json:"digest"`
	Dependencies    []string `json:"dependencies,omitempty"`
	Scope           []string `json:"scope,omitempty"`
	Module          bool     `json:"module,omitempty"`
	Loading         string   `json:"loading,omitempty"`
	Integrity       string   `json:"integrity,omitempty"`
	CSP             []string `json:"csp,omitempty"`
}

type ManifestContent struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Handler         string `json:"handler,omitempty"`
	Schema          string `json:"schema"`
	Renderer        string `json:"renderer,omitempty"`
	Migration       string `json:"migration,omitempty"`
}

// ManifestEditor declares a Tiptap trusted L2 editor surface (node/mark/command/toolbar).
// Prebuilt modules bind through packageFiles (frontend) by path+digest; Content
// Registry remains separate for block/shortcode/embed storage kinds.
type ManifestEditor struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Kind            string `json:"kind"`
	Schema          string `json:"schema,omitempty"`
	ExtensionName   string `json:"extensionName,omitempty"`
	L2Module        string `json:"l2Module,omitempty"`
	L2Digest        string `json:"l2Digest,omitempty"`
	CommandKey      string `json:"commandKey,omitempty"`
	CommandID       string `json:"commandId,omitempty"`
	Label           string `json:"label,omitempty"`
	Icon            string `json:"icon,omitempty"`
	Group           string `json:"group,omitempty"`
	Order           int    `json:"order,omitempty"`
	Priority        int    `json:"priority,omitempty"`
	Permission      string `json:"permission,omitempty"`
}

type ManifestService struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Action          string `json:"action"`
	TargetID        string `json:"targetId,omitempty"`
	Handler         string `json:"handler"`
	RequestSchema   string `json:"requestSchema"`
	ResponseSchema  string `json:"responseSchema"`
	Priority        int    `json:"priority,omitempty"`
}

type ManifestCommand struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Handler         string `json:"handler"`
	Permission      string `json:"permission,omitempty"`
	InputSchema     string `json:"inputSchema"`
	ResultSchema    string `json:"resultSchema"`
	Description     string `json:"description,omitempty"`
	RecoverySafe    bool   `json:"recoverySafe,omitempty"`
	TimeoutMS       int    `json:"timeoutMs,omitempty"`
}

type ManifestPackageFile struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Digest  string `json:"digest"`
	Locale  string `json:"locale,omitempty"`
	Version string `json:"version,omitempty"`
}

type ManifestOpenAPIFragment struct {
	ID              string `json:"id"`
	ContractVersion string `json:"contractVersion"`
	Path            string `json:"path"`
	Digest          string `json:"digest"`
	Namespace       string `json:"namespace"`
}
