package forumcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	apitokens "github.com/zhuchunshu/sforum/apps/api/app/Models/APITokens"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type forumTestEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type forumTestErrorData struct {
	Reason string `json:"reason"`
}

func TestControllerListsPublicForumData(t *testing.T) {
	app, _, _ := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/categories", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 categories, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var categories forumTestEnvelope[[]forum.Category]
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	if len(categories.Data) != 1 || categories.Data[0].Slug != "general" {
		t.Fatalf("unexpected categories %#v", categories.Data)
	}

	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 topics, got %d", resp.StatusCode)
	}
}

func TestGuestReadLoginRequiredUsesBearerAuthenticationResult(t *testing.T) {
	for _, test := range []struct {
		name       string
		bearer     forumTestBearer
		wantStatus int
	}{
		{
			name: "active PAT allowed",
			bearer: forumTestBearer{auth: apitokens.Authenticated{
				UserID: 1, TokenID: 1, PublicID: "active", Scopes: []string{identity.PermissionPostCreate},
			}},
			wantStatus: nethttp.StatusOK,
		},
		{name: "inactive or deleted PAT rejected", bearer: forumTestBearer{err: apitokens.ErrTokenInvalid}, wantStatus: nethttp.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &controllerForumStore{guestRead: "login_required"}
			controller := NewController(forum.NewServiceWithSettingsAndEvents(store, store, nil), controllerForumActors{}, nil)
			app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
				BearerTokens:   test.bearer,
				RouteProviders: []apphttp.RouteProvider{controller},
			})
			req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/categories", nil)
			req.Header.Set("Authorization", "Bearer sft_forum-test")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != test.wantStatus {
				t.Fatalf("expected %d, got %d", test.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestControllerListsCategoryGroupsAndTags(t *testing.T) {
	app, _, _ := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/category-groups", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 category groups, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var groups forumTestEnvelope[[]forum.CategoryGroup]
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decode category groups: %v", err)
	}
	if len(groups.Data) != 1 || len(groups.Data[0].Categories) != 1 || groups.Data[0].Categories[0].Slug != "general" {
		t.Fatalf("unexpected category groups %#v", groups.Data)
	}

	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/tags", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 tags, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var tags forumTestEnvelope[[]forum.Tag]
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		t.Fatalf("decode tags: %v", err)
	}
	if len(tags.Data) != 1 || tags.Data[0].Slug != "go" || tags.Data[0].Status != forum.TagStatusActive {
		t.Fatalf("unexpected tags %#v", tags.Data)
	}
}

func TestControllerPassesTagSlugToTopicList(t *testing.T) {
	app, _, store := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics?tagSlug=nuxt", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 topics, got %d", resp.StatusCode)
	}
	if store.lastTopicList.TagSlug != "nuxt" {
		t.Fatalf("expected tagSlug nuxt, got %#v", store.lastTopicList)
	}
}

func TestControllerRequiresLoginAndPermissionToCreateTopic(t *testing.T) {
	app, _, store := newForumTestApp()

	body := []byte(`{"categorySlug":"general","title":"新帖子","content":{"rawContent":"正文","sourceFormat":"markdown","editorType":"markdown"}}`)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics", body, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}

	cookie := loginForumUser(t, app, 2)
	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics", body, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without topic.create, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var errorBody forumTestEnvelope[forumTestErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&errorBody); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errorBody.Data.Reason != "permission.denied" {
		t.Fatalf("expected permission.denied, got %q", errorBody.Data.Reason)
	}

	cookie = loginForumUser(t, app, 1)
	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics", body, cookie)
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201 create topic, got %d", resp.StatusCode)
	}
	if store.createdTopic.Title != "新帖子" {
		t.Fatalf("expected store create topic call, got %#v", store.createdTopic)
	}
}

func TestControllerCreateTopicAcceptsTagSlugs(t *testing.T) {
	app, _, store := newForumTestApp()
	cookie := loginForumUser(t, app, 1)

	body := []byte(`{"categorySlug":"general","title":"带标签帖子","tagSlugs":["Go","nuxt"],"content":{"rawContent":"正文","sourceFormat":"markdown","editorType":"markdown"}}`)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics", body, cookie)
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201 create topic, got %d", resp.StatusCode)
	}
	if !stringSlicesEqual(store.createdTopic.TagSlugs, []string{"go", "nuxt"}) {
		t.Fatalf("expected normalized tag slugs on create topic, got %#v", store.createdTopic.TagSlugs)
	}
}

func TestControllerAdminForumPermissions(t *testing.T) {
	app, _, _ := newForumTestApp()
	body := []byte(`{"slug":"support","name":"支持","visibility":"public","position":1}`)

	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/category-groups", body, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", resp.StatusCode)
	}

	cookie := loginForumUser(t, app, 2)
	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/category-groups", body, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without category.manage, got %d", resp.StatusCode)
	}

	cookie = loginForumUser(t, app, 3)
	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/tags", []byte(`{"slug":"go","name":"Go","status":"active"}`), cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without tag.manage, got %d", resp.StatusCode)
	}
}

func TestControllerAdminForumVisualFields(t *testing.T) {
	app, _, _ := newForumTestApp()
	cookie := loginForumUser(t, app, 4)

	categoryBody := []byte(`{"groupId":1,"slug":"support","name":"支持","visibility":"public","defaultSort":"latest","icon":"i-tabler-help","iconColor":"#0f766e"}`)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/categories", categoryBody, cookie)
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201 create category, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var categoryOut forumTestEnvelope[forum.Category]
	if err := json.NewDecoder(resp.Body).Decode(&categoryOut); err != nil {
		t.Fatalf("decode category: %v", err)
	}
	if categoryOut.Data.Icon != "i-tabler-help" || categoryOut.Data.IconColor != "#0f766e" {
		t.Fatalf("expected category visual fields in response, got %#v", categoryOut.Data)
	}

	tagBody := []byte(`{"slug":"go","name":"Go","status":"active","icon":"i-lucide-tag","iconColor":"#2563eb"}`)
	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/tags", tagBody, cookie)
	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201 create tag, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var tagOut forumTestEnvelope[forum.Tag]
	if err := json.NewDecoder(resp.Body).Decode(&tagOut); err != nil {
		t.Fatalf("decode tag: %v", err)
	}
	if tagOut.Data.Icon != "i-lucide-tag" || tagOut.Data.IconColor != "#2563eb" {
		t.Fatalf("expected tag visual fields in response, got %#v", tagOut.Data)
	}
}

func TestControllerAdminSettingsResetWithPermission(t *testing.T) {
	app, _, store := newForumTestApp()
	cookie := loginForumUser(t, app, 4)

	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/settings/reset", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 settings reset, got %d", resp.StatusCode)
	}
	if !store.settingsReset {
		t.Fatal("expected settings reset to reach service manager")
	}
}

func TestControllerAdminSettingsBindsPaginationFields(t *testing.T) {
	app, _, store := newForumTestApp()
	cookie := loginForumUser(t, app, 4)
	resp := performForumRequest(t, app, nethttp.MethodPut, "/api/v1/admin/forum/settings", []byte(`{"topicsPerPage":30,"commentsPerPage":40}`), cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 settings update, got %d", resp.StatusCode)
	}
	if store.updatedSettings.TopicsPerPage == nil || *store.updatedSettings.TopicsPerPage != 30 {
		t.Fatalf("topicsPerPage input = %#v", store.updatedSettings.TopicsPerPage)
	}
	if store.updatedSettings.CommentsPerPage == nil || *store.updatedSettings.CommentsPerPage != 40 {
		t.Fatalf("commentsPerPage input = %#v", store.updatedSettings.CommentsPerPage)
	}
}

func TestControllerUpdateTopicRequiresLoginAndPermission(t *testing.T) {
	app, _, _ := newForumTestApp()
	body := []byte(`{"title":"新标题","expectedRevision":1}`)

	// 未登录 -> 401。
	resp := performForumRequest(t, app, nethttp.MethodPatch, "/api/v1/topics/10", body, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}

	// 登录但无编辑权限 -> 403。
	cookie := loginForumUser(t, app, 2)
	resp = performForumRequest(t, app, nethttp.MethodPatch, "/api/v1/topics/10", body, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without edit permission, got %d", resp.StatusCode)
	}
}

func TestControllerUpdateTopicAllowsModerator(t *testing.T) {
	app, _, store := newForumTestApp()
	cookie := loginForumUser(t, app, 5) // 版主，含 topic.edit_any。

	body := []byte(`{"title":"版主编辑标题","categorySlug":"general","tagSlugs":["go"],"expectedRevision":1,"reason":"moderation correction"}`)
	resp := performForumRequest(t, app, nethttp.MethodPatch, "/api/v1/topics/10", body, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 update topic, got %d", resp.StatusCode)
	}
	if store.updatedTopic.Title != "版主编辑标题" {
		t.Fatalf("expected title update, got %#v", store.updatedTopic)
	}
}

func TestControllerDeleteTopicRequiresPermission(t *testing.T) {
	app, _, store := newForumTestApp()

	// 未登录 -> 401。
	resp := performForumRequest(t, app, nethttp.MethodDelete, "/api/v1/topics/10", nil, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}

	// 无权限 -> 403。
	cookie := loginForumUser(t, app, 2)
	resp = performForumRequest(t, app, nethttp.MethodDelete, "/api/v1/topics/10", nil, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without delete permission, got %d", resp.StatusCode)
	}

	// 版主 -> 200。
	cookie = loginForumUser(t, app, 5)
	resp = performForumRequest(t, app, nethttp.MethodDelete, "/api/v1/topics/10", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 delete topic, got %d", resp.StatusCode)
	}
	if store.deletedTopicID != 10 {
		t.Fatalf("expected deleted topic id 10, got %d", store.deletedTopicID)
	}
}

func TestControllerTopicActionRequiresPermission(t *testing.T) {
	actions := []string{"hide", "restore", "lock", "unlock", "pin", "unpin"}

	for _, action := range actions {
		t.Run(action+"/denied", func(t *testing.T) {
			app, _, _ := newForumTestApp()
			// 无权限用户 -> 403。
			cookie := loginForumUser(t, app, 2)
			resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics/10/"+action, nil, cookie)
			if resp.StatusCode != nethttp.StatusForbidden {
				t.Fatalf("expected 403 for %s without permission, got %d", action, resp.StatusCode)
			}
		})
		t.Run(action+"/unauthenticated", func(t *testing.T) {
			app, _, _ := newForumTestApp()
			resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics/10/"+action, nil, nil)
			if resp.StatusCode != nethttp.StatusUnauthorized {
				t.Fatalf("expected 401 for %s without session, got %d", action, resp.StatusCode)
			}
		})
	}
}

func TestControllerTopicActionAllowsModerator(t *testing.T) {
	app, _, store := newForumTestApp()
	cookie := loginForumUser(t, app, 5)

	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics/10/lock", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 lock, got %d", resp.StatusCode)
	}
	if store.appliedAction != forum.TopicActionLock {
		t.Fatalf("expected applied action lock, got %s", store.appliedAction)
	}

	resp = performForumRequest(t, app, nethttp.MethodPost, "/api/v1/topics/10/pin", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 pin, got %d", resp.StatusCode)
	}
	if store.appliedAction != forum.TopicActionPin {
		t.Fatalf("expected applied action pin, got %s", store.appliedAction)
	}
}

func TestControllerPassesTreeAndFlatCommentViews(t *testing.T) {
	app, _, store := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics/10/comments?view=tree", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 tree comments, got %d", resp.StatusCode)
	}
	if store.lastCommentView != "tree" {
		t.Fatalf("expected tree view, got %q", store.lastCommentView)
	}
	defer resp.Body.Close()
	var treeBody forumTestEnvelope[forum.CommentList]
	if err := json.NewDecoder(resp.Body).Decode(&treeBody); err != nil {
		t.Fatalf("decode tree comments: %v", err)
	}
	if len(treeBody.Data.Items) != 1 || len(treeBody.Data.Items[0].Children) != 1 {
		t.Fatalf("expected nested tree response, got %#v", treeBody.Data.Items)
	}

	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics/10/comments?view=flat", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 flat comments, got %d", resp.StatusCode)
	}
	if store.lastCommentView != "flat" {
		t.Fatalf("expected flat view, got %q", store.lastCommentView)
	}
}

func TestControllerRevisionHistoryPermissionsAndPayloads(t *testing.T) {
	app, _, store := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics/10/revisions", nil, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", resp.StatusCode)
	}

	cookie := loginForumUser(t, app, 2)
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics/10/revisions", nil, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without history permission, got %d", resp.StatusCode)
	}
	if store.topicRevisionCalls != 0 {
		t.Fatal("permission-denied history request should not reach store")
	}

	cookie = loginForumUser(t, app, 6)
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics/10/revisions?perPage=200", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 revision list, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var listBody forumTestEnvelope[forum.RevisionList]
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode revision list: %v", err)
	}
	if listBody.Data.PerPage != 100 || len(listBody.Data.Items) != 1 {
		t.Fatalf("unexpected list response %#v", listBody.Data)
	}
	rawList, err := json.Marshal(listBody.Data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(rawList, []byte("rawContent")) || bytes.Contains(rawList, []byte("legacy source")) {
		t.Fatalf("revision list leaked raw source: %s", rawList)
	}

	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics/10/revisions/1", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 revision detail, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var detailBody forumTestEnvelope[forum.ForumRevisionDetail]
	if err := json.NewDecoder(resp.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode revision detail: %v", err)
	}
	if detailBody.Data.RawContent != "legacy source" || detailBody.Data.Preview == nil || detailBody.Data.Preview.HTMLContent == "" {
		t.Fatalf("expected authorized raw source and preview, got %#v", detailBody.Data)
	}
}

func TestControllerCommentRevisionPermissionAndRedactedErrors(t *testing.T) {
	app, _, store := newForumTestApp()
	cookie := loginForumUser(t, app, 6)

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/comments/20/revisions", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 comment revision list, got %d", resp.StatusCode)
	}

	store.revisionErr = forum.ErrRevisionNotFound
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/comments/20/revisions/999", nil, cookie)
	if resp.StatusCode != nethttp.StatusNotFound {
		t.Fatalf("expected 404 missing revision, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var missing forumTestEnvelope[forumTestErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&missing); err != nil {
		t.Fatalf("decode missing error: %v", err)
	}
	if missing.Data.Reason != forum.CodeRevisionNotFound {
		t.Fatalf("expected %s, got %q", forum.CodeRevisionNotFound, missing.Data.Reason)
	}

	store.revisionErr = forum.ErrRevisionRedacted
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/comments/20/revisions/1", nil, cookie)
	if resp.StatusCode != nethttp.StatusUnprocessableEntity {
		t.Fatalf("expected 422 redacted revision, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var redacted forumTestEnvelope[forumTestErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&redacted); err != nil {
		t.Fatalf("decode redacted error: %v", err)
	}
	if redacted.Data.Reason != forum.CodeRevisionRedacted {
		t.Fatalf("expected %s, got %q", forum.CodeRevisionRedacted, redacted.Data.Reason)
	}
}

func TestControllerAdminContentReadPermissions(t *testing.T) {
	app, _, store := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/topics", nil, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", resp.StatusCode)
	}

	cookie := loginForumUser(t, app, 6)
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/topics", nil, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without admin.access, got %d", resp.StatusCode)
	}
	if store.adminTopicCalls != 0 {
		t.Fatal("admin.access denial should not reach admin topic store")
	}

	cookie = loginForumUser(t, app, 8)
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/comments", nil, cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without edit/history permission, got %d", resp.StatusCode)
	}

	cookie = loginForumUser(t, app, 7)
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/topics?titlePrefix=公", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 admin topic list, got %d", resp.StatusCode)
	}
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/topics/10", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 admin topic detail, got %d", resp.StatusCode)
	}
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/comments", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 admin comment list, got %d", resp.StatusCode)
	}
	resp = performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/content/comments/20", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 admin comment detail, got %d", resp.StatusCode)
	}
	if store.adminTopicCalls == 0 || store.adminCommentCalls == 0 {
		t.Fatalf("expected admin content stores to be called, topics=%d comments=%d", store.adminTopicCalls, store.adminCommentCalls)
	}
}

func newForumTestApp() (*fiber.App, *authsession.Manager, *controllerForumStore) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerForumActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionTopicCreate:    true,
			identity.PermissionTopicEditOwn:   true,
			identity.PermissionTopicDeleteOwn: true,
			identity.PermissionPostCreate:     true,
			identity.PermissionPostEditOwn:    true,
			identity.PermissionPostDeleteOwn:  true,
		}},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
		3: {ID: 3, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionCategoryManage: true,
		}},
		4: {ID: 4, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionCategoryManage:      true,
			identity.PermissionTagManage:           true,
			identity.PermissionForumSettingsManage: true,
		}},
		5: {ID: 5, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionTopicEditAny:   true,
			identity.PermissionTopicDeleteAny: true,
			identity.PermissionTopicLock:      true,
			identity.PermissionTopicPin:       true,
		}},
		6: {ID: 6, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionTopicRevisionViewAny: true,
			identity.PermissionPostRevisionViewAny:  true,
		}},
		7: {ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionAdminAccess:          true,
			identity.PermissionTopicRevisionViewAny: true,
			identity.PermissionPostRevisionViewAny:  true,
		}},
		8: {ID: 8, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionAdminAccess: true,
		}},
	}}
	store := &controllerForumStore{}
	controller := NewController(forum.NewServiceWithSettingsAndEvents(store, store, nil), users, manager)
	loginProvider := forumRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
			if err != nil || userID == 0 {
				userID = 1
			}
			_, err = manager.Start(c, userID)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
	return app, manager, store
}

func loginForumUser(t *testing.T, app *fiber.App, userID int64) *nethttp.Cookie {
	t.Helper()
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/test-login/"+strconv.FormatInt(userID, 10), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected login 200, got %d", resp.StatusCode)
	}
	if len(resp.Cookies()) == 0 {
		t.Fatal("expected login cookie")
	}
	return resp.Cookies()[0]
}

func performForumRequest(t *testing.T, app *fiber.App, method string, path string, body []byte, cookie *nethttp.Cookie) *nethttp.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	return resp
}

type controllerForumActors struct {
	actors map[int64]identity.Actor
}

func (s controllerForumActors) LoadActor(_ context.Context, userID int64) (identity.Actor, error) {
	return s.actors[userID], nil
}

type forumRouteProviderFunc func(api fiber.Router)

func (f forumRouteProviderFunc) RegisterRoutes(api fiber.Router) {
	f(api)
}

type controllerForumStore struct {
	createdTopic         forum.CreateTopicRecord
	updatedTopic         forum.UpdateTopicRecord
	deletedTopicID       int64
	appliedAction        string
	actionTopic          forum.TopicSummary
	lastCommentView      string
	lastTopicList        forum.TopicListInput
	settingsReset        bool
	updatedSettings      forum.UpdateForumSettingsInput
	guestRead            string
	revisionErr          error
	topicRevisionCalls   int
	commentRevisionCalls int
	adminTopicCalls      int
	adminCommentCalls    int
}

func (s *controllerForumStore) ListCategories(context.Context) ([]forum.Category, error) {
	return []forum.Category{{ID: 1, Slug: "general", Name: "综合讨论", Visibility: "public"}}, nil
}

func (s *controllerForumStore) ListCategoryGroups(context.Context) ([]forum.CategoryGroup, error) {
	return []forum.CategoryGroup{{
		ID:         1,
		Slug:       "default",
		Name:       "默认版块",
		Visibility: "public",
		Categories: []forum.Category{{ID: 1, Slug: "general", Name: "综合讨论", Visibility: "public"}},
	}}, nil
}

func (s *controllerForumStore) ListTags(_ context.Context, includePending bool) ([]forum.Tag, error) {
	if includePending {
		return []forum.Tag{
			{ID: 1, Slug: "go", Name: "Go", Status: forum.TagStatusActive},
			{ID: 2, Slug: "pending", Name: "Pending", Status: forum.TagStatusPending},
		}, nil
	}
	return []forum.Tag{{ID: 1, Slug: "go", Name: "Go", Status: forum.TagStatusActive}}, nil
}

func (s *controllerForumStore) CreateCategoryGroup(_ context.Context, input forum.CreateCategoryGroupInput) (forum.CategoryGroup, error) {
	return forum.CategoryGroup{ID: 2, Slug: input.Slug, Name: input.Name, Description: input.Description, Visibility: input.Visibility, Position: input.Position}, nil
}

func (s *controllerForumStore) UpdateCategoryGroup(_ context.Context, input forum.UpdateCategoryGroupInput) (forum.CategoryGroup, error) {
	item := forum.CategoryGroup{ID: input.ID, Slug: "default", Name: "默认版块", Visibility: "public"}
	if input.Slug != nil {
		item.Slug = *input.Slug
	}
	if input.Name != nil {
		item.Name = *input.Name
	}
	return item, nil
}

func (s *controllerForumStore) CreateCategory(_ context.Context, input forum.CreateCategoryInput) (forum.Category, error) {
	return forum.Category{ID: 2, GroupID: input.GroupID, Slug: input.Slug, Name: input.Name, Description: input.Description, Icon: input.Icon, IconColor: input.IconColor, Visibility: input.Visibility, Position: input.Position, DefaultSort: input.DefaultSort}, nil
}

func (s *controllerForumStore) UpdateCategory(_ context.Context, input forum.UpdateCategoryInput) (forum.Category, error) {
	item := forum.Category{ID: input.ID, GroupID: 1, Slug: "general", Name: "综合讨论", Visibility: "public", DefaultSort: "latest"}
	if input.Slug != nil {
		item.Slug = *input.Slug
	}
	if input.Name != nil {
		item.Name = *input.Name
	}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
	return item, nil
}

func (s *controllerForumStore) CreateTag(_ context.Context, input forum.CreateTagInput) (forum.Tag, error) {
	return forum.Tag{ID: 2, Slug: input.Slug, Name: input.Name, Description: input.Description, Icon: input.Icon, IconColor: input.IconColor, Status: input.Status}, nil
}

func (s *controllerForumStore) UpdateTag(_ context.Context, input forum.UpdateTagInput) (forum.Tag, error) {
	item := forum.Tag{ID: input.ID, Slug: "go", Name: "Go", Status: forum.TagStatusActive}
	if input.Slug != nil {
		item.Slug = *input.Slug
	}
	if input.Name != nil {
		item.Name = *input.Name
	}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.IconColor != nil {
		item.IconColor = *input.IconColor
	}
	if input.Status != nil {
		item.Status = *input.Status
	}
	return item, nil
}

func (s *controllerForumStore) ListTopics(_ context.Context, input forum.TopicListInput) (forum.TopicList, error) {
	s.lastTopicList = input
	return forum.TopicList{Items: []forum.TopicSummary{{ID: 10, Title: "公开帖子", Slug: "topic", Status: forum.TopicStatusActive}}, Total: 1, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *controllerForumStore) ListAllTopicIDs(context.Context) ([]int64, error) {
	return []int64{10}, nil
}

func (s *controllerForumStore) GetTopic(context.Context, int64) (forum.TopicDetail, error) {
	return forum.TopicDetail{TopicSummary: forum.TopicSummary{ID: 10, Title: "公开帖子", Slug: "topic", Status: forum.TopicStatusActive}}, nil
}

func (s *controllerForumStore) GetTopicBySlug(context.Context, string) (forum.TopicDetail, error) {
	return forum.TopicDetail{TopicSummary: forum.TopicSummary{ID: 10, Title: "公开帖子", Slug: "topic", Status: forum.TopicStatusActive}}, nil
}

func (s *controllerForumStore) TopicSlugExists(context.Context, string, int64) (bool, error) {
	return false, nil
}

func (s *controllerForumStore) ActiveTopicTitleExists(context.Context, string, int64) (bool, error) {
	return false, nil
}

func (s *controllerForumStore) AutoLockIdleTopics(context.Context, int, int) (int, error) {
	return 0, nil
}

func (s *controllerForumStore) CreateTopic(_ context.Context, input forum.CreateTopicRecord) (forum.TopicDetail, error) {
	s.createdTopic = input
	input.Content.ID = 100
	return forum.TopicDetail{
		TopicSummary: forum.TopicSummary{ID: 10, AuthorUserID: input.AuthorUserID, Title: input.Title, Slug: input.Slug, Status: forum.TopicStatusActive},
		Content:      input.Content,
	}, nil
}

func (s *controllerForumStore) UpdateTopic(_ context.Context, input forum.UpdateTopicRecord) (forum.TopicDetail, error) {
	s.updatedTopic = input
	title := input.Title
	if title == "" {
		title = "公开帖子"
	}
	return forum.TopicDetail{TopicSummary: forum.TopicSummary{ID: input.TopicID, Title: title, Slug: "topic", Status: forum.TopicStatusActive, CurrentRevision: input.ExpectedRevision + 1}, UpdateApplied: true}, nil
}

func (s *controllerForumStore) DeleteTopic(_ context.Context, topicID int64) (forum.TopicDetail, error) {
	s.deletedTopicID = topicID
	return forum.TopicDetail{TopicSummary: forum.TopicSummary{ID: topicID, Status: forum.TopicStatusDeleted}}, nil
}

func (s *controllerForumStore) ApplyTopicAction(_ context.Context, input forum.TopicLifecycleInput) (forum.TopicLifecycleRecord, error) {
	s.appliedAction = input.Action
	return forum.TopicLifecycleRecord{TopicID: input.TopicID, Status: forum.TopicStatusActive, IsPinned: input.Action == forum.TopicActionPin}, nil
}

func (s *controllerForumStore) ResolveTopicTags(_ context.Context, input forum.ResolveTopicTagsInput) ([]forum.TopicTagSummary, error) {
	items := make([]forum.TopicTagSummary, 0, len(input.Slugs))
	for index, slug := range input.Slugs {
		items = append(items, forum.TopicTagSummary{ID: int64(index + 1), Slug: slug, Name: slug, Status: forum.TagStatusActive})
	}
	return items, nil
}

func (s *controllerForumStore) GetTopicForComment(context.Context, int64) (forum.TopicSummary, error) {
	return forum.TopicSummary{ID: 10, Status: forum.TopicStatusActive}, nil
}

func (s *controllerForumStore) GetTopicForAction(context.Context, int64) (forum.TopicSummary, error) {
	if s.actionTopic.ID == 0 {
		return forum.TopicSummary{ID: 10, AuthorUserID: 1, Status: forum.TopicStatusActive, CurrentRevision: 1}, nil
	}
	return s.actionTopic, nil
}

func (s *controllerForumStore) CreateComment(_ context.Context, input forum.CreateCommentRecord) (forum.Comment, error) {
	return forum.Comment{ID: 20, TopicID: input.TopicID, AuthorUserID: input.AuthorUserID, Content: input.Content, Status: forum.CommentStatusActive}, nil
}

func (s *controllerForumStore) GetCommentSummary(context.Context, int64) (forum.CommentSummary, error) {
	return forum.CommentSummary{ID: 20, TopicID: 10, AuthorUserID: 1, Status: forum.CommentStatusActive, CurrentRevision: 1}, nil
}

func (s *controllerForumStore) UpdateComment(_ context.Context, input forum.UpdateCommentRecord) (forum.Comment, error) {
	return forum.Comment{ID: input.CommentID, AuthorUserID: 1, Content: input.Content, Status: forum.CommentStatusActive, CurrentRevision: input.ExpectedRevision + 1, UpdateApplied: true}, nil
}

func (s *controllerForumStore) DeleteComment(context.Context, int64) (forum.Comment, error) {
	return forum.Comment{ID: 20, Status: forum.CommentStatusDeleted}, nil
}

func (s *controllerForumStore) ListComments(_ context.Context, input forum.CommentListInput) (forum.CommentList, error) {
	s.lastCommentView = input.View
	childParent := int64(20)
	root := forum.Comment{ID: 20, TopicID: input.TopicID, Status: forum.CommentStatusActive, Content: forum.RenderedContent{ID: 1, HTMLContent: "<p>root</p>"}}
	child := forum.Comment{ID: 21, TopicID: input.TopicID, ParentID: &childParent, Status: forum.CommentStatusActive, Content: forum.RenderedContent{ID: 2, HTMLContent: "<p>child</p>"}}
	items := []forum.Comment{root, child}
	if input.View == "tree" {
		root.Children = []forum.Comment{child}
		items = []forum.Comment{root}
	}
	return forum.CommentList{Items: items, Total: int64(len(items)), Page: input.Page, PerPage: input.PerPage, View: input.View}, nil
}

func (s *controllerForumStore) ListCommentReplies(context.Context, forum.CommentReplyListInput) ([]forum.Comment, error) {
	return []forum.Comment{{ID: 21, TopicID: 10, Status: forum.CommentStatusActive}}, nil
}

func (s *controllerForumStore) ListTopicRevisions(_ context.Context, _ int64, input forum.RevisionListInput) (forum.RevisionList, error) {
	s.topicRevisionCalls++
	perPage := input.PerPage
	if perPage > 100 {
		perPage = 100
	}
	return forum.RevisionList{
		Items: []forum.ForumRevisionSummary{{
			ID:               101,
			RevisionNo:       1,
			Current:          true,
			Operation:        "migration",
			Origin:           "migration",
			ChangedFields:    []string{"content"},
			CommittedAt:      time.Now(),
			SnapshotComplete: false,
			RestorableFields: []string{"content"},
		}},
		PerPage: perPage,
	}, nil
}

func (s *controllerForumStore) GetTopicRevision(context.Context, int64, int64) (forum.ForumRevisionDetail, error) {
	s.topicRevisionCalls++
	if s.revisionErr != nil {
		return forum.ForumRevisionDetail{}, s.revisionErr
	}
	detail := forum.ForumRevisionDetail{
		ForumRevisionSummary: forum.ForumRevisionSummary{
			ID:               101,
			RevisionNo:       1,
			Current:          true,
			Operation:        "migration",
			Origin:           "migration",
			ChangedFields:    []string{"content"},
			CommittedAt:      time.Now(),
			SnapshotComplete: false,
			RestorableFields: []string{"content"},
		},
		RawContent:    "legacy source",
		SourceFormat:  forum.SourceFormatMarkdown,
		EditorType:    forum.EditorTypeMarkdown,
		EditorVersion: "test",
		RenderVersion: forum.RenderVersion,
		ContentHash:   "hash",
	}
	return detail, nil
}

func (s *controllerForumStore) ListCommentRevisions(_ context.Context, _ int64, input forum.RevisionListInput) (forum.RevisionList, error) {
	s.commentRevisionCalls++
	perPage := input.PerPage
	if perPage > 100 {
		perPage = 100
	}
	return forum.RevisionList{
		Items: []forum.ForumRevisionSummary{{
			ID:               201,
			RevisionNo:       1,
			Current:          true,
			Operation:        "create",
			Origin:           "self",
			ChangedFields:    []string{"content"},
			CommittedAt:      time.Now(),
			SnapshotComplete: true,
			RestorableFields: []string{"attachments", "content"},
		}},
		PerPage: perPage,
	}, nil
}

func (s *controllerForumStore) GetCommentRevision(context.Context, int64, int64) (forum.ForumRevisionDetail, error) {
	s.commentRevisionCalls++
	if s.revisionErr != nil {
		return forum.ForumRevisionDetail{}, s.revisionErr
	}
	return forum.ForumRevisionDetail{
		ForumRevisionSummary: forum.ForumRevisionSummary{
			ID:               201,
			RevisionNo:       1,
			Current:          true,
			Operation:        "create",
			Origin:           "self",
			ChangedFields:    []string{"content"},
			CommittedAt:      time.Now(),
			SnapshotComplete: true,
			RestorableFields: []string{"attachments", "content"},
		},
		RawContent:    "comment source",
		SourceFormat:  forum.SourceFormatMarkdown,
		EditorType:    forum.EditorTypeMarkdown,
		EditorVersion: "test",
		RenderVersion: forum.RenderVersion,
		ContentHash:   "hash",
	}, nil
}

func (s *controllerForumStore) ListAdminForumTopics(context.Context, forum.AdminForumContentListInput) (forum.AdminForumContentList, error) {
	s.adminTopicCalls++
	return forum.AdminForumContentList{Items: []forum.AdminForumContentRow{{
		TargetType:      "topic",
		ID:              10,
		TopicID:         10,
		TopicTitle:      "公开帖子",
		CategorySlug:    "general",
		AuthorUserID:    1,
		Status:          forum.TopicStatusActive,
		Title:           "公开帖子",
		Excerpt:         "摘要",
		CurrentRevision: 1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}}, PerPage: 20}, nil
}

func (s *controllerForumStore) GetAdminForumTopic(context.Context, int64) (forum.AdminForumTopicDetail, error) {
	s.adminTopicCalls++
	return forum.AdminForumTopicDetail{
		AdminForumContentRow: forum.AdminForumContentRow{
			TargetType:      "topic",
			ID:              10,
			TopicID:         10,
			TopicTitle:      "公开帖子",
			CategorySlug:    "general",
			AuthorUserID:    1,
			Status:          forum.TopicStatusActive,
			Title:           "公开帖子",
			Excerpt:         "摘要",
			CurrentRevision: 1,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		Content: forum.RenderedContent{RawContent: "body", HTMLContent: "<p>body</p>", PlainText: "body", SourceFormat: forum.SourceFormatMarkdown},
		Slug:    "topic",
	}, nil
}

func (s *controllerForumStore) ListAdminForumComments(context.Context, forum.AdminForumContentListInput) (forum.AdminForumContentList, error) {
	s.adminCommentCalls++
	return forum.AdminForumContentList{Items: []forum.AdminForumContentRow{{
		TargetType:      "comment",
		ID:              20,
		TopicID:         10,
		TopicTitle:      "公开帖子",
		CategorySlug:    "general",
		AuthorUserID:    1,
		Status:          forum.CommentStatusActive,
		Excerpt:         "评论摘要",
		CurrentRevision: 1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}}, PerPage: 20}, nil
}

func (s *controllerForumStore) GetAdminForumComment(context.Context, int64) (forum.AdminForumCommentDetail, error) {
	s.adminCommentCalls++
	return forum.AdminForumCommentDetail{
		AdminForumContentRow: forum.AdminForumContentRow{
			TargetType:      "comment",
			ID:              20,
			TopicID:         10,
			TopicTitle:      "公开帖子",
			CategorySlug:    "general",
			AuthorUserID:    1,
			Status:          forum.CommentStatusActive,
			Excerpt:         "评论摘要",
			CurrentRevision: 1,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		Content:       forum.RenderedContent{RawContent: "comment", HTMLContent: "<p>comment</p>", PlainText: "comment", SourceFormat: forum.SourceFormatMarkdown},
		RootCommentID: 20,
		PathKey:       "000000000020",
	}, nil
}

func (s *controllerForumStore) LatestAuthorTopicCreatedAt(context.Context, int64) (time.Time, bool, error) {
	return time.Time{}, false, nil
}
func (s *controllerForumStore) CountAuthorTopicsSince(context.Context, int64, time.Time) (int64, error) {
	return 0, nil
}
func (s *controllerForumStore) LatestAuthorCommentCreatedAt(context.Context, int64) (time.Time, bool, error) {
	return time.Time{}, false, nil
}
func (s *controllerForumStore) CountAuthorCommentsSince(context.Context, int64, time.Time) (int64, error) {
	return 0, nil
}

func (s *controllerForumStore) ForumSettings(context.Context) (forum.ForumSettings, error) {
	return forum.ForumSettings{
		DefaultCategorySlug:    "general",
		TagCreationMode:        forum.TagCreationModeControlled,
		TagPublicPages:         true,
		TagMinPerTopic:         0,
		TagMaxPerTopic:         5,
		TopicsPerPage:          20,
		CommentsPerPage:        20,
		TopicTitleMinRunes:     2,
		TopicTitleMaxRunes:     100,
		TopicContentMinRunes:   0,
		TopicContentMaxRunes:   50000,
		CommentMinRunes:        1,
		CommentMaxRunes:        10000,
		CommentMaxNestingDepth: 5,
		TreeDescendantsPerRoot: 50,
		ExcerptRuneLimit:       180,
		GuestRead:              s.guestRead,
	}, nil
}

type forumTestBearer struct {
	auth apitokens.Authenticated
	err  error
}

func (b forumTestBearer) AuthenticatePlaintext(fiber.Ctx, string) (apitokens.Authenticated, error) {
	return b.auth, b.err
}

func (s *controllerForumStore) UpdateForumSettings(_ context.Context, _ identity.Actor, input forum.UpdateForumSettingsInput) (forum.ForumSettings, error) {
	s.updatedSettings = input
	settings, _ := s.ForumSettings(context.Background())
	if input.DefaultCategorySlug != nil {
		settings.DefaultCategorySlug = *input.DefaultCategorySlug
	}
	if input.TagCreationMode != nil {
		settings.TagCreationMode = *input.TagCreationMode
	}
	if input.TagPublicPages != nil {
		settings.TagPublicPages = *input.TagPublicPages
	}
	if input.TagMaxPerTopic != nil {
		settings.TagMaxPerTopic = *input.TagMaxPerTopic
	}
	if input.TopicsPerPage != nil {
		settings.TopicsPerPage = *input.TopicsPerPage
	}
	if input.CommentsPerPage != nil {
		settings.CommentsPerPage = *input.CommentsPerPage
	}
	return settings, nil
}

func (s *controllerForumStore) ResetForumSettings(context.Context, identity.Actor) (forum.ForumSettings, error) {
	s.settingsReset = true
	return s.ForumSettings(context.Background())
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

var _ = time.Time{}

// --- 搜索端点测试 ---

// fakeSearchService 记录搜索请求并返回预设结果。
type fakeSearchService struct {
	lastInput SearchInput
	result    SearchOutput
	err       error
}

func (f *fakeSearchService) Search(_ context.Context, input SearchInput) (SearchOutput, error) {
	f.lastInput = input
	return f.result, f.err
}

// newForumTestAppWithSearch 构造带 search service 的测试 app。
func newForumTestAppWithSearch(searchSvc SearchService) (*fiber.App, *controllerForumStore) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerForumActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	store := &controllerForumStore{}
	controller := NewControllerWithSearch(forum.NewServiceWithSettingsAndEvents(store, store, nil), searchSvc, nil, users, manager)
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller},
	})
	return app, store
}

func TestControllerSearchEndpoint(t *testing.T) {
	svc := &fakeSearchService{result: SearchOutput{
		Items: []forum.TopicSummary{{
			ID: 1, Title: "Go 指南", Slug: "go-guide", Status: "active",
			CategoryID: 1, CategorySlug: "general", CategoryName: "综合讨论",
			AuthorUserID: 1, Excerpt: "hello",
		}},
		Total:   1,
		Page:    1,
		PerPage: 20,
	}}
	app, _ := newForumTestAppWithSearch(svc)

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/search?query=go", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 search, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var out forumTestEnvelope[SearchOutput]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if len(out.Data.Items) != 1 || out.Data.Items[0].Title != "Go 指南" {
		t.Fatalf("unexpected search items %#v", out.Data.Items)
	}
	if svc.lastInput.Query != "go" {
		t.Fatalf("expected query 'go', got %q", svc.lastInput.Query)
	}
}

func TestControllerSearchWithoutServiceReturns503(t *testing.T) {
	// 无 search service 时应返回 503。
	app, _ := newForumTestAppWithSearch(nil)

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/search?query=go", nil, nil)
	if resp.StatusCode != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected 503 when search unavailable, got %d", resp.StatusCode)
	}
}

func TestControllerTopicsRejectsQuery(t *testing.T) {
	// topics 列表带 query 应返回 400，引导走 /search。
	app, _, _ := newForumTestApp()

	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/topics?query=keyword", nil, nil)
	if resp.StatusCode != nethttp.StatusBadRequest {
		t.Fatalf("expected 400 for topics with query, got %d", resp.StatusCode)
	}
}

// --- 搜索重建端点测试 ---

type fakeReindexService struct {
	reindexRun    ReindexRunOutput
	reindexErr    error
	statusOutput  ReindexStatusOutput
	statusErr     error
	runsOutput    []ReindexRunOutput
	reindexCalled bool
}

func (f *fakeReindexService) Reindex(_ context.Context, _ int64) (ReindexRunOutput, error) {
	f.reindexCalled = true
	return f.reindexRun, f.reindexErr
}

func (f *fakeReindexService) ReindexStatus(_ context.Context) (ReindexStatusOutput, error) {
	return f.statusOutput, f.statusErr
}

func (f *fakeReindexService) ListReindexRuns(_ context.Context) ([]ReindexRunOutput, error) {
	return f.runsOutput, nil
}

// newForumTestAppWithReindex 构造带 reindex service 的测试 app。
// actor 6 拥有 search.manage 权限。
func newForumTestAppWithReindex(reindexer ReindexService) *fiber.App {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerForumActors{actors: map[int64]identity.Actor{
		6: {ID: 6, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionSearchManage: true}},
		7: {ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	store := &controllerForumStore{}
	controller := NewControllerWithSearch(forum.NewServiceWithSettingsAndEvents(store, store, nil), nil, reindexer, users, manager)
	loginProvider := forumRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			userID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
			if userID == 0 {
				userID = 6
			}
			_, err := manager.Start(c, userID)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
	return app
}

func TestControllerReindexRequiresLogin(t *testing.T) {
	app := newForumTestAppWithReindex(&fakeReindexService{})

	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/search/reindex", []byte(`{}`), nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", resp.StatusCode)
	}
}

func TestControllerReindexRequiresPermission(t *testing.T) {
	app := newForumTestAppWithReindex(&fakeReindexService{})

	cookie := loginForumUser(t, app, 7) // 无 search.manage
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/search/reindex", []byte(`{}`), cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403 without permission, got %d", resp.StatusCode)
	}
}

func TestControllerReindexTriggersRebuild(t *testing.T) {
	svc := &fakeReindexService{reindexRun: ReindexRunOutput{ID: 1, Total: 5, Status: "running"}}
	app := newForumTestAppWithReindex(svc)

	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/search/reindex", []byte(`{}`), cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !svc.reindexCalled {
		t.Fatal("expected Reindex to be called")
	}
	defer resp.Body.Close()
	var out forumTestEnvelope[ReindexRunOutput]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Data.ID != 1 || out.Data.Total != 5 {
		t.Fatalf("unexpected run %#v", out.Data)
	}
}

func TestControllerReindexStatusReturnsProgress(t *testing.T) {
	svc := &fakeReindexService{statusOutput: ReindexStatusOutput{
		ReindexRunOutput: ReindexRunOutput{ID: 2, Total: 10, Status: "running"},
		Processed:        7, Remaining: 3, Percent: 70,
	}}
	app := newForumTestAppWithReindex(svc)

	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/search/reindex", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var out forumTestEnvelope[ReindexStatusOutput]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Data.Percent != 70 || out.Data.Processed != 7 {
		t.Fatalf("unexpected status %#v", out.Data)
	}
}

func TestControllerReindexRunsReturnsHistory(t *testing.T) {
	svc := &fakeReindexService{runsOutput: []ReindexRunOutput{{ID: 1, Total: 5, Status: "completed"}}}
	app := newForumTestAppWithReindex(svc)

	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/search/reindex/runs", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var out forumTestEnvelope[[]ReindexRunOutput]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Status != "completed" {
		t.Fatalf("unexpected runs %#v", out.Data)
	}
}

func TestControllerReindexWithoutServiceReturns503(t *testing.T) {
	// reindexer 为 nil 时应返回 503。
	app := newForumTestAppWithReindex(nil)

	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/search/reindex", []byte(`{}`), cookie)
	if resp.StatusCode != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected 503 when reindexer nil, got %d", resp.StatusCode)
	}
}

type fakeSearchProviderAdmin struct {
	state      SearchProvidersState
	selectID   string
	selectErr  error
	restoreErr error
	listErr    error
}

func (f *fakeSearchProviderAdmin) List(context.Context) (SearchProvidersState, error) {
	if f.listErr != nil {
		return SearchProvidersState{}, f.listErr
	}
	return f.state, nil
}

func (f *fakeSearchProviderAdmin) Select(_ context.Context, extensionID string) error {
	f.selectID = extensionID
	return f.selectErr
}

func (f *fakeSearchProviderAdmin) RestoreDefault(context.Context) error {
	return f.restoreErr
}

func newForumTestAppWithSearchProviders(admin SearchProviderAdmin) *fiber.App {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerForumActors{actors: map[int64]identity.Actor{
		6: {ID: 6, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionSearchManage: true}},
		7: {ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	store := &controllerForumStore{}
	controller := NewControllerWithSearch(forum.NewServiceWithSettingsAndEvents(store, store, nil), nil, nil, users, manager).
		WithSearchProviderAdmin(admin)
	loginProvider := forumRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			userID, _ := strconv.ParseInt(c.Params("id"), 10, 64)
			if userID == 0 {
				userID = 6
			}
			_, err := manager.Start(c, userID)
			return err
		})
	})
	return apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
}

func TestControllerListSearchProviders(t *testing.T) {
	admin := &fakeSearchProviderAdmin{state: SearchProvidersState{
		Items: []SearchProviderItem{
			{ExtensionID: "sforum.search-site", Label: "Site Search", Healthy: true, IsDefault: true},
		},
		Selected:           SearchProviderItem{ExtensionID: "sforum.search-site", Label: "Site Search", Healthy: true, IsDefault: true},
		Pinned:             false,
		DefaultExtensionID: "sforum.search-site",
	}}
	app := newForumTestAppWithSearchProviders(admin)
	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodGet, "/api/v1/admin/forum/search/providers", nil, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var out forumTestEnvelope[SearchProvidersState]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Data.DefaultExtensionID != "sforum.search-site" || len(out.Data.Items) != 1 {
		t.Fatalf("unexpected state %#v", out.Data)
	}
}

func TestControllerSelectSearchProvider(t *testing.T) {
	admin := &fakeSearchProviderAdmin{}
	app := newForumTestAppWithSearchProviders(admin)
	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodPut, "/api/v1/admin/forum/search/provider", []byte(`{"extensionId":"sforum.search-meilisearch"}`), cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if admin.selectID != "sforum.search-meilisearch" {
		t.Fatalf("selectID=%q", admin.selectID)
	}
}

func TestControllerSelectSearchProviderRequiresPermission(t *testing.T) {
	app := newForumTestAppWithSearchProviders(&fakeSearchProviderAdmin{})
	cookie := loginForumUser(t, app, 7)
	resp := performForumRequest(t, app, nethttp.MethodPut, "/api/v1/admin/forum/search/provider", []byte(`{"extensionId":"x"}`), cookie)
	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestControllerResetSearchProvider(t *testing.T) {
	admin := &fakeSearchProviderAdmin{}
	app := newForumTestAppWithSearchProviders(admin)
	cookie := loginForumUser(t, app, 6)
	resp := performForumRequest(t, app, nethttp.MethodPost, "/api/v1/admin/forum/search/provider/reset", []byte(`{}`), cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
