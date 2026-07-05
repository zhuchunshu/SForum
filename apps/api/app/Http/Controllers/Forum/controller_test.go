package forumcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
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

func newForumTestApp() (*fiber.App, *authsession.Manager, *controllerForumStore) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerForumActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, Permissions: map[string]bool{
			identity.PermissionTopicCreate:   true,
			identity.PermissionPostCreate:    true,
			identity.PermissionPostEditOwn:   true,
			identity.PermissionPostDeleteOwn: true,
		}},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	store := &controllerForumStore{}
	controller := NewController(forum.NewService(store), users, manager)
	loginProvider := forumRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			userID := int64(1)
			if c.Params("id") == "2" {
				userID = 2
			}
			_, err := manager.Start(c, userID)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
	return app, manager, store
}

func loginForumUser(t *testing.T, app *fiber.App, userID int64) *nethttp.Cookie {
	t.Helper()
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/test-login/1", nil)
	if userID == 2 {
		req = httptest.NewRequest(nethttp.MethodPost, "/api/v1/test-login/2", nil)
	}
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
	createdTopic    forum.CreateTopicRecord
	lastCommentView string
}

func (s *controllerForumStore) ListCategories(context.Context) ([]forum.Category, error) {
	return []forum.Category{{ID: 1, Slug: "general", Name: "综合讨论", Visibility: "public"}}, nil
}

func (s *controllerForumStore) ListTopics(_ context.Context, input forum.TopicListInput) (forum.TopicList, error) {
	return forum.TopicList{Items: []forum.TopicSummary{{ID: 10, Title: "公开帖子", Slug: "topic", Status: forum.TopicStatusActive}}, Total: 1, Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *controllerForumStore) GetTopic(context.Context, int64) (forum.TopicDetail, error) {
	return forum.TopicDetail{TopicSummary: forum.TopicSummary{ID: 10, Title: "公开帖子", Slug: "topic", Status: forum.TopicStatusActive}}, nil
}

func (s *controllerForumStore) CreateTopic(_ context.Context, input forum.CreateTopicRecord) (forum.TopicDetail, error) {
	s.createdTopic = input
	input.Content.ID = 100
	return forum.TopicDetail{
		TopicSummary: forum.TopicSummary{ID: 10, AuthorUserID: input.AuthorUserID, Title: input.Title, Slug: input.Slug, Status: forum.TopicStatusActive},
		Content:      input.Content,
	}, nil
}

func (s *controllerForumStore) GetTopicForComment(context.Context, int64) (forum.TopicSummary, error) {
	return forum.TopicSummary{ID: 10, Status: forum.TopicStatusActive}, nil
}

func (s *controllerForumStore) CreateComment(_ context.Context, input forum.CreateCommentRecord) (forum.Comment, error) {
	return forum.Comment{ID: 20, TopicID: input.TopicID, AuthorUserID: input.AuthorUserID, Content: input.Content, Status: forum.CommentStatusActive}, nil
}

func (s *controllerForumStore) GetCommentSummary(context.Context, int64) (forum.CommentSummary, error) {
	return forum.CommentSummary{ID: 20, TopicID: 10, AuthorUserID: 1, Status: forum.CommentStatusActive}, nil
}

func (s *controllerForumStore) UpdateComment(_ context.Context, input forum.UpdateCommentRecord) (forum.Comment, error) {
	return forum.Comment{ID: input.CommentID, AuthorUserID: 1, Content: input.Content, Status: forum.CommentStatusActive}, nil
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

func (s *controllerForumStore) ListCommentReplies(context.Context, int64) ([]forum.Comment, error) {
	return []forum.Comment{{ID: 21, TopicID: 10, Status: forum.CommentStatusActive}}, nil
}

var _ = time.Time{}
