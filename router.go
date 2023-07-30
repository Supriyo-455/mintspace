package main

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

type routerFunc func(res http.ResponseWriter, req *http.Request, params httprouter.Params) error

func makeRouterHandleFunc(f routerFunc) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if err := f(res, req, params); err != nil {
			fmt.Fprintln(res, "Error Occured!")
			LogError().Println(err.Error())
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

	router.mux.GET("/", makeRouterHandleFunc(home))
	router.mux.GET("/blog/", makeRouterHandleFunc(getBlogsHandle))
	router.mux.GET("/blog/:id", makeRouterHandleFunc(getBlogByIdHandle))

	return router
}

func home(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	fmt.Fprintln(res, "Hello world!")
	return nil
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

	// id, err := strconv.Atoi(params.ByName("id"))
	// if err != nil {
	// 	return err
	// }

	// blog, err := getSampleBlogById(id)
	// if err != nil {
	// 	return err
	// }

	blogContent := ReadFile("blogs/SampleBlog.md")
	blogContentHtml := MdToHTML(blogContent)

	fmt.Fprintln(res, string(blogContentHtml))
	return nil
}
