package main

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

const (
	publicQueryID  = "sforum.query-reference.items"
	privateQueryID = "sforum.query-reference.private"
	filterID       = "sforum.query-reference.items.mask"
	handler        = "sforum.query-reference.items"
	filter         = "sforum.query-reference.items.mask"
)

func main() {
	pluginv2.Serve(pluginv2.NewServer().
		WithFeatures(pluginv2.QueryRuntimeProtocolFeature()).
		WithQueryRuntimeHandlers(pluginv2.QueryRuntimeHandlers{
			InvokeQuery:       invokeReferenceItems,
			FilterQueryResult: filterReferenceItems,
		}),
	)
}

func invokeReferenceItems(ctx context.Context, call *pluginv2.QueryRuntimeCall) ([]json.RawMessage, error) {
	if call == nil || call.Binding == nil || call.Plan == nil {
		return nil, errors.New("missing query call")
	}
	if call.Context.GetActor() != nil || len(call.Context.GetGrantedAuthority()) != 0 {
		return nil, errors.New("authority projection leaked into query runtime")
	}
	// public/private 共用同一 handler 符号；Host 按 queryId 区分权限与分页语义。
	queryID := call.Binding.GetQueryId()
	if call.Binding.GetHandler() != handler || (queryID != publicQueryID && queryID != privateQueryID) {
		return nil, errors.New("query binding drifted")
	}
	// 测试触发：filter state=fail → 插件失败；state=bad-schema → 返回未声明字段。
	for _, item := range call.Plan.GetFilters() {
		switch item.GetField() {
		case "state":
			switch item.GetValue() {
			case "fail":
				return nil, errors.New("reference query failure")
			case "timeout":
				<-ctx.Done()
				return nil, context.Cause(ctx)
			case "bad-schema":
				return []json.RawMessage{json.RawMessage(`{"id":"1","title":"leak","secret":"no"}`)}, nil
			}
		}
	}
	offset := int(call.Plan.GetPagination().GetOffset())
	limit := int(call.Plan.GetFetchLimit())
	if limit < 1 {
		limit = 1
	}
	rows := make([]json.RawMessage, 0, limit)
	for index := 0; index < limit; index++ {
		id := offset + index + 1
		title := "item-" + strconv.Itoa(id)
		// 保留大整数词素，证明 Host 解码路径无损。
		body, err := json.Marshal(map[string]any{
			"id":    strconv.Itoa(id),
			"title": title,
			"score": json.Number("9007199254740993"),
		})
		if err != nil {
			return nil, err
		}
		rows = append(rows, body)
	}
	return rows, nil
}

func filterReferenceItems(_ context.Context, call *pluginv2.QueryResultFilterRuntimeCall) ([]json.RawMessage, error) {
	if call == nil || call.Binding == nil {
		return nil, errors.New("missing filter call")
	}
	if call.Binding.GetHandler() != filter || call.Binding.GetFilterId() != filterID {
		return nil, errors.New("filter binding drifted")
	}
	result := make([]json.RawMessage, 0, len(call.Rows))
	for _, row := range call.Rows {
		decoded, err := pluginv2.DecodeQueryRuntimeRow(row)
		if err != nil {
			return nil, err
		}
		title, _ := decoded["title"].(string)
		decoded["title"] = strings.TrimSpace(title) + " | masked"
		// 禁止新增 undeclared 字段；仅改写已选 title。
		body, err := json.Marshal(decoded)
		if err != nil {
			return nil, err
		}
		result = append(result, body)
	}
	return result, nil
}
