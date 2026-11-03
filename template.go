package main

import (
	"fmt"
	"html/template"
	"path/filepath"
)

type TemplateData struct {
	Title   string
	Content interface{}
	User    *SessionUser
	Error   string
	Success string
}

func MakeTemplateData(title string, content interface{}, user *SessionUser) TemplateData {
	return TemplateData{
		Title:   title,
		Content: content,
		User:    user,
	}
}

func parseTemplate(files ...string) (*template.Template, error) {
	name := filepath.Base(files[0])
	return template.New(name).Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call: uneven number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFiles(files...)
}
