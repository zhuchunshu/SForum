package extensionmanifest

import "testing"

func TestContributionEnabledBySetting(t *testing.T) {
	manifest := Manifest{
		Settings: []ManifestSetting{{
			Key: "show_topic_badge", Type: "boolean", Default: "false",
		}},
	}

	if !ContributionEnabledBySetting(manifest, nil, "") {
		t.Fatal("empty gate must always enable")
	}
	// 默认 false：无已存值时不展示。
	if ContributionEnabledBySetting(manifest, nil, "show_topic_badge") {
		t.Fatal("default false should hide contribution")
	}
	if ContributionEnabledBySetting(manifest, map[string]string{}, "show_topic_badge") {
		t.Fatal("empty store should fall back to default false")
	}
	if !ContributionEnabledBySetting(manifest, map[string]string{"show_topic_badge": "true"}, "show_topic_badge") {
		t.Fatal("stored true should show contribution")
	}
	if ContributionEnabledBySetting(manifest, map[string]string{"show_topic_badge": "false"}, "show_topic_badge") {
		t.Fatal("stored false should hide contribution")
	}
	// 已存空串回落到 default。
	if ContributionEnabledBySetting(manifest, map[string]string{"show_topic_badge": "  "}, "show_topic_badge") {
		t.Fatal("blank stored value should fall back to default false")
	}

	manifest.Settings[0].Default = "true"
	if !ContributionEnabledBySetting(manifest, nil, "show_topic_badge") {
		t.Fatal("default true should show contribution")
	}
}
