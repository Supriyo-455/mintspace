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
	temp, err := template.ParseFiles("templates/layout.html", "templates/404.html")
	if err != nil {
		LogError().Fatalln(err)
	}
	temp.ExecuteTemplate(res, "layout", nil)
}

func getBlogsHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	// TODO: Replace with original DB
	blogs := getSampleBlogs()
	templ, err := template.ParseFiles("templates/layout.html", "templates/bloglist.html")
	if err != nil {
		return err
	}

	return templ.ExecuteTemplate(res, "layout", blogs.Array)
}

// TODO: Handle XSS attack
func getBlogByIdHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/blog.html", "templates/blogcontent.html")
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		return err
	}

	// TODO: Replace with original DB
	blog, err := getSampleBlogById(id)
	if err != nil {
		return err
	}

	// TODO: Move to controller
	path := fmt.Sprintf("blogs/%s.md", (blog.Id).(string))
	blogContent := ReadFile(path)
	blogContentHtml := MdToHTML(blogContent)

	type BlogWithContent struct {
		Blog    *Blog
		Content string
	}

	blogWithContent := new(BlogWithContent)
	blogWithContent.Blog = blog
	blogWithContent.Content = string(blogContentHtml)

	return templ.ExecuteTemplate(res, "layout", blogWithContent)
}
