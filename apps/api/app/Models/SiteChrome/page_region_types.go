package sitechrome

// PageRegionWidgetRef 引用插件的公开 L2 组件;挂载时由公开 L2 运行时按信任授予权威裁决。
type PageRegionWidgetRef struct {
	ExtensionID string `json:"extensionId"`
	ComponentID string `json:"componentId"`
}

// PageRegionItem 是宿主拥有的区域内容描述符(forum.page.regions 贡献解析结果)。
// Kind=link:站内相对链接卡;Kind=action:走 /extensions/{id}{path} 代理的动作卡;
// Kind=widget:L2 组件引用,无任何可执行 payload。
type PageRegionItem struct {
	ExtensionID    string               `json:"extensionId"`
	ContributionID string               `json:"contributionId"`
	Label          map[string]string    `json:"label,omitempty"`
	Icon           string               `json:"icon,omitempty"`
	Kind           string               `json:"kind"`
	Href           string               `json:"href,omitempty"`
	Method         string               `json:"method,omitempty"`
	Path           string               `json:"path,omitempty"`
	Widget         *PageRegionWidgetRef `json:"widget,omitempty"`
	Order          int                  `json:"order"`
}

// PageRegionViewModel 是一个标准区域及其按序内容。
type PageRegionViewModel struct {
	ID    string           `json:"id"`
	Kind  string           `json:"kind"`
	Items []PageRegionItem `json:"items"`
}
