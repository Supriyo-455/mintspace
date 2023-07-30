package main

import (
	"encoding/json"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func WriteToFile(path, data string) error {
	err := os.WriteFile(path, []byte(data), 0666)
	return err
}

func ReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		LogError().Fatalln(err.Error())
	}
	return data
}

func FileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		LogError().Fatalln(err.Error())
	}
	return true
}

func LoadJson(filename string, key interface{}) {
	inFile, err := os.Open(filename)
	if err != nil {
		LogError().Fatalln(err.Error())
	}

	decoder := json.NewDecoder(inFile)

	err = decoder.Decode(key)
	if err != nil {
		LogError().Fatalln(err.Error())
	}

	inFile.Close()
}

func MdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create html renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}
