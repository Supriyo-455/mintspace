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

	lineHeight := 3.8
	headerHeight := 6.0
	padding := 4.0
	totalHeight := headerHeight + float64(len(lines))*lineHeight + padding

	// If block is moderately sized and doesn't fit on current page, break to new page
	if totalHeight < 110 && pdf.GetY()+totalHeight > 265 {
		pdf.AddPage()
	}

	yStart := pdf.GetY()
	if yStart > 240 {
		pdf.AddPage()
		yStart = pdf.GetY()
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

	cleanLang := sanitizeForPDF(lang)
	if pdf.GetStringWidth(cleanLang) > 160 {
		cleanLang = cleanLang[:20] + "..."
	}
	pdf.CellFormat(166, 4, cleanLang, "", 0, "L", false, 0, "")

	pdf.SetY(yStart + headerHeight)

	// Code container lines
	pageStartY := pdf.GetY()
	pdf.SetFont("Courier", "", 7.5)
	pdf.SetTextColor(15, 23, 42)

	maxLineW := 166.0

	for _, l := range lines {
		if pdf.GetY()+lineHeight > 265 {
			// Finish border on current page
			pdf.SetDrawColor(203, 213, 225)
			pdf.Rect(18, pageStartY, 174, pdf.GetY()-pageStartY+1, "D")

			pdf.AddPage()
			pdf.SetY(20)
			pageStartY = 20
			pdf.SetFont("Courier", "", 7.5)
			pdf.SetTextColor(15, 23, 42)
		}

		cleanL := sanitizeForPDF(l)
		// Truncate line if it exceeds box width so it never overshoots horizontally
		if pdf.GetStringWidth(cleanL) > maxLineW {
			for len(cleanL) > 4 && pdf.GetStringWidth(cleanL+"...") > maxLineW {
				cleanL = cleanL[:len(cleanL)-1]
			}
			cleanL += "..."
		}

		pdf.SetX(22)
		pdf.CellFormat(168, lineHeight, cleanL, "", 1, "L", false, 0, "")
	}

	// Draw closing border on final page
	pdf.SetDrawColor(203, 213, 225)
	pdf.Rect(18, pageStartY, 174, pdf.GetY()-pageStartY+1, "D")
	pdf.Ln(4)
}

func (g *AcademicPDFGenerator) renderMathEquation(expr string) {
	pdf := g.pdf

	if pdf.GetY()+18 > 265 {
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

// renderMermaidDiagram parses Mermaid DSL and routes to specialized flowchart, sequence, or class diagram renderers
func (g *AcademicPDFGenerator) renderMermaidDiagram(lines []string) {
	// Detect diagram type
	diagramType := "flowchart"
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "sequenceDiagram") {
			diagramType = "sequence"
			break
		} else if strings.HasPrefix(trimmed, "classDiagram") {
			diagramType = "class"
			break
		} else if strings.HasPrefix(trimmed, "flowchart") || strings.HasPrefix(trimmed, "graph") {
			diagramType = "flowchart"
			break
		}
	}

	switch diagramType {
	case "sequence":
		g.renderSequenceDiagram(lines)
	case "class":
		g.renderClassDiagram(lines)
	default:
		g.renderFlowchart(lines)
	}
}

func (g *AcademicPDFGenerator) renderSequenceDiagram(lines []string) {
	g.figCounter++
	pdf := g.pdf

	type SeqStep struct {
		From string
		To   string
		Text string
	}
	var steps []SeqStep
	msgRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*(?:->>|-->>|->|-->)\s*([A-Za-z0-9_]+)\s*:\s*(.*)$`)

	for _, l := range lines {
		if m := msgRegex.FindStringSubmatch(l); len(m) == 4 {
			steps = append(steps, SeqStep{
				From: m[1],
				To:   m[2],
				Text: strings.TrimSpace(m[3]),
			})
		}
	}

	if len(steps) == 0 {
		g.renderCodeBlock("ARCHITECTURE SEQUENCE SPECIFICATION", lines)
		return
	}

	stepH := 7.0
	boxHeight := float64(len(steps))*stepH + 13.0
	if boxHeight > 180 {
		boxHeight = 180
	}

	if pdf.GetY()+boxHeight+8 > 265 {
		pdf.AddPage()
	}

	yStart := pdf.GetY() + 2

	// Figure Outer Frame
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.3)
	pdf.Rect(18, yStart, 174, boxHeight, "DF")

	// Header Caption
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(18, yStart, 174, 6.5, "DF")
	pdf.SetXY(22, yStart+1.2)
	pdf.SetFont("Helvetica", "B", 7.5)
	pdf.SetTextColor(71, 85, 105)
	capText := fmt.Sprintf("FIGURE %d: ARCHITECTURAL SEQUENCE SPECIFICATION", g.figCounter)
	pdf.CellFormat(166, 4, capText, "", 0, "L", false, 0, "")

	// Render sequence steps
	curY := yStart + 8.5
	for idx, s := range steps {
		if curY+stepH > yStart+boxHeight {
			break
		}

		// Step Pill
		pdf.SetFillColor(241, 245, 249)
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetLineWidth(0.2)
		pdf.RoundedRect(22, curY, 15, 5.2, 1.0, "1234", "DF")
		pdf.SetXY(22, curY+0.6)
		pdf.SetFont("Helvetica", "B", 6.5)
		pdf.SetTextColor(71, 85, 105)
		pdf.CellFormat(15, 4, fmt.Sprintf("Step %d", idx+1), "", 0, "C", false, 0, "")

		// Actor direction pill
		actorW := 38.0
		pdf.RoundedRect(39, curY, actorW, 5.2, 1.0, "1234", "DF")
		pdf.SetXY(39, curY+0.6)
		pdf.SetFont("Helvetica", "B", 6.5)
		pdf.SetTextColor(15, 23, 42)
		actorStr := fmt.Sprintf("%s -> %s", s.From, s.To)
		if pdf.GetStringWidth(actorStr) > actorW-2 {
			actorStr = s.From + "->" + s.To
		}
		pdf.CellFormat(actorW, 4, actorStr, "", 0, "C", false, 0, "")

		// Action description text
		descX := 80.0
		descW := 108.0
		pdf.SetXY(descX, curY+0.6)
		pdf.SetFont("Times", "", 7.5)
		pdf.SetTextColor(30, 41, 59)
		cleanDesc := sanitizeForPDF(s.Text)
		if pdf.GetStringWidth(cleanDesc) > descW {
			for len(cleanDesc) > 4 && pdf.GetStringWidth(cleanDesc+"...") > descW {
				cleanDesc = cleanDesc[:len(cleanDesc)-1]
			}
			cleanDesc += "..."
		}
		pdf.CellFormat(descW, 4, cleanDesc, "", 0, "L", false, 0, "")

		curY += stepH
	}

	pdf.SetY(yStart + boxHeight + 6)
}

func (g *AcademicPDFGenerator) renderClassDiagram(lines []string) {
	g.figCounter++
	pdf := g.pdf

	type ClassBlock struct {
		Name    string
		Members []string
	}
	var classes []ClassBlock
	var curClass *ClassBlock

	classHeaderRegex := regexp.MustCompile(`^\s*class\s+([A-Za-z0-9_]+)\s*\{?`)

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if m := classHeaderRegex.FindStringSubmatch(trimmed); len(m) == 2 {
			if curClass != nil {
				classes = append(classes, *curClass)
			}
			curClass = &ClassBlock{Name: m[1]}
			continue
		}
		if trimmed == "}" {
			if curClass != nil {
				classes = append(classes, *curClass)
				curClass = nil
			}
			continue
		}
		if curClass != nil && (strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "-")) {
			curClass.Members = append(curClass.Members, trimmed)
		}
	}
	if curClass != nil {
		classes = append(classes, *curClass)
	}

	if len(classes) == 0 {
		g.renderCodeBlock("CLASS & DATA STRUCTURE SPECIFICATION", lines)
		return
	}

	// Calculate layout
	boxHeight := 0.0
	for _, c := range classes {
		cardH := float64(len(c.Members))*5.0 + 9.0
		boxHeight += cardH + 3.0
	}
	boxHeight += 12.0
	if boxHeight > 200 {
		boxHeight = 200
	}

	if pdf.GetY()+boxHeight+8 > 265 {
		pdf.AddPage()
	}

	yStart := pdf.GetY() + 2

	// Outer frame
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.3)
	pdf.Rect(18, yStart, 174, boxHeight, "DF")

	// Caption bar
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(18, yStart, 174, 6.5, "DF")
	pdf.SetXY(22, yStart+1.2)
	pdf.SetFont("Helvetica", "B", 7.5)
	pdf.SetTextColor(71, 85, 105)
	capText := fmt.Sprintf("FIGURE %d: DATA STRUCTURE SPECIFICATION", g.figCounter)
	pdf.CellFormat(166, 4, capText, "", 0, "L", false, 0, "")

	curY := yStart + 9.0
	for _, c := range classes {
		cardH := float64(len(c.Members))*5.0 + 8.0
		if curY+cardH > yStart+boxHeight {
			break
		}

		// Class header
		pdf.SetFillColor(241, 245, 249)
		pdf.SetDrawColor(100, 116, 139)
		pdf.SetLineWidth(0.2)
		pdf.Rect(24, curY, 162, cardH, "DF")

		pdf.Rect(24, curY, 162, 5.5, "DF")
		pdf.SetXY(26, curY+0.8)
		pdf.SetFont("Courier", "B", 7.5)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(158, 4, "struct "+c.Name, "", 0, "L", false, 0, "")

		mY := curY + 6.2
		pdf.SetFont("Courier", "", 7.0)
		pdf.SetTextColor(51, 65, 85)
		for _, m := range c.Members {
			cleanM := sanitizeForPDF(m)
			if pdf.GetStringWidth(cleanM) > 156 {
				cleanM = cleanM[:75] + "..."
			}
			pdf.SetXY(28, mY)
			pdf.CellFormat(154, 4.5, cleanM, "", 0, "L", false, 0, "")
			mY += 5.0
		}

		curY += cardH + 3.0
	}

	pdf.SetY(yStart + boxHeight + 6)
}

type FlowchartNode struct {
	ID       string
	Label    string
	Subgraph string
}

func (g *AcademicPDFGenerator) renderFlowchart(lines []string) {
	g.figCounter++
	pdf := g.pdf

	nodesMap := make(map[string]bool)
	var orderedNodes []FlowchartNode
	var subgraphs []string
	subgraphNodeMap := make(map[string][]FlowchartNode)

	currentSubgraph := ""

	// Support all Mermaid node shapes: [label], ["label"], [(label)], ([label]), {label}
	nodeRegex := regexp.MustCompile(`([A-Za-z0-9_-]+)(?:\[\((.*?)\)\]|\["(.*?)"\]|\[(.*?)\]|\((.*?)\)|\{(.*?)\})`)

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "subgraph") {
			sgName := strings.TrimSpace(strings.TrimPrefix(trimmed, "subgraph"))
			// Clean any trailing or bracketed annotation
			if idx := strings.Index(sgName, "["); idx != -1 {
				sgName = strings.TrimSpace(sgName[:idx])
			}
			subgraphs = append(subgraphs, sgName)
			currentSubgraph = sgName
			continue
		}
		if trimmed == "end" {
			currentSubgraph = ""
			continue
		}

		if matches := nodeRegex.FindAllStringSubmatch(l, -1); len(matches) > 0 {
			for _, m := range matches {
				id := m[1]
				// Find first non-empty match group for label
				lbl := ""
				for gIdx := 2; gIdx < len(m); gIdx++ {
					if m[gIdx] != "" {
						lbl = m[gIdx]
						break
					}
				}
				if lbl == "" {
					lbl = id
				}
				lbl = strings.ReplaceAll(lbl, "<br/>", " ")
				lbl = strings.ReplaceAll(lbl, "<br>", " ")
				lbl = strings.Trim(lbl, "\"")

				if !nodesMap[id] {
					nodesMap[id] = true
					nodeObj := FlowchartNode{ID: id, Label: lbl, Subgraph: currentSubgraph}
					orderedNodes = append(orderedNodes, nodeObj)
					if currentSubgraph != "" {
						subgraphNodeMap[currentSubgraph] = append(subgraphNodeMap[currentSubgraph], nodeObj)
					}
				}
			}
		}
	}

	numNodes := len(orderedNodes)
	if numNodes == 0 {
		g.renderCodeBlock("ARCHITECTURE DIAGRAM", lines)
		return
	}

	// Figure Header Caption (Guaranteed fit inside header cell)
	captionPrefix := fmt.Sprintf("FIGURE %d: ARCHITECTURE FLOWCHART SPECIFICATION", g.figCounter)
	fullCaption := captionPrefix
	if len(subgraphs) > 0 {
		fullCaption += fmt.Sprintf(" (%s)", strings.Join(subgraphs, " / "))
	}
	fullCaption = sanitizeForPDF(fullCaption)

	maxCapW := 162.0
	pdf.SetFont("Helvetica", "B", 7.5)
	if pdf.GetStringWidth(fullCaption) > maxCapW {
		for len(fullCaption) > 10 && pdf.GetStringWidth(fullCaption+"...") > maxCapW {
			fullCaption = fullCaption[:len(fullCaption)-1]
		}
		fullCaption += "..."
	}

	// Check if this diagram qualifies for 2-Column Side-By-Side Layout
	if len(subgraphs) == 2 && len(subgraphNodeMap[subgraphs[0]]) > 0 && len(subgraphNodeMap[subgraphs[1]]) > 0 {
		g.renderTwoColumnFlowchart(subgraphs[0], subgraphNodeMap[subgraphs[0]], subgraphs[1], subgraphNodeMap[subgraphs[1]], fullCaption)
		return
	}

	// Single Column Centered Layout
	stepY := 9.2
	nodeH := 6.8
	if numNodes > 9 {
		// Compact mode for taller flowcharts
		stepY = 7.8
		nodeH = 5.8
	}

	boxHeight := float64(numNodes)*stepY + 12.0
	if boxHeight > 200 {
		boxHeight = 200
	}

	// Check page break: NEVER overshoot the 265mm printable baseline
	if pdf.GetY()+boxHeight+8 > 265 {
		pdf.AddPage()
	}

	yStart := pdf.GetY() + 2

	// Outer Frame
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.3)
	pdf.Rect(18, yStart, 174, boxHeight, "DF")

	// Header Bar
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(18, yStart, 174, 6.5, "DF")
	pdf.SetXY(22, yStart+1.2)
	pdf.SetFont("Helvetica", "B", 7.5)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(166, 4, fullCaption, "", 0, "L", false, 0, "")

	// Render node cards
	nodeW := 154.0
	curY := yStart + 8.5

	for idx, n := range orderedNodes {
		if curY+nodeH > yStart+boxHeight {
			break
		}

		// Node Box
		pdf.SetFillColor(241, 245, 249)
		pdf.SetDrawColor(100, 116, 139)
		pdf.SetLineWidth(0.2)
		pdf.RoundedRect(28, curY, nodeW, nodeH, 1.2, "1234", "DF")

		// Node Text (strictly measured to never spill horizontally)
		pdf.SetFont("Helvetica", "B", 7.0)
		pdf.SetTextColor(15, 23, 42)
		cleanLbl := sanitizeForPDF(n.Label)
		maxLblW := nodeW - 8.0
		if pdf.GetStringWidth(cleanLbl) > maxLblW {
			for len(cleanLbl) > 4 && pdf.GetStringWidth(cleanLbl+"...") > maxLblW {
				cleanLbl = cleanLbl[:len(cleanLbl)-1]
			}
			cleanLbl += "..."
		}
		pdf.SetXY(28, curY+(nodeH-4.5)/2.0)
		pdf.CellFormat(nodeW, 4.5, cleanLbl, "", 0, "C", false, 0, "")

		// Downward arrow
		if idx < len(orderedNodes)-1 && curY+stepY+nodeH <= yStart+boxHeight {
			arrowY := curY + nodeH
			arrowLen := stepY - nodeH
			midX := 105.0
			pdf.SetDrawColor(71, 85, 105)
			pdf.SetLineWidth(0.4)
			pdf.Line(midX, arrowY, midX, arrowY+arrowLen)

			// Arrowhead
			pdf.Line(midX-1.5, arrowY+arrowLen-1.0, midX, arrowY+arrowLen)
			pdf.Line(midX+1.5, arrowY+arrowLen-1.0, midX, arrowY+arrowLen)
		}

		curY += stepY
	}

	pdf.SetY(yStart + boxHeight + 6)
}

func (g *AcademicPDFGenerator) renderTwoColumnFlowchart(name1 string, nodes1 []FlowchartNode, name2 string, nodes2 []FlowchartNode, fullCaption string) {
	pdf := g.pdf

	maxCount := len(nodes1)
	if len(nodes2) > maxCount {
		maxCount = len(nodes2)
	}

	stepY := 8.5
	nodeH := 6.2
	boxHeight := float64(maxCount)*stepY + 19.0
	if boxHeight > 200 {
		boxHeight = 200
	}

	if pdf.GetY()+boxHeight+8 > 265 {
		pdf.AddPage()
	}

	yStart := pdf.GetY() + 2

	// Outer frame
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.3)
	pdf.Rect(18, yStart, 174, boxHeight, "DF")

	// Header Caption
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(18, yStart, 174, 6.5, "DF")
	pdf.SetXY(22, yStart+1.2)
	pdf.SetFont("Helvetica", "B", 7.5)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(166, 4, fullCaption, "", 0, "L", false, 0, "")

	colW := 78.0
	col1X := 22.0
	col2X := 105.0
	innerNodeW := 74.0

	// Column 1 Header Pill
	pdf.SetFillColor(241, 245, 249)
	pdf.SetDrawColor(148, 163, 184)
	pdf.SetLineWidth(0.2)
	pdf.RoundedRect(col1X, yStart+8.0, colW, 5.0, 1.0, "1234", "DF")
	pdf.SetXY(col1X, yStart+8.5)
	pdf.SetFont("Helvetica", "B", 6.8)
	pdf.SetTextColor(71, 85, 105)
	cleanName1 := sanitizeForPDF(name1)
	if pdf.GetStringWidth(cleanName1) > colW-4 {
		cleanName1 = cleanName1[:25] + "..."
	}
	pdf.CellFormat(colW, 4.0, cleanName1, "", 0, "C", false, 0, "")

	// Column 2 Header Pill
	pdf.RoundedRect(col2X, yStart+8.0, colW, 5.0, 1.0, "1234", "DF")
	pdf.SetXY(col2X, yStart+8.5)
	pdf.SetFont("Helvetica", "B", 6.8)
	pdf.SetTextColor(71, 85, 105)
	cleanName2 := sanitizeForPDF(name2)
	if pdf.GetStringWidth(cleanName2) > colW-4 {
		cleanName2 = cleanName2[:25] + "..."
	}
	pdf.CellFormat(colW, 4.0, cleanName2, "", 0, "C", false, 0, "")

	renderCol := func(startX float64, nodes []FlowchartNode) {
		curY := yStart + 15.0
		nodeX := startX + (colW-innerNodeW)/2.0
		midX := nodeX + innerNodeW/2.0

		for idx, n := range nodes {
			if curY+nodeH > yStart+boxHeight {
				break
			}

			pdf.SetFillColor(248, 250, 252)
			pdf.SetDrawColor(100, 116, 139)
			pdf.SetLineWidth(0.2)
			pdf.RoundedRect(nodeX, curY, innerNodeW, nodeH, 1.0, "1234", "DF")

			pdf.SetFont("Helvetica", "B", 6.5)
			pdf.SetTextColor(15, 23, 42)
			cleanLbl := sanitizeForPDF(n.Label)
			maxLblW := innerNodeW - 6.0
			if pdf.GetStringWidth(cleanLbl) > maxLblW {
				for len(cleanLbl) > 4 && pdf.GetStringWidth(cleanLbl+"...") > maxLblW {
					cleanLbl = cleanLbl[:len(cleanLbl)-1]
				}
				cleanLbl += "..."
			}
			pdf.SetXY(nodeX, curY+(nodeH-4.0)/2.0)
			pdf.CellFormat(innerNodeW, 4.0, cleanLbl, "", 0, "C", false, 0, "")

			// Downward arrow
			if idx < len(nodes)-1 && curY+stepY+nodeH <= yStart+boxHeight {
				arrowY := curY + nodeH
				arrowLen := stepY - nodeH
				pdf.SetDrawColor(71, 85, 105)
				pdf.SetLineWidth(0.4)
				pdf.Line(midX, arrowY, midX, arrowY+arrowLen)
				pdf.Line(midX-1.2, arrowY+arrowLen-0.8, midX, arrowY+arrowLen)
				pdf.Line(midX+1.2, arrowY+arrowLen-0.8, midX, arrowY+arrowLen)
			}

			curY += stepY
		}
	}

	renderCol(col1X, nodes1)
	renderCol(col2X, nodes2)

	pdf.SetY(yStart + boxHeight + 6)
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
			maxCellW := colW - 3.0
			if pdf.GetStringWidth(val) > maxCellW {
				for len(val) > 4 && pdf.GetStringWidth(val+"...") > maxCellW {
					val = val[:len(val)-1]
				}
				val += "..."
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
