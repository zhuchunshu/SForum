package main

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

// P13 custom-content reference: Entity/Content/Editor/Navigation + Query surface.
// Content handlers are Host-published declarations; render/sanitize remain Host-final.

const (
	articlesQueryID = "sforum.custom-content.articles"
	articlesHandler = "sforum.custom-content.articles"
	totalArticles   = 3
)

func main() {
	// Query handler 证明 Entity 声明可挂 Host Query Registry；其余表面由 Manifest 发布。
	pluginv2.Serve(pluginv2.NewServer().
		WithFeatures(pluginv2.QueryRuntimeProtocolFeature()).
		WithQueryRuntimeHandlers(pluginv2.QueryRuntimeHandlers{
			InvokeQuery: invokeReferenceArticles,
		}),
	)
}

func invokeReferenceArticles(_ context.Context, call *pluginv2.QueryRuntimeCall) ([]json.RawMessage, error) {
	if call == nil || call.Binding == nil || call.Plan == nil {
		return nil, errors.New("missing query call")
	}
	if call.Binding.GetHandler() != articlesHandler || call.Binding.GetQueryId() != articlesQueryID {
		return nil, errors.New("query binding drifted")
	}
	for _, item := range call.Plan.GetFilters() {
		if item.GetField() == "state" && item.GetValue() == "fail" {
			return nil, errors.New("reference custom-content query failure")
		}
	}
	offset := int(call.Plan.GetPagination().GetOffset())
	limit := int(call.Plan.GetFetchLimit())
	if limit < 1 {
		limit = 1
	}
	rows := make([]json.RawMessage, 0, limit)
	for index := 0; index < limit && offset+index < totalArticles; index++ {
		id := offset + index + 1
		body, err := json.Marshal(map[string]any{
			"id":      strconv.Itoa(id),
			"title":   "article-" + strconv.Itoa(id),
			"summary": "summary-" + strconv.Itoa(id),
			"slug":    "article-" + strconv.Itoa(id),
		})
		if err != nil {
			return nil, err
		}
		rows = append(rows, body)
	}
	return rows, nil
}
