package main

import (
	"fmt"
	"net/http"
	"path/filepath"
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
	router.mux.GET("/login", makeRouterHandleFunc(getLoginHandle))
	router.mux.GET("/signup", makeRouterHandleFunc(getSignupHandle))
	router.mux.GET("/write", makeRouterHandleFunc(getWriteBlogHandle))

	router.mux.POST("/login", makeRouterHandleFunc(postLoginHandle))
	router.mux.POST("/signup", makeRouterHandleFunc(postSignupHandle))
	router.mux.POST("/write", makeRouterHandleFunc(postWriteBlogHandle))

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

	id := params.ByName("id")

	// TODO: Replace with original DB
	blog, err := getSampleBlogById(ObjectID(id))
	if err != nil {
		return err
	}

	// TODO: Move to controller
	path := fmt.Sprintf("blogs/%s.md", blog.Id)
	blogContent := ReadFile(path)
	blogContentHtml := MdToHTML(blogContent)

	blogWithContent := new(BlogWithContent)
	blogWithContent.Blog = blog
	blogWithContent.Content = string(blogContentHtml)

	return templ.ExecuteTemplate(res, "layout", blogWithContent)
}

func getLoginHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/login.html")
	if err != nil {
		return err
	}
	return templ.ExecuteTemplate(res, "layout", nil)
}

func postLoginHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	UserLoginRequest := UserLoginRequest{
		Email:             req.FormValue("email"),
		EncryptedPassword: req.FormValue("password"),
	}

	LogInfo().Println("Details got: ", UserLoginRequest)
	return nil
}

func getSignupHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/signup.html")
	if err != nil {
		return err
	}
	return templ.ExecuteTemplate(res, "layout", nil)
}

func postSignupHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	userSignupRequest := UserSignupRequest{
		Name:              req.FormValue("name"),
		Email:             req.FormValue("email"),
		DateOfBirth:       req.FormValue("dob"),
		EncryptedPassword: req.FormValue("password"),
	}

	LogInfo().Println("Details got: ", userSignupRequest)
	return nil
}

func getWriteBlogHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/writeBlog.html")
	if err != nil {
		return err
	}
	return templ.ExecuteTemplate(res, "layout", nil)
}

func postWriteBlogHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	blogTitle := req.FormValue("title")
	blogContent := req.FormValue("content")

	path := filepath.Join("blogs", blogTitle+".md")
	return WriteToFile(path, blogContent)
}
