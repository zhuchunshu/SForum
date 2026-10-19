package editordocument

import (
	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
)

// goldmarkTableNode 把 GFM 表格 AST 转成 Host table 节点。单元格内的 inline
// 子节点统一包进 paragraph，与原生编辑器的块内容契约对齐。
func goldmarkTableNode(table *extensionast.Table, source []byte) Node {
	node := Node{Type: "table"}
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch typed := child.(type) {
		case *extensionast.TableHeader:
			node.Content = append(node.Content, goldmarkTableRowNode(typed, source, true, table.Alignments))
		case *extensionast.TableRow:
			node.Content = append(node.Content, goldmarkTableRowNode(typed, source, false, table.Alignments))
		}
	}
	return node
}

func goldmarkTableRowNode(row ast.Node, source []byte, header bool, alignments []extensionast.Alignment) Node {
	result := Node{Type: "tableRow"}
	column := 0
	for child := row.FirstChild(); child != nil; child = child.NextSibling() {
		cell, ok := child.(*extensionast.TableCell)
		if !ok {
			continue
		}
		cellType := "tableCell"
		if header {
			cellType = "tableHeader"
		}
		alignment := cell.Alignment
		if alignment == extensionast.AlignNone && column < len(alignments) {
			alignment = alignments[column]
		}
		var attrs map[string]any
		if name := goldmarkTableAlignName(alignment); name != "" {
			attrs = map[string]any{"align": name}
		}
		result.Content = append(result.Content, Node{
			Type:    cellType,
			Attrs:   attrs,
			Content: []Node{{Type: "paragraph", Content: goldmarkInlineChildren(cell, source, nil)}},
		})
		column++
	}
	return result
}

func goldmarkTableAlignName(alignment extensionast.Alignment) string {
	switch alignment {
	case extensionast.AlignLeft:
		return "left"
	case extensionast.AlignRight:
		return "right"
	case extensionast.AlignCenter:
		return "center"
	default:
		return ""
	}
}
