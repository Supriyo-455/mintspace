package main

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type routerFunc func(res http.ResponseWriter, req *http.Request, params httprouter.Params) error

func makeRouterHandleFunc(f routerFunc) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if err := f(res, req, params); err != nil {
			LogError().Println(err.Error()) // Not so fatal error

			temp, err := template.ParseFiles("templates/404.html")
			if err != nil {
				LogError().Fatalln(err.Error()) // Fatal error for not finding 404.html
			}

			err = temp.Execute(res, nil)
			if err != nil {
				LogError().Fatalln(err.Error()) // Fatal error for not finding 404.html
			}
		}
	}
}

type Router struct {
	mux *httprouter.Router
}

func NewRouter() *Router {
	router := new(Router)
	router.mux = httprouter.New()

	files := http.FileServer(http.Dir(config.Static))
	router.mux.Handler("GET", "/static/", http.StripPrefix("/static/", files))

	router.mux.GET("/blog/", makeRouterHandleFunc(getBlogsHandle))
	router.mux.GET("/blog/:id", makeRouterHandleFunc(getBlogByIdHandle))

	router.mux.NotFound = http.HandlerFunc(handle404)

	return router
}

func handle404(res http.ResponseWriter, req *http.Request) {
	temp, err := template.ParseFiles("templates/404.html")
	if err != nil {
		LogError().Fatalln(err)
	}
	temp.Execute(res, nil)
}

func getBlogsHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	blogs := getSampleBlogs()
	templ, err := template.ParseFiles("templates/home.html")
	if err != nil {
		return err
	}

	return templ.Execute(res, blogs.Array)
}

// TODO: Error handleing of the params
func getBlogByIdHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	// templ, err := template.ParseFiles("templates/blogcontent.html")
	// if err != nil {
	// 	return err
	// }

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		return err
	}

	blog, err := getSampleBlogById(id)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("blogs/%d.md", blog.Id)
	blogContent := ReadFile(path)
	blogContentHtml := MdToHTML(blogContent)

	fmt.Fprintln(res, string(blogContentHtml))
	return nil
}
