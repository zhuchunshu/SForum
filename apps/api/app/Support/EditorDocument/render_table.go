package editordocument

import (
	"strconv"
	"strings"
)

// renderTable 输出 Host 契约表格 HTML：首段连续全 header 行进 thead，其余
// 进 tbody；不输出 style/colgroup，单元格内按块渲染。
func renderTable(builder *strings.Builder, node Node, schema Schema, plan *shortcodeRenderPlanState, shortcodeDepth int, locale string) {
	rows := tableRows(node)
	if len(rows) == 0 {
		return
	}
	headerRows := 0
	for headerRows < len(rows) && tableRowIsHeader(rows[headerRows]) {
		headerRows++
	}
	builder.WriteString("<table>")
	if headerRows > 0 {
		builder.WriteString("<thead>")
		for _, row := range rows[:headerRows] {
			renderTableRow(builder, row, schema, plan, shortcodeDepth, locale)
		}
		builder.WriteString("</thead>")
	}
	if headerRows < len(rows) {
		builder.WriteString("<tbody>")
		for _, row := range rows[headerRows:] {
			renderTableRow(builder, row, schema, plan, shortcodeDepth, locale)
		}
		builder.WriteString("</tbody>")
	}
	builder.WriteString("</table>")
}

func renderTableRow(builder *strings.Builder, row Node, schema Schema, plan *shortcodeRenderPlanState, shortcodeDepth int, locale string) {
	builder.WriteString("<tr>")
	for _, cell := range row.Content {
		if cell.Type != "tableHeader" && cell.Type != "tableCell" {
			continue
		}
		renderTableCell(builder, cell, schema, plan, shortcodeDepth, locale)
	}
	builder.WriteString("</tr>")
}

func renderTableCell(builder *strings.Builder, cell Node, schema Schema, plan *shortcodeRenderPlanState, shortcodeDepth int, locale string) {
	tag := "td"
	if cell.Type == "tableHeader" {
		tag = "th"
	}
	builder.WriteString("<")
	builder.WriteString(tag)
	if align, ok := cell.Attrs["align"].(string); ok && align != "" {
		writeAttr(builder, "align", align)
	}
	if span, ok := normalizedTableSpan(cell.Attrs["colspan"]); ok && span > 1 {
		builder.WriteString(` colspan="`)
		builder.WriteString(strconv.Itoa(span))
		builder.WriteString(`"`)
	}
	if span, ok := normalizedTableSpan(cell.Attrs["rowspan"]); ok && span > 1 {
		builder.WriteString(` rowspan="`)
		builder.WriteString(strconv.Itoa(span))
		builder.WriteString(`"`)
	}
	builder.WriteString(">")
	for _, child := range cell.Content {
		renderNode(builder, child, schema, plan, shortcodeDepth, locale)
	}
	builder.WriteString("</")
	builder.WriteString(tag)
	builder.WriteString(">")
}

// renderMarkdownTable 输出 GFM 管道表：首行全 header 时作为表头，否则输出
// 空表头行；colspan 用空单元格补齐，单元格内 | 转义、换行转 <br>。
func renderMarkdownTable(builder *strings.Builder, node Node) {
	rows := tableRows(node)
	if len(rows) == 0 {
		return
	}
	columns := 0
	for _, row := range rows {
		if count := tableRowColumns(row); count > columns {
			columns = count
		}
	}
	if columns == 0 {
		return
	}
	if columns > maxTableColumns {
		columns = maxTableColumns
	}
	headerRows := 0
	if tableRowIsHeader(rows[0]) {
		headerRows = 1
	}
	lines := make([]string, 0, len(rows)+1)
	if headerRows == 1 {
		lines = append(lines, markdownTableRow(rows[0], columns))
	} else {
		lines = append(lines, markdownTableEmptyRow(columns))
	}
	// 对齐取自表头行；没有表头时退回第一行的单元格对齐。
	lines = append(lines, markdownTableDivider(rows[0], columns))
	for _, row := range rows[headerRows:] {
		lines = append(lines, markdownTableRow(row, columns))
	}
	builder.WriteString(strings.Join(lines, "\n"))
}

// shortcodeBodyEndsWithTable 判断短代码块正文是否以表格结尾。
func shortcodeBodyEndsWithTable(nodes []Node) bool {
	if len(nodes) == 0 {
		return false
	}
	return nodes[len(nodes)-1].Type == "table"
}

func tableRows(node Node) []Node {
	rows := make([]Node, 0, len(node.Content))
	for _, child := range node.Content {
		if child.Type == "tableRow" {
			rows = append(rows, child)
		}
	}
	return rows
}

func tableRowIsHeader(row Node) bool {
	if len(row.Content) == 0 {
		return false
	}
	for _, cell := range row.Content {
		if cell.Type != "tableHeader" {
			return false
		}
	}
	return true
}

func tableRowColumns(row Node) int {
	columns := 0
	for _, cell := range row.Content {
		if cell.Type != "tableHeader" && cell.Type != "tableCell" {
			continue
		}
		span := 1
		if value, ok := normalizedTableSpan(cell.Attrs["colspan"]); ok {
			span = value
		}
		columns += span
	}
	return columns
}

func markdownTableRow(row Node, columns int) string {
	cells := make([]string, 0, columns)
	for _, cell := range row.Content {
		if cell.Type != "tableHeader" && cell.Type != "tableCell" {
			continue
		}
		span := 1
		if value, ok := normalizedTableSpan(cell.Attrs["colspan"]); ok {
			span = value
		}
		cells = append(cells, markdownTableCellText(cell))
		for index := 1; index < span && len(cells) < columns; index++ {
			cells = append(cells, "")
		}
	}
	for len(cells) < columns {
		cells = append(cells, "")
	}
	if len(cells) > columns {
		cells = cells[:columns]
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

func markdownTableEmptyRow(columns int) string {
	return "| " + strings.Join(make([]string, columns), " | ") + " |"
}

func markdownTableDivider(row Node, columns int) string {
	alignments := make([]string, 0, columns)
	for _, cell := range row.Content {
		if cell.Type != "tableHeader" && cell.Type != "tableCell" {
			continue
		}
		align, _ := cell.Attrs["align"].(string)
		span := 1
		if value, ok := normalizedTableSpan(cell.Attrs["colspan"]); ok {
			span = value
		}
		for index := 0; index < span && len(alignments) < columns; index++ {
			alignments = append(alignments, align)
		}
	}
	for len(alignments) < columns {
		alignments = append(alignments, "")
	}
	cells := make([]string, columns)
	for index, align := range alignments[:columns] {
		switch align {
		case "left":
			cells[index] = ":---"
		case "right":
			cells[index] = "---:"
		case "center":
			cells[index] = ":---:"
		default:
			cells[index] = "---"
		}
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

func markdownTableCellText(cell Node) string {
	blocks := make([]string, 0, len(cell.Content))
	for _, child := range cell.Content {
		var block strings.Builder
		if child.Type == "paragraph" {
			renderMarkdownInline(&block, child.Content)
		} else {
			renderMarkdownNode(&block, child)
		}
		text := strings.TrimSpace(strings.ReplaceAll(block.String(), "\n", "<br>"))
		if text != "" {
			blocks = append(blocks, text)
		}
	}
	return strings.ReplaceAll(strings.Join(blocks, "<br>"), "|", `\|`)
}
