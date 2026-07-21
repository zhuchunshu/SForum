package themecompiler

// Page ViewModels deliberately use only passive, presentation-safe DTOs.
// There are no maps, interface fields, session objects, credentials, or raw
// actor models in this graph. Rich HTML is represented only by sealed SafeHTML.

type PageViewModelKind string

const (
	ViewModelHome          PageViewModelKind = "home"
	ViewModelList          PageViewModelKind = "list"
	ViewModelDetail        PageViewModelKind = "detail"
	ViewModelProfile       PageViewModelKind = "profile"
	ViewModelError         PageViewModelKind = "error"
	ViewModelCreate        PageViewModelKind = "create"

	ViewModelSettings      PageViewModelKind = "settings"
	ViewModelNotifications PageViewModelKind = "notifications"
	ViewModelModeration    PageViewModelKind = "moderation"
	ViewModelAuth          PageViewModelKind = "auth"
	ViewModelLegal         PageViewModelKind = "legal"
	ViewModelDevelopment   PageViewModelKind = "development"
)

type PageViewModelSchema struct {
	PageID        string            `json:"pageId"`
	SchemaVersion string            `json:"schemaVersion"`
	Kind          PageViewModelKind `json:"kind"`
}

type BoundPageViewModel struct {
	pageID             string
	schemaVersion      string
	pluginSchemaDigest string
	themePackageDigest string
	value              any
	seo                PageSEOView
}

func (m BoundPageViewModel) PageID() string        { return m.pageID }
func (m BoundPageViewModel) SchemaVersion() string { return m.schemaVersion }

type PageViewModelBase struct {
	PageID        string           `json:"pageId"`
	SchemaVersion string           `json:"schemaVersion"`
	Locale        string           `json:"locale"`
	Route         PageRouteView    `json:"route"`
	Viewer        PageViewerState  `json:"viewer"`
	Pagination    *PaginationView  `json:"pagination,omitempty"`
	SEO           PageSEOView      `json:"seo"`
	Navigation    []NavigationItem `json:"navigation,omitempty"`
	Breadcrumbs   []BreadcrumbItem `json:"breadcrumbs,omitempty"`
	Regions       []PageRegion     `json:"regions,omitempty"`
}

type PageRouteView struct {
	Path   string       `json:"path"`
	Params []RouteParam `json:"params,omitempty"`
}

type RouteParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PageViewerState is a projection, never the identity Actor/session object.
// Permissions are copied strings used for presentation only; API policy stays
// authoritative for every mutation.
type PageViewerState struct {
	Authenticated bool     `json:"authenticated"`
	UserID        int64    `json:"userId,omitempty"`
	Username      string   `json:"username,omitempty"`
	DisplayName   string   `json:"displayName,omitempty"`
	AvatarURL     string   `json:"avatarUrl,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
}

type PaginationView struct {
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
	Total       int64  `json:"total"`
	PreviousURL string `json:"previousUrl,omitempty"`
	NextURL     string `json:"nextUrl,omitempty"`
}

type PageSEOView struct {
	Title          string               `json:"title"`
	Description    string               `json:"description,omitempty"`
	CanonicalURL   string               `json:"canonicalUrl,omitempty"`
	Robots         string               `json:"robots,omitempty"`
	AlternateLinks []AlternateLink      `json:"alternateLinks,omitempty"`
	StructuredData []StructuredDataView `json:"structuredData,omitempty"`
}

type AlternateLink struct {
	Locale string `json:"locale"`
	URL    string `json:"url"`
}

// StructuredDataView covers the common inspectable fields without accepting
// arbitrary JSON-LD maps from themes or plugins.
type StructuredDataView struct {
	Kind        string `json:"kind"`
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	DateCreated string `json:"dateCreated,omitempty"`
	DateUpdated string `json:"dateUpdated,omitempty"`
}

type NavigationItem struct {
	ID       string           `json:"id"`
	Label    string           `json:"label"`
	URL      string           `json:"url"`
	Active   bool             `json:"active,omitempty"`
	Children []NavigationItem `json:"children,omitempty"`
}

type BreadcrumbItem struct {
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
}

type PageRegion struct {
	ID    string           `json:"id"`
	Items []PageRegionItem `json:"items,omitempty"`
}

// PageRegionItem is intentionally fixed-shape. P9 may version additional
// component props, but a generic props map is never part of a Page ViewModel.
type PageRegionItem struct {
	ComponentID string `json:"componentId"`
	Label       string `json:"label,omitempty"`
	Text        string `json:"text,omitempty"`
	URL         string `json:"url,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

type PublicUserView struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

type TaxonomyLinkView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Count       int64  `json:"count,omitempty"`
}

type TopicSummaryView struct {
	ID         int64              `json:"id"`
	Title      string             `json:"title"`
	URL        string             `json:"url"`
	Excerpt    string             `json:"excerpt,omitempty"`
	Author     PublicUserView     `json:"author"`
	Category   TaxonomyLinkView   `json:"category"`
	Tags       []TaxonomyLinkView `json:"tags,omitempty"`
	ReplyCount int64              `json:"replyCount"`
	CreatedAt  string             `json:"createdAt"`
	UpdatedAt  string             `json:"updatedAt"`
}

type CommentView struct {
	ID        int64          `json:"id"`
	Author    PublicUserView `json:"author"`
	Body      SafeHTML       `json:"body"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

type HomePageViewModel struct {
	Base       PageViewModelBase  `json:"base"`
	Topics     []TopicSummaryView `json:"topics"`
	Categories []TaxonomyLinkView `json:"categories,omitempty"`
	Tags       []TaxonomyLinkView `json:"tags,omitempty"`
	Search     *SearchStateView   `json:"search,omitempty"`
}

type PageListView struct {
	Title       string             `json:"title"`
	Description SafeHTML           `json:"description"`
	Items       []TopicSummaryView `json:"items"`
	Taxonomy    []TaxonomyLinkView `json:"taxonomy,omitempty"`
	Search      *SearchStateView   `json:"search,omitempty"`
}

type CategoryIndexPageViewModel struct {
	Base PageViewModelBase `json:"base"`
	List PageListView      `json:"list"`
}

type CategoryShowPageViewModel struct {
	Base     PageViewModelBase `json:"base"`
	Category TaxonomyLinkView  `json:"category"`
	List     PageListView      `json:"list"`
}

type TagIndexPageViewModel struct {
	Base PageViewModelBase `json:"base"`
	List PageListView      `json:"list"`
}

type TagShowPageViewModel struct {
	Base PageViewModelBase `json:"base"`
	Tag  TaxonomyLinkView  `json:"tag"`
	List PageListView      `json:"list"`
}

type TopicDetailPageViewModel struct {
	Base     PageViewModelBase `json:"base"`
	Topic    TopicSummaryView  `json:"topic"`
	Body     SafeHTML          `json:"body"`
	Comments []CommentView     `json:"comments"`
}

type ProfilePageViewModel struct {
	Base    PageViewModelBase  `json:"base"`
	Profile PublicUserView     `json:"profile"`
	Bio     SafeHTML           `json:"bio"`
	Topics  []TopicSummaryView `json:"topics,omitempty"`
}

type SearchResultView struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Excerpt string `json:"excerpt,omitempty"`
}

// SearchStateView lives inside the existing home/list contracts. There is no
// standalone public search Page Registry identity today.
type SearchStateView struct {
	Query   string             `json:"query"`
	Results []SearchResultView `json:"results"`
}

// HostFormBoundary identifies a Host-owned interactive form island. It never
// carries submitted values, credentials, CSRF material, or session state.
type HostFormBoundary struct {
	ComponentID    string   `json:"componentId"`
	ActionRouteIDs []string `json:"actionRouteIds"`
}

type TopicCreatePageViewModel struct {
	Base       PageViewModelBase  `json:"base"`
	Form       HostFormBoundary   `json:"form"`
	Categories []TaxonomyLinkView `json:"categories"`
	Tags       []TaxonomyLinkView `json:"tags,omitempty"`
}

type ProfileSettingsPageViewModel struct {
	Base    PageViewModelBase `json:"base"`
	Form    HostFormBoundary  `json:"form"`
	Profile PublicUserView    `json:"profile"`
}

type SecurityDeviceView struct {
	Label      string `json:"label"`
	Current    bool   `json:"current"`
	LastSeenAt string `json:"lastSeenAt"`
}

type SecuritySettingsPageViewModel struct {
	Base                 PageViewModelBase    `json:"base"`
	Form                 HostFormBoundary     `json:"form"`
	CredentialConfigured bool                 `json:"credentialConfigured"`
	Devices              []SecurityDeviceView `json:"devices,omitempty"`
}

type NotificationItemView struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"createdAt"`
}

type NotificationsPageViewModel struct {
	Base  PageViewModelBase      `json:"base"`
	Items []NotificationItemView `json:"items"`
}

type ModerationQueueItemView struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type ModerationReviewPageViewModel struct {
	Base  PageViewModelBase         `json:"base"`
	Items []ModerationQueueItemView `json:"items"`
}

type LoginPageViewModel struct {
	Base                PageViewModelBase `json:"base"`
	Form                HostFormBoundary  `json:"form"`
	RegistrationEnabled bool              `json:"registrationEnabled"`
	RecoveryEnabled     bool              `json:"recoveryEnabled"`
}

type RegisterPageViewModel struct {
	Base                PageViewModelBase `json:"base"`
	Form                HostFormBoundary  `json:"form"`
	RegistrationEnabled bool              `json:"registrationEnabled"`
}

type ForgotPasswordPageViewModel struct {
	Base PageViewModelBase `json:"base"`
	Form HostFormBoundary  `json:"form"`
}

type ResetPasswordPageViewModel struct {
	Base           PageViewModelBase `json:"base"`
	Form           HostFormBoundary  `json:"form"`
	ChallengeReady bool              `json:"challengeReady"`
}

type TermsPageViewModel struct {
	Base      PageViewModelBase `json:"base"`
	Content   SafeHTML          `json:"content"`
	UpdatedAt string            `json:"updatedAt"`
}

type PrivacyPageViewModel struct {
	Base      PageViewModelBase `json:"base"`
	Content   SafeHTML          `json:"content"`
	UpdatedAt string            `json:"updatedAt"`
}

type GuidelinesPageViewModel struct {
	Base      PageViewModelBase `json:"base"`
	Content   SafeHTML          `json:"content"`
	UpdatedAt string            `json:"updatedAt"`
}

type ComponentPreviewView struct {
	ComponentID string `json:"componentId"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type DevelopmentComponentsPageViewModel struct {
	Base       PageViewModelBase      `json:"base"`
	Components []ComponentPreviewView `json:"components"`
}

type ErrorPageViewModel struct {
	Base       PageViewModelBase `json:"base"`
	StatusCode int               `json:"statusCode"`
	Title      string            `json:"title"`
	Message    string            `json:"message"`
}
