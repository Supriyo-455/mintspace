package main

type TemplateData struct {
	Title   string
	Content interface{}
}

func MakeTemplateData(title string, content interface{}) TemplateData {
	data := TemplateData{
		Title:   title,
		Content: content,
	}
	return data
}
