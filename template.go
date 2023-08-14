package main

type TemplateData struct {
	Title   string
	URL     string
	Content interface{}
}

func MakeTemplateData(title string, content interface{}) TemplateData {
	data := TemplateData{
		Title:   title,
		URL:     config.Address,
		Content: content,
	}
	return data
}
