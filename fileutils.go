package main

import (
	"encoding/json"
	"os"
	"regexp"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

func WriteToFile(path, data string) error {
	return os.WriteFile(path, []byte(data), 0666)
}

func CreateFolder(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func ReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		LogError().Println("Error reading file:", err.Error())
		return nil
	}
	return data
}

func FileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func LoadJson(filename string, key interface{}) {
	inFile, err := os.Open(filename)
	if err != nil {
		LogError().Println("Warning: cannot open json file:", filename, err.Error())
		return
	}
	defer inFile.Close()

	decoder := json.NewDecoder(inFile)
	if err := decoder.Decode(key); err != nil {
		LogError().Println("Warning: failed to decode json:", filename, err.Error())
	}
}

// MdToHTML parses markdown to HTML and sanitizes it against XSS
func MdToHTML(md []byte) []byte {
	// Enable extensions: tables, footnotes, autolink, fenced code, heading IDs, math equations
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.MathJax | parser.Footnotes
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	rendered := markdown.Render(doc, renderer)

	// Sanitize output to prevent stored XSS while preserving legitimate styles and code blocks
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z0-9_\-\s]+$`)).Globally()
	policy.AllowAttrs("id").Matching(regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)).Globally()
	policy.AllowElements("span", "div", "pre", "code", "table", "thead", "tbody", "tr", "th", "td", "sup", "sub", "hr", "blockquote")
	policy.AllowAttrs("align").OnElements("th", "td")

	return policy.SanitizeBytes(rendered)
}
