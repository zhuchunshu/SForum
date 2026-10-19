package editordocument

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

const (
	// maxTableRows/maxTableColumns 是存储层的硬上限，超出即截断，
	// 避免超大表格拖垮渲染与 Markdown 投影。
	maxTableRows    = 200
	maxTableColumns = 50
	maxTableSpan    = 20
)

var tableCellAlignments = map[string]struct{}{
	"left":   {},
	"center": {},
	"right":  {},
}

// normalizeTableStructure 在 normalizeNodes 已产出子节点之后修复表格骨架：
// table 只保留 tableRow，限制行数，空表降级为段落文本 [table]。
func normalizeTableStructure(node Node, fallbacks *[]string) Node {
	rows := make([]Node, 0, len(node.Content))
	for _, child := range node.Content {
		if child.Type != "tableRow" {
			*fallbacks = append(*fallbacks, "table")
			continue
		}
		if len(child.Content) == 0 {
			continue
		}
		rows = append(rows, child)
	}
	if len(rows) > maxTableRows {
		rows = rows[:maxTableRows]
		*fallbacks = append(*fallbacks, "table")
	}
	if len(rows) == 0 {
		*fallbacks = append(*fallbacks, "table")
		return Node{Type: "paragraph", Content: []Node{{Type: "text", Text: "[table]"}}}
	}
	node.Content = rows
	return node
}

// normalizeTableRow 只保留 tableHeader/tableCell，并按有效列数（含 colspan）
// 截断超宽行。
func normalizeTableRow(node Node, fallbacks *[]string) Node {
	cells := make([]Node, 0, len(node.Content))
	columns := 0
	truncated := false
	for _, child := range node.Content {
		if child.Type != "tableHeader" && child.Type != "tableCell" {
			*fallbacks = append(*fallbacks, "tableRow")
			continue
		}
		span := 1
		if value, ok := normalizedTableSpan(child.Attrs["colspan"]); ok {
			span = value
		}
		if truncated || columns+span > maxTableColumns {
			truncated = true
			continue
		}
		cells = append(cells, child)
		columns += span
	}
	if truncated {
		*fallbacks = append(*fallbacks, "tableRow")
	}
	node.Content = cells
	return node
}

// normalizeTableCell 把单元格内容收敛为块节点序列：inline 子节点包进段落，
// 嵌套表格展平为纯文本段落，空单元格补空段落。
func normalizeTableCell(node Node, fallbacks *[]string) Node {
	node.Attrs = normalizeTableCellAttrs(node.Attrs)
	content := make([]Node, 0, len(node.Content))
	inline := make([]Node, 0, len(node.Content))
	flushInline := func() {
		if len(inline) == 0 {
			return
		}
		content = append(content, Node{Type: "paragraph", Content: inline})
		inline = nil
	}
	for _, child := range node.Content {
		switch child.Type {
		case "text", "image", "sforumEmoji", "hardBreak":
			inline = append(inline, child)
		case "table":
			flushInline()
			content = append(content, flattenNestedTable(child))
			*fallbacks = append(*fallbacks, "table")
		default:
			flushInline()
			content = append(content, child)
		}
	}
	flushInline()
	if len(content) == 0 {
		content = append(content, Node{Type: "paragraph"})
	}
	node.Content = content
	return node
}

// normalizeTableCellAttrs 夹紧 colspan/rowspan 并只保留合法 align。
func normalizeTableCellAttrs(attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	if value, ok := normalizedTableSpan(attrs["colspan"]); ok {
		attrs["colspan"] = value
	} else {
		delete(attrs, "colspan")
	}
	if value, ok := normalizedTableSpan(attrs["rowspan"]); ok {
		attrs["rowspan"] = value
	} else {
		delete(attrs, "rowspan")
	}
	if align, ok := attrs["align"].(string); ok {
		if _, valid := tableCellAlignments[align]; valid {
			attrs["align"] = align
		} else {
			delete(attrs, "align")
		}
	} else {
		delete(attrs, "align")
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

// normalizedTableSpan 取整并夹到 [1,20]；非数值属性视为无效。
func normalizedTableSpan(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return clampTableSpan(typed), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return 0, false
		}
		if typed <= 0 {
			return 1, true
		}
		if typed >= maxTableSpan {
			return maxTableSpan, true
		}
		return clampTableSpan(int(typed)), true
	case json.Number:
		parsed, err := strconv.ParseInt(string(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		if parsed < 1 {
			return 1, true
		}
		if parsed > maxTableSpan {
			return maxTableSpan, true
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func clampTableSpan(value int) int {
	if value < 1 {
		return 1
	}
	if value > maxTableSpan {
		return maxTableSpan
	}
	return value
}

// flattenNestedTable 把单元格内的嵌套表格转成一行纯文本，避免递归表格。
func flattenNestedTable(node Node) Node {
	var rows []string
	for _, row := range node.Content {
		var cells []string
		for _, cell := range row.Content {
			if text := strings.TrimSpace(tableNodeText(cell.Content)); text != "" {
				cells = append(cells, text)
			}
		}
		if len(cells) > 0 {
			rows = append(rows, strings.Join(cells, " | "))
		}
	}
	text := strings.TrimSpace(strings.Join(rows, " "))
	if text == "" {
		text = "[table]"
	}
	return Node{Type: "paragraph", Content: []Node{{Type: "text", Text: text}}}
}

func tableNodeText(nodes []Node) string {
	var builder strings.Builder
	var walk func([]Node)
	walk = func(list []Node) {
		for _, node := range list {
			switch node.Type {
			case "text":
				builder.WriteString(node.Text)
			case "hardBreak":
				builder.WriteByte(' ')
			case "sforumEmoji":
				native, _ := node.Attrs["native"].(string)
				if native == "" {
					name, _ := node.Attrs["name"].(string)
					native = ":" + name + ":"
				}
				builder.WriteString(native)
			case "image":
				alt, _ := node.Attrs["alt"].(string)
				builder.WriteString(alt)
			}
			walk(node.Content)
		}
	}
	walk(nodes)
	return builder.String()
}
