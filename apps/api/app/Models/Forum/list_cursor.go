package forum

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 公开列表 keyset 游标（M5）。
// 载荷仅含公开排序键（pin / sort key / id），base64url(JSON)；非加密。
// 篡改最多跳到另一公开位置，不会把隐藏帖塞进列表（SQL 仍过滤 status）。

const listCursorVersion = 1

// topicListCursor 绑定 sort；换 sort 后旧 nextCursor 必须失效。
type topicListCursor struct {
	V    int    `json:"v"`
	Sort string `json:"s"`
	// Pin: 1=置顶，0=非置顶（与 is_pinned DESC 第一维对齐）
	Pin int `json:"p"`
	// Key: latest/active → RFC3339Nano；hot → 十进制 hot_score
	Key string `json:"k"`
	ID  int64  `json:"i"`
}

// commentListCursor flat 列表：path_key ASC, id ASC。
type commentListCursor struct {
	V    int    `json:"v"`
	Path string `json:"pk"`
	ID   int64  `json:"i"`
}

func encodeTopicListCursor(c topicListCursor) (string, error) {
	c.V = listCursorVersion
	c.Sort = strings.TrimSpace(strings.ToLower(c.Sort))
	if c.Sort == "" || c.ID <= 0 || c.Key == "" {
		return "", ErrInvalidCursor
	}
	if c.Pin != 0 && c.Pin != 1 {
		return "", ErrInvalidCursor
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", ErrInvalidCursor
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeTopicListCursor(token string) (topicListCursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return topicListCursor{}, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return topicListCursor{}, ErrInvalidCursor
	}
	var c topicListCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return topicListCursor{}, ErrInvalidCursor
	}
	c.Sort = strings.TrimSpace(strings.ToLower(c.Sort))
	if c.V != listCursorVersion || c.ID <= 0 || c.Key == "" {
		return topicListCursor{}, ErrInvalidCursor
	}
	switch c.Sort {
	case "latest", "active", "hot":
	default:
		return topicListCursor{}, ErrInvalidCursor
	}
	if c.Pin != 0 && c.Pin != 1 {
		return topicListCursor{}, ErrInvalidCursor
	}
	return c, nil
}

func encodeCommentListCursor(c commentListCursor) (string, error) {
	c.V = listCursorVersion
	if c.ID <= 0 || strings.TrimSpace(c.Path) == "" {
		return "", ErrInvalidCursor
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", ErrInvalidCursor
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCommentListCursor(token string) (commentListCursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return commentListCursor{}, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return commentListCursor{}, ErrInvalidCursor
	}
	var c commentListCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return commentListCursor{}, ErrInvalidCursor
	}
	if c.V != listCursorVersion || c.ID <= 0 || strings.TrimSpace(c.Path) == "" {
		return commentListCursor{}, ErrInvalidCursor
	}
	return c, nil
}

func topicCursorFromSummary(sort string, item TopicSummary) (string, error) {
	pin := 0
	if item.IsPinned {
		pin = 1
	}
	key, err := topicSortKeyValue(sort, item)
	if err != nil {
		return "", err
	}
	return encodeTopicListCursor(topicListCursor{
		Sort: sort,
		Pin:  pin,
		Key:  key,
		ID:   item.ID,
	})
}

func topicSortKeyValue(sort string, item TopicSummary) (string, error) {
	switch strings.TrimSpace(strings.ToLower(sort)) {
	case "latest":
		return item.CreatedAt.UTC().Format(time.RFC3339Nano), nil
	case "hot":
		return strconv.FormatInt(item.HotScore, 10), nil
	default: // active
		return item.LastActivityAt.UTC().Format(time.RFC3339Nano), nil
	}
}

// topicKeysetPredicate 生成 is_pinned DESC + sort DESC + id DESC 的「之后」谓词。
// 置顶作为第一维稳定参与 keyset（非「仅首页插 pin」）。
//
// 使用 `col <= $key AND (col < $key OR id < $id)` 让 PG 把 sort 键推进 Index Cond，
// 避免 OR 前缀 Filter 扫掉与 OFFSET 等量的行（depth 2000 时 Rows Removed 爆炸）。
// argStart 起三个占位：pin bool、sortKey、id（与 topicCursorSQLArgs 对齐）。
func topicKeysetPredicate(sort string, argStart int) (string, error) {
	pin := fmt.Sprintf("$%d", argStart)
	key := fmt.Sprintf("$%d", argStart+1)
	id := fmt.Sprintf("$%d", argStart+2)
	var sortCol string
	switch strings.TrimSpace(strings.ToLower(sort)) {
	case "latest":
		sortCol = "topics.created_at"
	case "hot":
		sortCol = "topics.hot_score"
	default:
		sortCol = "topics.last_activity_at"
	}
	// 同桶续页片段（seek 友好）
	sameBucket := fmt.Sprintf(
		`%s <= %s AND (%s < %s OR topics.id < %s)`,
		sortCol, key, sortCol, key, id,
	)
	// pin=true：置顶桶内续页，或整段非置顶；pin=false：仅非置顶桶
	return fmt.Sprintf(`(
		(
			%s::boolean
			AND (
				(topics.is_pinned = true AND %s)
				OR topics.is_pinned = false
			)
		)
		OR
		(
			NOT %s::boolean
			AND topics.is_pinned = false
			AND %s
		)
	)`, pin, sameBucket, pin, sameBucket), nil
}

func topicCursorSQLArgs(c topicListCursor) ([]any, error) {
	pin := c.Pin == 1
	switch c.Sort {
	case "latest", "active":
		ts, err := time.Parse(time.RFC3339Nano, c.Key)
		if err != nil {
			// 兼容 RFC3339（无纳秒）
			ts, err = time.Parse(time.RFC3339, c.Key)
			if err != nil {
				return nil, ErrInvalidCursor
			}
		}
		return []any{pin, ts, c.ID}, nil
	case "hot":
		score, err := strconv.ParseInt(c.Key, 10, 64)
		if err != nil {
			return nil, ErrInvalidCursor
		}
		return []any{pin, score, c.ID}, nil
	default:
		return nil, ErrInvalidCursor
	}
}

func commentCursorFromItem(item Comment) (string, error) {
	return encodeCommentListCursor(commentListCursor{
		Path: item.PathKey,
		ID:   item.ID,
	})
}
