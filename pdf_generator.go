package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// AcademicPDFGenerator produces high-fidelity, research-grade PDFs from blog manuscripts.
type AcademicPDFGenerator struct {
	blog        *Blog
	rawMarkdown string
	pdf         *gofpdf.Fpdf
	figCounter  int
}

func NewAcademicPDFGenerator(blog *Blog, rawMarkdown string) *AcademicPDFGenerator {
	return &AcademicPDFGenerator{
		blog:        blog,
		rawMarkdown: rawMarkdown,
	}
}

// Generate builds the complete publication PDF and returns the raw bytes
func (g *AcademicPDFGenerator) Generate() ([]byte, error) {
	// Standard A4 page (210 x 297 mm) with 18mm margins
	pdf := gofpdf.New("P", "mm", "A4", "")
	g.pdf = pdf
	g.figCounter = 0

	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AliasNbPages("{nb}")

	// Running Academic Header
	pdf.SetHeaderFunc(func() {
		pdf.SetY(10)
		pdf.SetFont("Helvetica", "B", 7)
		pdf.SetTextColor(71, 85, 105) // #475569

		// Left Header: Brand
		pdf.SetX(18)
		pdf.CellFormat(80, 5, "MINTSPACE OPEN RESEARCH ARCHIVE", "", 0, "L", false, 0, "")

		// Right Header: DOI & Paper ID
		paperIDText := fmt.Sprintf("PAPER ID: MS-%d  |  DOI: 10.mintspace/paper.%d", g.blog.Id, g.blog.Id)
		pdf.CellFormat(96, 5, paperIDText, "", 0, "R", false, 0, "")

		// Thin rule under header
		pdf.SetDrawColor(203, 213, 225) // #cbd5e1
		pdf.SetLineWidth(0.3)
		pdf.Line(18, 15.5, 192, 15.5)
		pdf.Ln(4)
	})

	// Running Academic Footer
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		// Thin rule above footer
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetLineWidth(0.3)
		pdf.Line(18, 283, 192, 283)

		pdf.SetY(-12)
		pdf.SetFont("Courier", "", 6.5)
		pdf.SetTextColor(100, 116, 139)

		// Truncate hash if necessary for footer fit
		shortHash := g.blog.ContentHash
		if len(shortHash) > 40 {
			shortHash = shortHash[:40] + "..."
		}
		hashText := fmt.Sprintf("SHA-256 CAS: %s  |  %s", shortHash, g.blog.DateCreated)
		pdf.CellFormat(120, 5, hashText, "", 0, "L", false, 0, "")

		// Page numbers: Page X of Y
		pdf.SetFont("Times", "", 8)
		pageText := fmt.Sprintf("Page %d of {nb}", pdf.PageNo())
		pdf.CellFormat(54, 5, pageText, "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	// 1. Title Block
	g.renderTitleBlock()

	// 2. Abstract Block
	if strings.TrimSpace(g.blog.Abstract) != "" {
		g.renderAbstractBlock()
	}

	// 3. Body Content (Markdown Parsing)
	g.renderMarkdownBody()

	// 4. Archival Colophon
	g.renderColophon()

	// Output buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *AcademicPDFGenerator) renderTitleBlock() {
	pdf := g.pdf

	pdf.Ln(4)

	// Article Title
	pdf.SetFont("Times", "B", 20)
	pdf.SetTextColor(15, 23, 42) // #0f172a
	pdf.MultiCell(174, 8, sanitizeForPDF(g.blog.Title), "", "L", false)
	pdf.Ln(3)

	// Author Information
	pdf.SetFont("Times", "B", 11)
	pdf.SetTextColor(30, 41, 59)
	authorName := g.blog.AuthorName
	if authorName == "" {
		authorName = g.blog.AuthorEmail
	}
	pdf.CellFormat(174, 5, sanitizeForPDF(authorName), "", 1, "L", false, 0, "")

	pdf.SetFont("Times", "I", 9.5)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(174, 4.5, sanitizeForPDF(g.blog.AuthorEmail), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Publication Metadata Strip
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(100, 116, 139)

	metaStr := fmt.Sprintf("Date Published: %s    |    Access Policy: %s    |    CAS Multi-Hash Verification Verified",
		g.blog.DateCreated,
		ternary(g.blog.IsPrivate, "Restricted", "Open Access Archival Copy"))
	pdf.CellFormat(174, 4, metaStr, "", 1, "L", false, 0, "")

	// Accent rule under header
	pdf.SetDrawColor(15, 23, 42)
	pdf.SetLineWidth(0.6)
	pdf.Line(18, pdf.GetY()+2, 192, pdf.GetY()+2)
	pdf.Ln(5)
}

func (g *AcademicPDFGenerator) renderAbstractBlock() {
	pdf := g.pdf

	abstractY := pdf.GetY()
	pdf.SetFont("Times", "I", 9)
	lines := pdf.SplitLines([]byte(sanitizeForPDF(g.blog.Abstract)), 160)
	boxHeight := float64(len(lines))*4.5 + 10

	// Check page break
	if abstractY+boxHeight > 270 {
		pdf.AddPage()
		abstractY = pdf.GetY()
	}

	// Shaded abstract box
	pdf.SetFillColor(248, 250, 252) // #f8fafc
	pdf.SetDrawColor(226, 232, 240) // #e2e8f0
	pdf.SetLineWidth(0.2)
	pdf.Rect(18, abstractY, 174, boxHeight, "DF")

	// Left accent bar
	pdf.SetFillColor(15, 23, 42) // #0f172a
	pdf.Rect(18, abstractY, 2.5, boxHeight, "F")

	// Abstract heading
	pdf.SetXY(23, abstractY+2.5)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(160, 4, "ABSTRACT", "", 1, "L", false, 0, "")

	// Abstract text
	pdf.SetXY(23, abstractY+7)
	pdf.SetFont("Times", "I", 9)
	pdf.SetTextColor(51, 65, 85)
	pdf.MultiCell(162, 4.3, sanitizeForPDF(g.blog.Abstract), "", "L", false)

	pdf.SetY(abstractY + boxHeight + 4)
}

func (g *AcademicPDFGenerator) renderMarkdownBody() {
	lines := strings.Split(g.rawMarkdown, "\n")
	inCodeBlock := false
	codeBlockLang := ""
	var codeLines []string

	inMermaidBlock := false
	var mermaidLines []string

	inTable := false
	var tableRows [][]string

	for i := 0; i < len(lines); i++ {
		rawLine := lines[i]
		trimmed := strings.TrimSpace(rawLine)

		// 1. Code block boundary check (```)
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock && !inMermaidBlock {
				lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				langLower := strings.ToLower(lang)
				if langLower == "mermaid" || langLower == "flowchart" {
					inMermaidBlock = true
					mermaidLines = []string{}
				} else {
					inCodeBlock = true
					codeBlockLang = strings.ToUpper(lang)
					codeLines = []string{}
				}
				continue
			} else if inMermaidBlock {
				inMermaidBlock = false
				g.renderMermaidDiagram(mermaidLines)
				continue
			} else if inCodeBlock {
				inCodeBlock = false
				g.renderCodeBlock(codeBlockLang, codeLines)
				continue
			}
		}

		if inMermaidBlock {
			mermaidLines = append(mermaidLines, rawLine)
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, rawLine)
			continue
		}

		// 2. Table row check (| ... |)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			if strings.Contains(trimmed, "---") {
				// Table header divider, skip
				continue
			}
			parts := strings.Split(trimmed, "|")
			var row []string
			for _, p := range parts[1 : len(parts)-1] {
				row = append(row, strings.TrimSpace(p))
			}
			tableRows = append(tableRows, row)
			inTable = true
			continue
		} else if inTable {
			inTable = false
			g.renderTable(tableRows)
			tableRows = [][]string{}
		}

		// 3. Headings
		if strings.HasPrefix(trimmed, "### ") {
			g.renderHeading(3, strings.TrimPrefix(trimmed, "### "))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			g.renderHeading(2, strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			g.renderHeading(1, strings.TrimPrefix(trimmed, "# "))
			continue
		}

		// 4. Blockquotes (> ...)
		if strings.HasPrefix(trimmed, ">") {
			g.renderBlockquote(strings.TrimPrefix(trimmed, ">"))
			continue
		}

		// 5. Horizontal rule (--- or ***)
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			g.renderHorizontalRule()
			continue
		}

		// 6. Bullet lists (* or -)
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			g.renderBullet(strings.TrimSpace(trimmed[2:]))
			continue
		}

		// 7. Numbered lists (1. , 2. )
		numberedRegex := regexp.MustCompile(`^(\d+)\.\s+(.*)$`)
		if matches := numberedRegex.FindStringSubmatch(trimmed); len(matches) == 3 {
			g.renderNumberedItem(matches[1], matches[2])
			continue
		}

		// 8. Math block ($$...$$)
		if strings.HasPrefix(trimmed, "$$") && strings.HasSuffix(trimmed, "$$") && len(trimmed) > 4 {
			mathExpr := strings.TrimSuffix(strings.TrimPrefix(trimmed, "$$"), "$$")
			g.renderMathEquation(mathExpr)
			continue
		}

		// 9. Standard body paragraph
		if trimmed != "" {
			g.renderParagraph(trimmed)
		} else {
			// Spacing between paragraphs
			g.pdf.Ln(2)
		}
	}

	// Flush any trailing blocks
	if inCodeBlock {
		g.renderCodeBlock(codeBlockLang, codeLines)
	}
	if inMermaidBlock {
		g.renderMermaidDiagram(mermaidLines)
	}
	if inTable && len(tableRows) > 0 {
		g.renderTable(tableRows)
	}
}

func (g *AcademicPDFGenerator) renderHeading(level int, title string) {
	pdf := g.pdf

	// Prevent orphan headings near the bottom of a page
	if pdf.GetY() > 245 {
		pdf.AddPage()
	} else {
		pdf.Ln(4)
	}

	cleanTitle := sanitizeForPDF(title)

	switch level {
	case 1:
		pdf.SetFont("Times", "B", 14)
		pdf.SetTextColor(15, 23, 42)
		pdf.MultiCell(174, 6.5, cleanTitle, "", "L", false)
		// Subtle bottom rule for H1
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetLineWidth(0.3)
		pdf.Line(18, pdf.GetY()+1, 192, pdf.GetY()+1)
		pdf.Ln(3)
	case 2:
		pdf.SetFont("Times", "B", 12)
		pdf.SetTextColor(30, 41, 59)
		pdf.MultiCell(174, 5.5, cleanTitle, "", "L", false)
		pdf.Ln(2)
	default:
		pdf.SetFont("Times", "B", 10.5)
		pdf.SetTextColor(51, 65, 85)
		pdf.MultiCell(174, 5, cleanTitle, "", "L", false)
		pdf.Ln(1.5)
	}
}

func (g *AcademicPDFGenerator) renderParagraph(text string) {
	pdf := g.pdf

	// Clean markdown formatting: **bold**, `code`, etc.
	clean := sanitizeMarkdownInline(text)

	pdf.SetFont("Times", "", 10)
	pdf.SetTextColor(30, 41, 59) // #1e293b
	pdf.MultiCell(174, 4.8, clean, "", "L", false)
	pdf.Ln(1.5)
}

func (g *AcademicPDFGenerator) renderBullet(text string) {
	pdf := g.pdf
	clean := sanitizeMarkdownInline(text)

	x := pdf.GetX()
	y := pdf.GetY()

	// Bullet point symbol
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.SetXY(x+2, y+0.5)
	pdf.CellFormat(3, 4, "-", "", 0, "L", false, 0, "")

	// Content text indented
	pdf.SetXY(x+6, y)
	pdf.SetFont("Times", "", 9.5)
	pdf.SetTextColor(30, 41, 59)
	pdf.MultiCell(168, 4.5, clean, "", "L", false)
	pdf.Ln(1)
}

func (g *AcademicPDFGenerator) renderNumberedItem(num string, text string) {
	pdf := g.pdf
	clean := sanitizeMarkdownInline(text)

	x := pdf.GetX()
	y := pdf.GetY()

	// Number label
	pdf.SetFont("Times", "B", 9.5)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetXY(x+2, y)
	pdf.CellFormat(6, 4.5, num+".", "", 0, "L", false, 0, "")

	// Content text indented
	pdf.SetXY(x+8, y)
	pdf.SetFont("Times", "", 9.5)
	pdf.SetTextColor(30, 41, 59)
	pdf.MultiCell(166, 4.5, clean, "", "L", false)
	pdf.Ln(1)
}

func (g *AcademicPDFGenerator) renderBlockquote(text string) {
	pdf := g.pdf
	clean := sanitizeMarkdownInline(strings.TrimSpace(text))

	y := pdf.GetY()
	lines := pdf.SplitLines([]byte(clean), 164)
	h := float64(len(lines))*4.5 + 4

	if y+h > 270 {
		pdf.AddPage()
		y = pdf.GetY()
	}

	// Light background fill
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(18, y, 174, h, "F")

	// Left bar
	pdf.SetFillColor(71, 85, 105)
	pdf.Rect(18, y, 1.5, h, "F")

	pdf.SetXY(23, y+2)
	pdf.SetFont("Times", "I", 9.5)
	pdf.SetTextColor(51, 65, 85)
	pdf.MultiCell(165, 4.5, clean, "", "L", false)

	pdf.SetY(y + h + 2)
}

func (g *AcademicPDFGenerator) renderHorizontalRule() {
	pdf := g.pdf
	pdf.Ln(2)
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.4)
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	pdf.Ln(3)
}

func (g *AcademicPDFGenerator) renderCodeBlock(lang string, lines []string) {
	pdf := g.pdf

	// Calculate code block height
	lineHeight := 3.8
	headerHeight := 6.0
	padding := 5.0
	totalHeight := headerHeight + float64(len(lines))*lineHeight + padding

	// If block is moderately sized and doesn't fit on current page, break to new page
	if totalHeight < 120 && pdf.GetY()+totalHeight > 270 {
		pdf.AddPage()
	}

	yStart := pdf.GetY()
	currentLines := lines

	// If block is extremely long (>1 page), split it gracefully
	maxLinesPerPage := int((265 - yStart - headerHeight) / lineHeight)
	if maxLinesPerPage < 4 && yStart > 100 {
		pdf.AddPage()
		yStart = pdf.GetY()
		maxLinesPerPage = int((265 - yStart - headerHeight) / lineHeight)
	}

	// Header tag
	pdf.SetFillColor(241, 245, 249) // #f1f5f9
	pdf.SetDrawColor(203, 213, 225) // #cbd5e1
	pdf.SetLineWidth(0.2)

	if lang == "" {
		lang = "CODE"
	}

	pdf.Rect(18, yStart, 174, headerHeight, "DF")
	pdf.SetXY(22, yStart+1)
	pdf.SetFont("Courier", "B", 7.5)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(166, 4, lang, "", 0, "L", false, 0, "")

	pdf.SetY(yStart + headerHeight)

	// Code container
	codeY := pdf.GetY()
	pdf.SetFont("Courier", "", 7.5)
	pdf.SetTextColor(15, 23, 42)

	for idx, l := range currentLines {
		if pdf.GetY() > 268 {
			pdf.AddPage()
			pdf.SetY(22)
		}

		cleanL := sanitizeForPDF(l)
		// Indent code line
		pdf.SetX(22)
		pdf.CellFormat(168, lineHeight, cleanL, "", 1, "L", false, 0, "")
		_ = idx
	}

	// Border around code
	pdf.SetDrawColor(203, 213, 225)
	pdf.Rect(18, codeY, 174, pdf.GetY()-codeY+2, "D")
	pdf.Ln(4)
}

func (g *AcademicPDFGenerator) renderMathEquation(expr string) {
	pdf := g.pdf

	if pdf.GetY()+18 > 270 {
		pdf.AddPage()
	}

	pdf.Ln(2)
	y := pdf.GetY()

	// Math equation container box
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.2)
	pdf.Rect(25, y, 160, 10, "DF")

	pdf.SetXY(25, y+2)
	pdf.SetFont("Times", "I", 10)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(160, 6, sanitizeForPDF(expr), "", 0, "C", false, 0, "")

	pdf.SetY(y + 13)
}

// renderMermaidDiagram parses Mermaid flowchart DSL and renders an elegant, readable vector diagram
func (g *AcademicPDFGenerator) renderMermaidDiagram(lines []string) {
	pdf := g.pdf
	g.figCounter++

	// Extract nodes and relationships
	type Node struct {
		ID    string
		Label string
	}
	type Edge struct {
		From  string
		To    string
		Label string
	}

	nodesMap := make(map[string]string)
	var orderedNodes []Node
	var edges []Edge
	var subgraphs []string

	nodeRegex := regexp.MustCompile(`([A-Za-z0-9_-]+)\[([^\]]+)\]`)
	edgeRegex := regexp.MustCompile(`([A-Za-z0-9_-]+)\s*(?:-->|---|-.->)\s*(?:\|([^|]+)\|)?\s*([A-Za-z0-9_-]+)`)

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "subgraph") {
			sgName := strings.TrimSpace(strings.TrimPrefix(trimmed, "subgraph"))
			subgraphs = append(subgraphs, sgName)
			continue
		}
		if trimmed == "end" {
			continue
		}

		// Match nodes
		if matches := nodeRegex.FindAllStringSubmatch(l, -1); len(matches) > 0 {
			for _, m := range matches {
				id := m[1]
				lbl := m[2]
				lbl = strings.ReplaceAll(lbl, "<br/>", " ")
				lbl = strings.ReplaceAll(lbl, "<br>", " ")
				if _, exists := nodesMap[id]; !exists {
					nodesMap[id] = lbl
					orderedNodes = append(orderedNodes, Node{ID: id, Label: lbl})
				}
			}
		}

		// Match edges
		if matches := edgeRegex.FindAllStringSubmatch(l, -1); len(matches) > 0 {
			for _, m := range matches {
				edges = append(edges, Edge{
					From:  m[1],
					Label: m[2],
					To:    m[3],
				})
			}
		}
	}

	// Determine diagram height
	numNodes := len(orderedNodes)
	if numNodes == 0 {
		// Fallback to text representation
		g.renderCodeBlock("ARCHITECTURE DIAGRAM", lines)
		return
	}

	boxHeight := float64(numNodes)*11 + 22
	if boxHeight > 140 {
		boxHeight = 140
	}

	if pdf.GetY()+boxHeight+15 > 270 {
		pdf.AddPage()
	}

	yStart := pdf.GetY() + 2

	// Figure Outer Frame
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.3)
	pdf.Rect(18, yStart, 174, boxHeight+10, "DF")

	// Figure Header Caption
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(18, yStart, 174, 6.5, "DF")
	pdf.SetXY(22, yStart+1.2)
	pdf.SetFont("Helvetica", "B", 7.5)
	pdf.SetTextColor(71, 85, 105)

	captionText := fmt.Sprintf("FIGURE %d: ARCHITECTURE FLOWCHART SPECIFICATION", g.figCounter)
	if len(subgraphs) > 0 {
		captionText += fmt.Sprintf(" (%s)", strings.Join(subgraphs, " / "))
	}
	pdf.CellFormat(166, 4, captionText, "", 0, "L", false, 0, "")

	// Render node cards inside figure
	curY := yStart + 9.0
	for idx, n := range orderedNodes {
		if curY+9.5 > yStart+boxHeight {
			break
		}

		// Node Box
		nodeW := 154.0
		nodeH := 7.0
		pdf.SetFillColor(241, 245, 249)
		pdf.SetDrawColor(100, 116, 139)
		pdf.SetLineWidth(0.2)
		pdf.RoundedRect(28, curY, nodeW, nodeH, 1.2, "1234", "DF")

		// Node Text
		pdf.SetXY(30, curY+1.2)
		pdf.SetFont("Helvetica", "B", 7.5)
		pdf.SetTextColor(15, 23, 42)
		cleanLbl := sanitizeForPDF(n.Label)
		if len(cleanLbl) > 85 {
			cleanLbl = cleanLbl[:82] + "..."
		}
		pdf.CellFormat(nodeW-4, 4.5, cleanLbl, "", 0, "C", false, 0, "")

		// Arrow downward to next node if sequential
		if idx < len(orderedNodes)-1 {
			arrowY := curY + nodeH
			pdf.SetDrawColor(71, 85, 105)
			pdf.SetLineWidth(0.4)
			pdf.Line(105, arrowY, 105, arrowY+3)

			// Arrowhead
			pdf.Line(103.5, arrowY+2, 105, arrowY+3)
			pdf.Line(106.5, arrowY+2, 105, arrowY+3)
		}

		curY += 10.0
	}

	pdf.SetY(yStart + boxHeight + 14)
}

func (g *AcademicPDFGenerator) renderTable(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	pdf := g.pdf

	numCols := len(rows[0])
	if numCols == 0 {
		return
	}

	colW := 174.0 / float64(numCols)
	rowH := 6.0

	tableHeight := float64(len(rows))*rowH + 6
	if pdf.GetY()+tableHeight > 270 {
		pdf.AddPage()
	}

	pdf.Ln(2)
	pdf.SetDrawColor(15, 23, 42)
	pdf.SetLineWidth(0.5)

	// Top rule
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())

	for rIdx, row := range rows {
		isHeader := rIdx == 0

		if isHeader {
			pdf.SetFont("Times", "B", 8.5)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetFillColor(241, 245, 249)
		} else {
			pdf.SetFont("Times", "", 8.5)
			pdf.SetTextColor(51, 65, 85)
			if rIdx%2 == 0 {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
		}

		pdf.SetX(18)
		for cIdx := 0; cIdx < numCols; cIdx++ {
			val := ""
			if cIdx < len(row) {
				val = sanitizeForPDF(row[cIdx])
			}
			pdf.CellFormat(colW, rowH, val, "", 0, "L", true, 0, "")
		}
		pdf.Ln(rowH)

		// Header bottom rule
		if isHeader {
			pdf.SetLineWidth(0.3)
			pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
		}
	}

	// Bottom rule
	pdf.SetLineWidth(0.5)
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	pdf.Ln(4)
}

func (g *AcademicPDFGenerator) renderColophon() {
	pdf := g.pdf

	if pdf.GetY()+22 > 270 {
		pdf.AddPage()
	}

	pdf.Ln(4)
	pdf.SetDrawColor(15, 23, 42)
	pdf.SetLineWidth(0.4)
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(174, 3.5, "ARCHIVAL PROVENANCE & CRYPTOGRAPHIC VERIFICATION", "", 1, "L", false, 0, "")

	pdf.SetFont("Courier", "", 7)
	pdf.SetTextColor(51, 65, 85)
	pdf.CellFormat(174, 3.5, fmt.Sprintf("SHA-256 CAS: %s", g.blog.ContentHash), "", 1, "L", false, 0, "")

	pdf.SetFont("Times", "", 7.5)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(174, 3.5, fmt.Sprintf("MintSpace Open Research Series  |  Permanent Canonical Record  |  Published %s", g.blog.DateCreated), "", 1, "L", false, 0, "")
	pdf.Ln(4)
}

// ----------------- STRING & ENCODING UTILITIES -----------------

// sanitizeForPDF maps Unicode characters to standard Latin-1 equivalents supported by PostScript fonts
func sanitizeForPDF(input string) string {
	r := strings.NewReplacer(
		"\u2014", "---",
		"\u2013", "--",
		"\u2018", "'",
		"\u2019", "'",
		"\u201c", "\"",
		"\u201d", "\"",
		"\u2022", "*",
		"\u2192", "->",
		"\u2190", "<-",
		"\u2194", "<->",
		"\u2264", "<=",
		"\u2265", ">=",
		"\u2260", "!=",
		"\u2248", "~=",
		"\u00d7", "x",
		"\u00f7", "/",
		"\u00b1", "+/-",
		"\u221e", "INF",
		"\u2713", "[x]",
		"\u2717", "[ ]",
		"\u2026", "...",
	)
	res := r.Replace(input)

	// Strip remaining non-ASCII characters that would cause PDF encoding glitches
	var clean strings.Builder
	for _, c := range res {
		if c >= 32 && c <= 126 {
			clean.WriteRune(c)
		} else if c == '\n' || c == '\t' || c == '\r' {
			clean.WriteRune(c)
		} else if c >= 160 && c <= 255 {
			clean.WriteRune(c)
		} else {
			clean.WriteRune(' ')
		}
	}
	return clean.String()
}

// sanitizeMarkdownInline strips inline bold/italic markers for clean text display
func sanitizeMarkdownInline(input string) string {
	res := sanitizeForPDF(input)
	// Strip bold **text** -> text
	res = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(res, "$1")
	// Strip italic *text* -> text
	res = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(res, "$1")
	// Strip inline code `code` -> code
	res = regexp.MustCompile("`([^`]+)`").ReplaceAllString(res, "$1")
	// Strip links [text](url) -> text (url)
	res = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(res, "$1 ($2)")
	return res
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
