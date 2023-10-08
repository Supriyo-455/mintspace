package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"
	"time"

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

type Server struct {
	mux     *httprouter.Router
	storage Storage
}

func NewServer() *Server {
	server := new(Server)
	server.mux = httprouter.New()
	server.storage = NewMySqlStorage()

	server.mux.GET("/blog", withJWTAuth(makeRouterHandleFunc(server.getBlogsHandle)))
	server.mux.GET("/blog/:id", withJWTAuth((makeRouterHandleFunc(server.getBlogByIdHandle))))
	server.mux.GET("/login", makeRouterHandleFunc(server.getLoginHandle))
	server.mux.GET("/signup", makeRouterHandleFunc(server.getSignupHandle))
	server.mux.GET("/write", withJWTAuth(makeRouterHandleFunc(server.getWriteBlogHandle)))
	server.mux.GET("/profile", withJWTAuth(makeRouterHandleFunc(server.getProfileHandle)))

	server.mux.POST("/login", makeRouterHandleFunc(server.postLoginHandle))
	server.mux.POST("/signup", makeRouterHandleFunc(server.postSignupHandle))
	server.mux.POST("/write", withJWTAuth(makeRouterHandleFunc(server.postWriteBlogHandle)))

	server.mux.NotFound = http.HandlerFunc(server.handle404)

	return server
}

func (server *Server) Run() {
	server.storage.Connect()

	httpServer := http.Server{
		Addr:    config.Address,
		Handler: server.mux,
	}

	print("Server config:", *config)
	err := httpServer.ListenAndServe()
	if err != nil {
		LogError().Fatalln(err)
	}

	defer server.storage.Disconnect()
}

func (server *Server) handle404(res http.ResponseWriter, req *http.Request) {
	temp, err := template.ParseFiles("templates/layout.html", "templates/404.html")
	if err != nil {
		LogError().Fatalln(err)
	}
	temp.ExecuteTemplate(res, "layout", MakeTemplateData("NotFound", nil))
}

func (server *Server) getProfileHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/profile.html")
	if err != nil {
		return err
	}

	cookie, err := req.Cookie("user")
	if err != nil {
		return err
	}

	user, err := server.storage.GetUserByEmail(cookie.Value)
	if err != nil {
		return err
	}

	return templ.ExecuteTemplate(res, "layout", MakeTemplateData("profile", user))
}

func (server *Server) getBlogsHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	blogs := make([]Blog, 0)
	templ, err := template.ParseFiles("templates/layout.html", "templates/bloglist.html")
	if err != nil {
		return err
	}

	return templ.ExecuteTemplate(res, "layout", MakeTemplateData("Blogs", blogs))
}

func (server *Server) getBlogByIdHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/blog.html", "templates/blogcontent.html")
	if err != nil {
		return err
	}

	blog, err := Blog{}, nil
	if err != nil {
		return err
	}

	path := fmt.Sprintf("blogs/%s.md", "nil")
	blogContent := ReadFile(path)
	blogContentHtml := MdToHTML(blogContent)

	blogWithContent := new(BlogWithContent)
	blogWithContent.Blog = &blog
	blogWithContent.Content = string(blogContentHtml)

	return templ.ExecuteTemplate(res, "layout", MakeTemplateData(blog.Title, blogWithContent))
}

func (server *Server) getLoginHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/login.html")
	if err != nil {
		return err
	}

	return templ.ExecuteTemplate(res, "layout", MakeTemplateData("login", nil))
}

func (server *Server) postLoginHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	userLoginRequest := UserLoginRequest{
		Email:    req.FormValue("email"),
		Password: req.FormValue("password"),
	}

	LogInfo().Println("Details got: ", userLoginRequest)

	entry, err := server.storage.GetUserByEmail(userLoginRequest.Email)
	if err != nil {
		return err
	}

	if !CheckPasswordHash(userLoginRequest.Password, entry.EncryptedPassword) {
		return fmt.Errorf("error occured! wrong password or email")
	}

	token, err := createJWT(&userLoginRequest)
	if err != nil {
		return err
	}

	cookie1 := http.Cookie{
		Name:    "Auth",
		Value:   token,
		Expires: time.Now().Add(time.Hour * 24 * 30),
	}

	cookie2 := http.Cookie{
		Name:    "user",
		Value:   userLoginRequest.Email,
		Expires: time.Now().Add(time.Hour * 24 * 30),
	}

	LogInfo().Println("User logged in!")

	http.SetCookie(res, &cookie1)
	http.SetCookie(res, &cookie2)

	http.Redirect(res, req, "/blog", http.StatusSeeOther)
	return nil
}

func (server *Server) getSignupHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/signup.html")
	if err != nil {
		return err
	}
	return templ.ExecuteTemplate(res, "layout", MakeTemplateData("signup", nil))
}

func (server *Server) postSignupHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	userSignupRequest := UserSignupRequest{
		Name:        req.FormValue("name"),
		Email:       req.FormValue("email"),
		DateOfBirth: req.FormValue("dob"),
		Password:    req.FormValue("password"),
	}

	hashPassword, err := HashPassword(userSignupRequest.Password)
	if err != nil {
		return err
	}

	user := User{
		Name:              userSignupRequest.Name,
		EncryptedPassword: hashPassword,
		Email:             userSignupRequest.Email,
		Admin:             false,
		DateOfBirth:       userSignupRequest.DateOfBirth,
		DateCreated:       time.Now().Format("2006-01-02"),
	}

	err = server.storage.CreateUser(&user)
	if err != nil {
		return err
	}

	http.Redirect(res, req, "/login", http.StatusSeeOther)
	return nil
}

func (server *Server) getWriteBlogHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	templ, err := template.ParseFiles("templates/layout.html", "templates/writeBlog.html")
	if err != nil {
		return err
	}
	return templ.ExecuteTemplate(res, "layout", MakeTemplateData("write blog", nil))
}

func (server *Server) postWriteBlogHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	var blogCreateRequest BlogCreateRequest

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &blogCreateRequest)
	if err != nil {
		return err
	}

	LogInfo().Println("Data received :", blogCreateRequest)
	return nil
}
