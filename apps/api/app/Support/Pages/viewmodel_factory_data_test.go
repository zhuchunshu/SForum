package pages

import (
	"testing"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

func TestBuildCorePageViewModelCopiesTypedProductData(t *testing.T) {
	homeData := &themecompiler.HomePageViewModel{Topics: []themecompiler.TopicSummaryView{{ID: 42, Title: "Production topic"}}}
	model, err := BuildCorePageViewModel(CorePageViewModelRequest{
		PageID: "forum.home", Locale: "en-US", Path: "/", Data: CorePageViewModelData{Home: homeData},
	})
	if err != nil {
		t.Fatal(err)
	}
	home, ok := model.(themecompiler.HomePageViewModel)
	if !ok || len(home.Topics) != 1 || home.Topics[0].ID != 42 {
		t.Fatalf("typed product data lost: %#v", model)
	}
	if home.Base.PageID != "forum.home" || home.Base.SchemaVersion != "sforum.page.home@1" {
		t.Fatalf("Host base was not authoritative: %#v", home.Base)
	}
	if homeData.Base.PageID != "" {
		t.Fatal("factory mutated the source-owned payload")
	}
}

func TestBuildCorePageViewModelOverridesHostFormBoundary(t *testing.T) {
	input := &themecompiler.TopicCreatePageViewModel{Form: themecompiler.HostFormBoundary{
		ComponentID: "plugin.fake.form", ActionRouteIDs: []string{"plugin.fake.route"},
	}}
	model, err := BuildCorePageViewModel(CorePageViewModelRequest{
		PageID: "forum.topic.create", Locale: "en-US", Path: "/topics/new",
		Data: CorePageViewModelData{TopicCreate: input},
	})
	if err != nil {
		t.Fatal(err)
	}
	created := model.(themecompiler.TopicCreatePageViewModel)
	if created.Form.ComponentID != "forum.component.topic_composer" || len(created.Form.ActionRouteIDs) != 1 || created.Form.ActionRouteIDs[0] != "core.route.forum.create_topic" {
		t.Fatalf("untrusted form boundary survived: %#v", created.Form)
	}
}
