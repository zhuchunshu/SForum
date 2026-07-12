package extensionmanifest

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestContributionPointDefinitionsContainJobsProductionSlots(t *testing.T) {
	points := ContributionPointDefinitions()
	want := map[string]bool{
		PointForumTopicActions:       true,
		PointForumComposerToolbar:    true,
		PointForumProfileTabs:        true,
		PointAdminDashboardWidgets:   true,
		PointSystemHealthChecks:      true,
		"admin.jobs.table.columns":   true,
		"admin.jobs.row.actions":     true,
		"admin.jobs.detail.sections": true,
	}
	for _, point := range points {
		delete(want, point.ID)
		if point.ID == "admin.test.fixture" {
			t.Fatal("test point leaked into production catalog")
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing production contribution points: %#v", want)
	}
}

func TestF43ContributionPayloadValidation(t *testing.T) {
	base := Manifest{
		ID:            "demo.f43",
		Name:          "F43 Demo",
		Description:   "F4.3 contribution payload tests.",
		URL:           "https://example.com/f43",
		Author:        ManifestAuthor{Name: "Demo"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
		Routes: []ManifestRoute{
			{Path: "/actions/x", Methods: []string{"POST"}, Access: RouteAccessLogin},
		},
	}

	okCases := [][]ManifestContribution{
		{{
			Point: PointForumComposerToolbar, ID: "demo.composer", Order: 10,
			Label: map[string]string{"en-US": "Insert"}, Icon: "i-lucide-wand",
			Payload: json.RawMessage(`{"type":"extensionRoute","method":"POST","path":"/actions/x"}`),
		}},
		{{
			Point: PointForumProfileTabs, ID: "demo.tab", Order: 10,
			Label: map[string]string{"en-US": "Extra"}, Icon: "i-lucide-user",
			Payload: json.RawMessage(`{"type":"hostLink","href":"/tags"}`),
		}},
		{{
			Point: PointAdminDashboardWidgets, ID: "demo.widget", Order: 10,
			Label: map[string]string{"en-US": "Queue"}, Icon: "i-lucide-gauge",
			Payload: json.RawMessage(`{"type":"adminLink","route":"/jobs","severity":"info"}`),
		}},
		{{
			Point: PointSystemHealthChecks, ID: "demo.health", Order: 10,
			Label:   map[string]string{"en-US": "Demo health"},
			Payload: json.RawMessage(`{"type":"static","component":"extension.demo.f43"}`),
		}},
	}
	for i, contributions := range okCases {
		m := base
		m.Contributions = contributions
		if err := Validate(m); err != nil {
			t.Fatalf("ok case %d should validate: %v", i, err)
		}
	}

	bad := base
	bad.Contributions = []ManifestContribution{{
		Point: PointAdminDashboardWidgets, ID: "evil", Order: 1,
		Label:   map[string]string{"en-US": "Evil"},
		Payload: json.RawMessage(`{"type":"adminLink","route":"https://evil.example/","severity":"info"}`),
	}}
	if err := Validate(bad); err == nil {
		t.Fatal("external dashboard route must be rejected")
	}
	bad.Contributions = []ManifestContribution{{
		Point: PointSystemHealthChecks, ID: "evil2", Order: 1,
		Label:   map[string]string{"en-US": "Evil"},
		Payload: json.RawMessage(`{"type":"rpc","component":"x"}`),
	}}
	if err := Validate(bad); err == nil {
		t.Fatal("executable/unknown health type must be rejected")
	}
}

func TestManifestLangsOptionalAndLocalizedDisplay(t *testing.T) {
	// 无 langs 时保持顶层默认，不要求翻译。
	withoutLangs := Manifest{
		ID:            "demo.plugin",
		Name:          "Demo Plugin",
		Description:   "Demo plugin.",
		URL:           "https://example.com/demo",
		Author:        ManifestAuthor{Name: "Demo Studio", URL: "https://example.com"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
	}
	if err := Validate(withoutLangs); err != nil {
		t.Fatalf("manifest without langs should validate: %v", err)
	}
	display := LocalizedDisplay(withoutLangs, "zh-CN")
	if display.Name != "Demo Plugin" || display.Description != "Demo plugin." {
		t.Fatalf("expected top-level defaults without langs, got %#v", display)
	}

	body := []byte(`{
		"id":"demo.plugin",
		"name":"Demo Plugin",
		"description":"Demo plugin.",
		"url":"https://example.com/demo",
		"author":{"name":"Demo Studio","url":"https://example.com"},
		"version":"1.0.0",
		"type":"plugin",
		"sforumVersion":"^1.0.0",
		"langs":{
			"zh":{
				"name":"演示插件",
				"description":"演示插件说明。",
				"author":{"name":"演示工作室"}
			}
		}
	}`)
	var withLangs Manifest
	if err := json.Unmarshal(body, &withLangs); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if err := Validate(withLangs); err != nil {
		t.Fatalf("manifest with langs should validate: %v", err)
	}
	normalized := Normalize(withLangs)
	if _, ok := normalized.Langs["zh"]; !ok {
		t.Fatalf("expected normalized zh locale, got %#v", normalized.Langs)
	}

	// zh-CN 应回退匹配 langs.zh。
	zhDisplay := LocalizedDisplay(withLangs, "zh-CN")
	if zhDisplay.Name != "演示插件" || zhDisplay.Description != "演示插件说明。" || zhDisplay.Author.Name != "演示工作室" {
		t.Fatalf("expected zh overrides, got %#v", zhDisplay)
	}
	// 未覆盖的 author.url 保留顶层默认。
	if zhDisplay.Author.URL != "https://example.com" {
		t.Fatalf("expected author url fallback, got %q", zhDisplay.Author.URL)
	}
	// en 无覆盖时用顶层默认。
	enDisplay := LocalizedDisplay(withLangs, "en-US")
	if enDisplay.Name != "Demo Plugin" || enDisplay.Description != "Demo plugin." {
		t.Fatalf("expected english defaults, got %#v", enDisplay)
	}

	// 无效语言码应拒绝。
	bad := withLangs
	bad.Langs = map[string]ManifestLocale{"   ": {Name: "x"}}
	if err := Validate(bad); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid locale key to fail, got %v", err)
	}
}

func TestAdminManifestV2NormalizeValidateAndResolveManagePath(t *testing.T) {
	body := []byte(`{
		"id":"demo.plugin",
		"name":"Demo Plugin",
		"description":"Demo plugin.",
		"url":"https://example.com/demo",
		"author":{"name":"Demo Studio"},
		"version":"1.0.0",
		"type":"plugin",
		"sforumVersion":"^1.0.0",
		"admin":{
			"entry":"settings",
			"pages":[
				{"path":"settings","label":"Settings","view":"settings"},
				{"path":"/dashboard","label":"Dashboard","view":"about","menu":true,"icon":"i-lucide-layout-dashboard","order":20}
			]
		}
	}`)

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	normalized := Normalize(manifest)
	if normalized.Admin.Entry != "/settings" {
		t.Fatalf("expected normalized admin entry, got %q", normalized.Admin.Entry)
	}
	pages := EffectiveAdminPages(normalized)
	if len(pages) != 2 {
		t.Fatalf("expected two effective pages, got %#v", pages)
	}
	if pages[0].Path != "/settings" || pages[0].Menu {
		t.Fatalf("settings page should normalize with menu false: %#v", pages[0])
	}
	if pages[1].Path != "/dashboard" || !pages[1].Menu {
		t.Fatalf("dashboard page should be an explicit menu page: %#v", pages[1])
	}
	if AdminManagePath(normalized) != "/settings" {
		t.Fatalf("expected entry to drive manage path, got %q", AdminManagePath(normalized))
	}
	menuPages := MenuAdminPages(normalized)
	if len(menuPages) != 1 || menuPages[0].Path != "/dashboard" {
		t.Fatalf("expected only explicit menu page, got %#v", menuPages)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("v2 admin manifest should validate: %v", err)
	}
}

func TestAdminManifestV2RejectsBrokenEntryAndExternalPaths(t *testing.T) {
	base := Manifest{
		ID:            "demo.plugin",
		Name:          "Demo Plugin",
		Description:   "Demo plugin.",
		URL:           "https://example.com/demo",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
	}

	cases := []struct {
		name     string
		manifest Manifest
	}{
		{
			name: "entry must target declared page or about",
			manifest: func() Manifest {
				next := base
				next.Admin.Entry = "/missing"
				next.Admin.Pages = []ManifestAdminPage{{Path: "/settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
		{
			name: "entry cannot be external url",
			manifest: func() Manifest {
				next := base
				next.Admin.Entry = "https://example.com/settings"
				next.Admin.Pages = []ManifestAdminPage{{Path: "/settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
		{
			name: "page cannot contain traversal",
			manifest: func() Manifest {
				next := base
				next.Admin.Pages = []ManifestAdminPage{{Path: "/../settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}

func TestLegacyAdminPagesRemainCompatible(t *testing.T) {
	manifest := Manifest{
		ID:            "legacy.plugin",
		Name:          "Legacy Plugin",
		Description:   "Legacy plugin.",
		URL:           "https://example.com/legacy",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
		AdminPages: []ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Menu: true},
		},
	}

	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy adminPages should validate: %v", err)
	}
	normalized := Normalize(manifest)
	if AdminManagePath(normalized) != "/settings" {
		t.Fatalf("expected legacy settings page as manage path, got %q", AdminManagePath(normalized))
	}
	menuPages := MenuAdminPages(normalized)
	if len(menuPages) != 1 || menuPages[0].Path != "/settings" {
		t.Fatalf("expected legacy explicit menu page, got %#v", menuPages)
	}
}

func TestManifestContributionsNormalizeAndValidateTopicAction(t *testing.T) {
	body := []byte(`{
		"id":"demo.plugin",
		"name":"Demo Plugin",
		"description":"Demo plugin.",
		"url":"https://example.com/demo",
		"author":{"name":"Demo Studio"},
		"version":"1.0.0",
		"type":"plugin",
		"sforumVersion":"^1.0.0",
		"contributions":[
			{
				"point":" forum.topic.actions ",
				"id":" Bookmark.Action ",
				"order":200,
				"label":{"zh-CN":"收藏","en-US":"Bookmark"},
				"icon":" i-lucide-bookmark ",
				"payload":{
					"type":"extensionRoute",
					"method":"post",
					"path":"topic-actions/bookmark",
					"confirm":true
				}
			}
		]
	}`)

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	normalized := Normalize(manifest)
	if len(normalized.Contributions) != 1 {
		t.Fatalf("expected one contribution, got %#v", normalized.Contributions)
	}
	contribution := normalized.Contributions[0]
	if contribution.Point != "forum.topic.actions" || contribution.ID != "bookmark.action" || contribution.Icon != "i-lucide-bookmark" {
		t.Fatalf("contribution was not normalized: %#v", contribution)
	}
	var payload TopicActionContributionPayload
	if err := json.Unmarshal(contribution.Payload, &payload); err != nil {
		t.Fatalf("decode normalized payload: %v", err)
	}
	if payload.Method != "POST" || payload.Path != "/topic-actions/bookmark" || !payload.Confirm {
		t.Fatalf("payload was not normalized: %#v", payload)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("valid contribution manifest should validate: %v", err)
	}
}

func TestManifestContributionsRejectUnsafeDeclarations(t *testing.T) {
	base := Manifest{
		ID:            "demo.plugin",
		Name:          "Demo Plugin",
		Description:   "Demo plugin.",
		URL:           "https://example.com/demo",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
	}
	valid := ManifestContribution{
		Point: "forum.topic.actions",
		ID:    "demo.bookmark",
		Label: map[string]string{"zh-CN": "收藏", "en-US": "Bookmark"},
		Icon:  "i-lucide-bookmark",
		Payload: json.RawMessage(`{
			"type":"extensionRoute",
			"method":"POST",
			"path":"/topic-actions/bookmark"
		}`),
	}

	cases := []struct {
		name         string
		contribution ManifestContribution
		extra        []ManifestContribution
	}{
		{
			name: "unknown contribution point",
			contribution: func() ManifestContribution {
				next := valid
				next.Point = "forum.unknown"
				return next
			}(),
		},
		{
			name:         "duplicate point and id inside one manifest",
			contribution: valid,
			extra:        []ManifestContribution{valid},
		},
		{
			name: "missing point",
			contribution: func() ManifestContribution {
				next := valid
				next.Point = ""
				return next
			}(),
		},
		{
			name: "missing id",
			contribution: func() ManifestContribution {
				next := valid
				next.ID = ""
				return next
			}(),
		},
		{
			name: "external payload path",
			contribution: func() ManifestContribution {
				next := valid
				next.Payload = json.RawMessage(`{"type":"extensionRoute","method":"POST","path":"https://example.com/action"}`)
				return next
			}(),
		},
		{
			name: "payload path traversal",
			contribution: func() ManifestContribution {
				next := valid
				next.Payload = json.RawMessage(`{"type":"extensionRoute","method":"POST","path":"/../action"}`)
				return next
			}(),
		},
		{
			name: "payload path targets core api",
			contribution: func() ManifestContribution {
				next := valid
				next.Payload = json.RawMessage(`{"type":"extensionRoute","method":"POST","path":"/api/v1/topics/1"}`)
				return next
			}(),
		},
		{
			name: "unknown method",
			contribution: func() ManifestContribution {
				next := valid
				next.Payload = json.RawMessage(`{"type":"extensionRoute","method":"GET","path":"/topic-actions/bookmark"}`)
				return next
			}(),
		},
		{
			name: "unknown payload type",
			contribution: func() ManifestContribution {
				next := valid
				next.Payload = json.RawMessage(`{"type":"rawHtml","method":"POST","path":"/topic-actions/bookmark"}`)
				return next
			}(),
		},
		{
			name: "unsupported icon prefix",
			contribution: func() ManifestContribution {
				next := valid
				next.Icon = "i-custom-bookmark"
				return next
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := base
			manifest.Contributions = append([]ManifestContribution{tc.contribution}, tc.extra...)
			if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}

// validBaseManifest 构造一个最小合法 manifest，供 C1 Version 校验测试复用。
func validBaseManifest() Manifest {
	return Manifest{
		ID:            "demo.plugin",
		Name:          "Demo",
		Description:   "Demo plugin.",
		URL:           "https://example.com/demo",
		Author:        ManifestAuthor{Name: "Demo"},
		Version:       "1.0.0",
		Type:          "plugin",
		SForumVersion: "^1.0.0",
	}
}

// TestManifestVersionRejectsPathTraversal 验证 C1：危险 Version 值（路径穿越）被拒绝。
func TestManifestVersionRejectsPathTraversal(t *testing.T) {
	dangerous := []string{
		"../../../tmp/evil",
		"..",
		"/absolute/path",
		"with/slash",
		"back\\slash",
	}
	for _, v := range dangerous {
		m := validBaseManifest()
		m.Version = v
		if err := Validate(m); !errors.Is(err, ErrInvalidManifest) {
			t.Fatalf("expected Version %q to be rejected, got %v", v, err)
		}
	}
}

// TestManifestVersionAcceptsValidSemver 验证 C1：合法版本号仍被接受。
func TestManifestVersionAcceptsValidSemver(t *testing.T) {
	valid := []string{"1.0.0", "2.5.1-beta.1", "1.0.0+build.123", "0.0.1", "v1.2.3", "1.0"}
	for _, v := range valid {
		m := validBaseManifest()
		m.Version = v
		if err := Validate(m); err != nil {
			t.Fatalf("expected Version %q to be accepted, got %v", v, err)
		}
	}
}

func TestManifestSettingPresentationMetadata(t *testing.T) {
	manifest := validBaseManifest()
	manifest.Settings = []ManifestSetting{{
		Key:              "encryption",
		Label:            LocalizedText{Default: "Encryption"},
		Description:      LocalizedText{Default: "Choose transport security."},
		Type:             "select",
		Placeholder:      LocalizedText{Default: "Select encryption"},
		RecommendedValue: "starttls",
		Group:            LocalizedText{Default: "server"},
		Options: []ManifestSettingOption{
			{Value: "starttls", Label: LocalizedText{Default: "STARTTLS"}, Description: LocalizedText{Default: "Recommended"}},
			{Value: "tls", Label: LocalizedText{Default: "TLS/SSL"}},
		},
	}}

	if err := Validate(manifest); err != nil {
		t.Fatalf("expected presentation metadata to validate: %v", err)
	}
	normalized := Normalize(manifest).Settings[0]
	if normalized.Placeholder.Resolve("") != "Select encryption" || normalized.RecommendedValue != "starttls" || normalized.Group.Resolve("") != "server" {
		t.Fatalf("unexpected normalized metadata: %#v", normalized)
	}
	if len(normalized.Options) != 2 || normalized.Options[0].Value != "starttls" {
		t.Fatalf("expected ordered options, got %#v", normalized.Options)
	}
}

func TestManifestSettingPresentationMetadataRejectsInvalidOptions(t *testing.T) {
	cases := []struct {
		name        string
		options     []ManifestSettingOption
		recommended string
	}{
		{name: "blank value", options: []ManifestSettingOption{{Value: "", Label: LocalizedText{Default: "Blank"}}}, recommended: ""},
		{name: "blank label", options: []ManifestSettingOption{{Value: "tls", Label: LocalizedText{}}}, recommended: "tls"},
		{name: "duplicate value", options: []ManifestSettingOption{{Value: "tls", Label: LocalizedText{Default: "TLS"}}, {Value: "tls", Label: LocalizedText{Default: "Again"}}}, recommended: "tls"},
		{name: "unknown recommendation", options: []ManifestSettingOption{{Value: "tls", Label: LocalizedText{Default: "TLS"}}}, recommended: "starttls"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validBaseManifest()
			manifest.Settings = []ManifestSetting{{Key: "encryption", Label: LocalizedText{Default: "Encryption"}, Type: "select", Options: tc.options, RecommendedValue: tc.recommended}}
			if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid setting metadata, got %v", err)
			}
		})
	}
}
