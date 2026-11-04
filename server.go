package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

type routerFunc func(res http.ResponseWriter, req *http.Request, params httprouter.Params) error

func makeRouterHandleFunc(f routerFunc) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if err := f(res, req, params); err != nil {
			LogError().Println("Handler error:", err.Error())

			tmpl, parseErr := parseTemplate("templates/layout.html", "templates/error.html")
			if parseErr != nil {
				http.Error(res, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
				return
			}

			user := getSessionUser(req)
			data := TemplateData{
				Title:   "Error // MintSpace",
				Content: err.Error(),
				User:    user,
			}
			res.WriteHeader(http.StatusBadRequest)
			_ = tmpl.ExecuteTemplate(res, "layout", data)
		}
	}
}

type Server struct {
	mux          *httprouter.Router
	storage      Storage
	blobStorage  BlobStorage
	authLimiter  *IPRateLimiter
	writeLimiter *IPRateLimiter
}

func NewServer() *Server {
	server := new(Server)
	server.storage = NewSqliteStorage("data/mintspace.db")
	localStore, err := NewLocalCASStorage("data/blobs")
	if err != nil {
		LogError().Fatalln("Failed to init CAS storage:", err)
	}
	server.blobStorage = NewDistributedCASStorage(localStore, nil)
	server.authLimiter = NewIPRateLimiter(2, 10)  // 2 tokens/sec, burst 10
	server.writeLimiter = NewIPRateLimiter(3, 15) // 3 tokens/sec, burst 15
	server.RegisterRoutes()
	return server
}

func (server *Server) RegisterRoutes() {
	server.mux = httprouter.New()

	// Serve Static Files
	server.mux.ServeFiles("/static/*filepath", http.Dir("static"))

	// Health & Metrics (Load Balancers & Orchestration)
	server.mux.GET("/healthz", server.handleHealthz)

	// Distributed CAS Mesh Sync Endpoint
	server.mux.POST("/api/v1/sync/blob", server.handleSyncBlob)

	// Public Routes
	server.mux.GET("/", makeRouterHandleFunc(server.getIndexHandle))
	server.mux.GET("/blog", makeRouterHandleFunc(server.getBlogsHandle))
	server.mux.GET("/blog/:id", makeRouterHandleFunc(server.getBlogByIdHandle))
	server.mux.GET("/blog/:id/pdf", makeRouterHandleFunc(server.getPdfBlogHandle))
	server.mux.GET("/blob/:hash", makeRouterHandleFunc(server.getRawBlobHandle))
	server.mux.GET("/login", makeRouterHandleFunc(server.getLoginHandle))
	server.mux.POST("/login", withRateLimit(server.authLimiter, makeRouterHandleFunc(server.postLoginHandle)))
	server.mux.GET("/signup", makeRouterHandleFunc(server.getSignupHandle))
	server.mux.POST("/signup", withRateLimit(server.authLimiter, makeRouterHandleFunc(server.postSignupHandle)))
	server.mux.GET("/logout", makeRouterHandleFunc(server.getLogoutHandle))

	// Authenticated Routes
	server.mux.GET("/write", withAuthMiddleware(makeRouterHandleFunc(server.getWriteBlogHandle)))
	server.mux.POST("/write", withAuthMiddleware(withRateLimit(server.writeLimiter, makeRouterHandleFunc(server.postWriteBlogHandle))))
	server.mux.POST("/blog/:id/like", withAuthMiddleware(makeRouterHandleFunc(server.postToggleLikeHandle)))
	server.mux.POST("/blog/:id/comment", withAuthMiddleware(withRateLimit(server.writeLimiter, makeRouterHandleFunc(server.postCommentHandle))))
	server.mux.POST("/comment/:id/like", withAuthMiddleware(makeRouterHandleFunc(server.postToggleCommentLikeHandle)))
	server.mux.POST("/blog/:id/delete", withAuthMiddleware(makeRouterHandleFunc(server.postDeleteBlogHandle)))
	server.mux.POST("/comment/:id/delete", withAuthMiddleware(makeRouterHandleFunc(server.postDeleteCommentHandle)))
	server.mux.GET("/profile", withAuthMiddleware(makeRouterHandleFunc(server.getProfileHandle)))

	// Admin Routes
	server.mux.GET("/admin", withAdminMiddleware(makeRouterHandleFunc(server.getAdminHandle)))
	server.mux.POST("/admin/user/toggle", withAdminMiddleware(makeRouterHandleFunc(server.postToggleAdminHandle)))
	server.mux.POST("/admin/blog/delete", withAdminMiddleware(makeRouterHandleFunc(server.postAdminDeleteBlogHandle)))

	server.mux.NotFound = http.HandlerFunc(server.handle404)
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Enforce max request size to prevent DoS (10MB limit)
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		}

		next.ServeHTTP(w, r)
	})
}

func (server *Server) handleHealthz(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	res.Header().Set("Content-Type", "application/json")
	var peers []string
	if server.blobStorage != nil {
		peers = server.blobStorage.GetPeers()
	}
	_ = json.NewEncoder(res).Encode(map[string]interface{}{
		"status":  "healthy",
		"mode":    config.Mode,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"storage": "sqlite+cas-mesh",
		"peers":   peers,
	})
}

func (server *Server) handleSyncBlob(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	req.Body = http.MaxBytesReader(res, req.Body, 20<<20)
	data, err := io.ReadAll(req.Body)
	if err != nil || len(data) == 0 {
		http.Error(res, "bad request", http.StatusBadRequest)
		return
	}

	hash, err := server.blobStorage.Save(data)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(res).Encode(map[string]string{
		"status": "replicated",
		"hash":   hash,
	})
}

func (server *Server) Run() {
	if err := server.storage.Connect(); err != nil {
		LogError().Fatalln("Database connection failed:", err)
	}
	defer server.storage.Disconnect()

	// Seed demo paper if no papers exist yet
	server.seedDemoPaperIfEmpty()

	httpServer := http.Server{
		Addr:         config.Address,
		Handler:      securityHeadersMiddleware(server.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	LogInfo().Printf("MintSpace Academic Server running on http://%s\n", config.Address)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		LogError().Fatalln(err)
	}
}

func (server *Server) handle404(res http.ResponseWriter, req *http.Request) {
	tmpl, err := parseTemplate("templates/layout.html", "templates/404.html")
	if err != nil {
		http.NotFound(res, req)
		return
	}
	user := getSessionUser(req)
	data := MakeTemplateData("404 Not Found // MintSpace", nil, user)
	res.WriteHeader(http.StatusNotFound)
	_ = tmpl.ExecuteTemplate(res, "layout", data)
}

func (server *Server) getIndexHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	http.Redirect(res, req, "/blog", http.StatusSeeOther)
	return nil
}

// ----------------- BLOG HANDLERS -----------------

func (server *Server) getBlogsHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	user := getSessionUser(req)

	viewerEmail := ""
	isAdmin := false
	if user != nil {
		viewerEmail = user.Email
		isAdmin = user.IsAdmin
	}

	blogs, err := server.storage.GetAllBlogs(viewerEmail, isAdmin)
	if err != nil {
		return err
	}

	tmpl, err := parseTemplate("templates/layout.html", "templates/bloglist.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData("Archive of Research Papers // MintSpace", blogs, user))
}

func (server *Server) getBlogByIdHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid paper ID")
	}

	blog, err := server.storage.GetBlogById(id)
	if err != nil {
		return err
	}

	user := getSessionUser(req)

	// Authorization check for private blogs
	if blog.IsPrivate {
		if user == nil || (!user.IsAdmin && user.Email != blog.AuthorEmail) {
			return fmt.Errorf("restricted access: This paper is marked private by the author")
		}
	}

	// Fetch markdown content from content-addressable storage
	rawMd, err := server.blobStorage.Get(blog.ContentHash)
	if err != nil {
		return fmt.Errorf("unable to load paper body from CAS (hash: %s): %w", blog.ContentHash, err)
	}

	viewerEmail := ""
	if user != nil {
		viewerEmail = user.Email
		hasLiked, _ := server.storage.HasUserLiked(blog.Id, user.Email)
		blog.UserHasLiked = hasLiked
	}

	// Load comments
	comments, err := server.storage.GetCommentsByBlogId(blog.Id, viewerEmail)
	if err != nil {
		return err
	}

	renderedHTML := template.HTML(MdToHTML(rawMd))

	viewData := &BlogDetailView{
		Blog:           blog,
		RenderedHTML:   renderedHTML,
		Comments:       comments,
		CurrentSession: user,
	}

	tmpl, err := parseTemplate("templates/layout.html", "templates/blog.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData(blog.Title+" // MintSpace", viewData, user))
}

func (server *Server) getRawBlobHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	hash := params.ByName("hash")
	if !IsValidHash(hash) {
		http.Error(res, "Security violation: invalid content hash format", http.StatusBadRequest)
		return nil
	}
	rawBytes, err := server.blobStorage.Get(hash)
	if err != nil {
		http.Error(res, "Blob not found", http.StatusNotFound)
		return nil
	}
	res.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = res.Write(rawBytes)
	return nil
}

func (server *Server) getPdfBlogHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid paper ID")
	}

	blog, err := server.storage.GetBlogById(id)
	if err != nil {
		return err
	}

	user := getSessionUser(req)

	// Authorization check for private blogs
	if blog.IsPrivate {
		if user == nil || (!user.IsAdmin && user.Email != blog.AuthorEmail) {
			return fmt.Errorf("restricted access: This paper is marked private by the author")
		}
	}

	rawBytes, err := server.blobStorage.Get(blog.ContentHash)
	if err != nil {
		return fmt.Errorf("failed to fetch paper content: %w", err)
	}

	generator := NewAcademicPDFGenerator(blog, string(rawBytes))
	pdfBytes, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate academic PDF: %w", err)
	}

	// Clean filename for HTTP Header
	sanitizedTitle := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ' {
			return r
		}
		return '_'
	}, blog.Title)
	sanitizedTitle = strings.TrimSpace(sanitizedTitle)
	if sanitizedTitle == "" {
		sanitizedTitle = fmt.Sprintf("MintSpace-Paper-%d", blog.Id)
	}
	filename := fmt.Sprintf("%s.pdf", sanitizedTitle)

	res.Header().Set("Content-Type", "application/pdf")
	res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	res.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	_, _ = res.Write(pdfBytes)
	return nil
}

func (server *Server) getWriteBlogHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	tmpl, err := parseTemplate("templates/layout.html", "templates/writeBlog.html")
	if err != nil {
		return err
	}
	user := getSessionUser(req)
	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData("Submit Research Paper // MintSpace", nil, user))
}

func (server *Server) postWriteBlogHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		return fmt.Errorf("unauthorized")
	}

	var reqBody BlogCreateRequest
	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &reqBody); err != nil {
			return err
		}
	} else {
		// Multipart or standard form submission
		reqBody.Title = strings.TrimSpace(req.FormValue("title"))
		reqBody.Abstract = strings.TrimSpace(req.FormValue("abstract"))
		reqBody.ImageURL = strings.TrimSpace(req.FormValue("imageurl"))
		reqBody.Content = strings.TrimSpace(req.FormValue("content"))
		reqBody.IsPrivate = req.FormValue("is_private") == "on" || req.FormValue("is_private") == "true"

		// Check for file upload of markdown
		file, _, fileErr := req.FormFile("markdown_file")
		if fileErr == nil {
			fileBytes, readErr := io.ReadAll(file)
			if readErr == nil && len(fileBytes) > 0 {
				reqBody.Content = string(fileBytes)
			}
			file.Close()
		}
	}

	if reqBody.Title == "" {
		return fmt.Errorf("paper title is required")
	}
	if reqBody.Content == "" {
		return fmt.Errorf("paper content cannot be empty")
	}

	// 1. Content-Addressable Storage (CAS): Hash the markdown payload and store in blob storage
	contentHash, err := server.blobStorage.Save([]byte(reqBody.Content))
	if err != nil {
		return fmt.Errorf("failed to save content to CAS: %w", err)
	}

	// 2. Persist metadata in database with pointer to content hash
	blog := Blog{
		AuthorEmail:   user.Email,
		Title:         reqBody.Title,
		Abstract:      reqBody.Abstract,
		CoverImageURL: reqBody.ImageURL,
		ContentHash:   contentHash,
		IsPrivate:     reqBody.IsPrivate,
		DateCreated:   time.Now().Format("2006-01-02"),
	}

	id, err := server.storage.CreateBlog(&blog)
	if err != nil {
		return fmt.Errorf("failed to save paper record: %w", err)
	}

	LogInfo().Printf("Created paper #%d with content hash %s\n", id, contentHash)

	if strings.Contains(contentType, "application/json") {
		res.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(res).Encode(map[string]interface{}{
			"id":          id,
			"contentHash": contentHash,
			"url":         fmt.Sprintf("/blog/%d", id),
		})
		return nil
	}

	http.Redirect(res, req, fmt.Sprintf("/blog/%d", id), http.StatusSeeOther)
	return nil
}

func (server *Server) postToggleLikeHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		http.Error(res, "Authentication required", http.StatusUnauthorized)
		return nil
	}

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid paper ID")
	}

	hasLiked, count, err := server.storage.ToggleLike(id, user.Email)
	if err != nil {
		return err
	}

	res.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(res).Encode(map[string]interface{}{
		"hasLiked": hasLiked,
		"count":    count,
	})
}

func (server *Server) postCommentHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		return fmt.Errorf("authentication required to participate in review discussion")
	}

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid paper ID")
	}

	content := strings.TrimSpace(req.FormValue("comment"))
	if content == "" {
		return fmt.Errorf("comment content cannot be empty")
	}

	parentId, _ := strconv.ParseInt(req.FormValue("parent_id"), 10, 64)

	authorName := user.Name
	if authorName == "" {
		authorName = user.Email
	}

	comment := Comment{
		BlogId:      id,
		ParentId:    parentId,
		UserEmail:   user.Email,
		AuthorName:  authorName,
		Content:     content,
		DateCreated: time.Now().Format("2006-01-02 15:04"),
	}

	newId, err := server.storage.CreateComment(&comment)
	if err != nil {
		return err
	}

	http.Redirect(res, req, fmt.Sprintf("/blog/%d#comment-%d", id, newId), http.StatusSeeOther)
	return nil
}

func (server *Server) postToggleCommentLikeHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		http.Error(res, "Authentication required", http.StatusUnauthorized)
		return nil
	}

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid comment ID")
	}

	hasLiked, count, err := server.storage.ToggleCommentLike(id, user.Email)
	if err != nil {
		return err
	}

	res.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(res).Encode(map[string]interface{}{
		"hasLiked": hasLiked,
		"count":    count,
	})
}

func (server *Server) postDeleteBlogHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		return fmt.Errorf("unauthorized")
	}

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid paper ID")
	}

	blog, err := server.storage.GetBlogById(id)
	if err != nil {
		return err
	}

	if !user.IsAdmin && user.Email != blog.AuthorEmail {
		return fmt.Errorf("permission denied: you cannot delete this paper")
	}

	if err := server.storage.DeleteBlog(id); err != nil {
		return err
	}

	http.Redirect(res, req, "/blog", http.StatusSeeOther)
	return nil
}

func (server *Server) postDeleteCommentHandle(res http.ResponseWriter, req *http.Request, params httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		return fmt.Errorf("unauthorized")
	}

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid comment ID")
	}

	comment, err := server.storage.GetCommentById(id)
	if err != nil {
		return err
	}

	if !user.IsAdmin && user.Email != comment.UserEmail {
		return fmt.Errorf("permission denied: you cannot delete this comment")
	}

	if err := server.storage.DeleteComment(id); err != nil {
		return err
	}

	referer := req.Header.Get("Referer")
	if referer == "" {
		referer = fmt.Sprintf("/blog/%d", comment.BlogId)
	}
	http.Redirect(res, req, referer, http.StatusSeeOther)
	return nil
}

// ----------------- AUTH HANDLERS -----------------

func (server *Server) getLoginHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	if user := getSessionUser(req); user != nil {
		http.Redirect(res, req, "/blog", http.StatusSeeOther)
		return nil
	}

	tmpl, err := parseTemplate("templates/layout.html", "templates/login.html")
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData("Sign In - MintSpace", nil, nil))
}

func (server *Server) postLoginHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	email := strings.TrimSpace(req.FormValue("email"))
	password := strings.TrimSpace(req.FormValue("password"))

	if email == "" || password == "" {
		return fmt.Errorf("both email and password are required")
	}

	user, err := server.storage.GetUserByEmail(email)
	if err != nil || !CheckPasswordHash(password, user.EncryptedPassword) {
		return fmt.Errorf("invalid credentials: email or password incorrect")
	}

	token, err := createJWT(user.Email, user.Name, user.Admin)
	if err != nil {
		return err
	}

	http.SetCookie(res, &http.Cookie{
		Name:     "Auth",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24 * 30),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(res, req, "/blog", http.StatusSeeOther)
	return nil
}

func (server *Server) getSignupHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	if user := getSessionUser(req); user != nil {
		http.Redirect(res, req, "/blog", http.StatusSeeOther)
		return nil
	}

	tmpl, err := parseTemplate("templates/layout.html", "templates/signup.html")
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData("Register - MintSpace", nil, nil))
}

func (server *Server) postSignupHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	name := strings.TrimSpace(req.FormValue("name"))
	email := strings.TrimSpace(req.FormValue("email"))
	password := strings.TrimSpace(req.FormValue("password"))
	dob := strings.TrimSpace(req.FormValue("dob"))

	if name == "" || email == "" || password == "" {
		return fmt.Errorf("all fields are required")
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	user := User{
		Name:              name,
		Email:             email,
		EncryptedPassword: hashedPassword,
		Admin:             false,
		DateOfBirth:       dob,
		DateCreated:       time.Now().Format("2006-01-02"),
	}

	if err := server.storage.CreateUser(&user); err != nil {
		return fmt.Errorf("account creation failed (email may already be registered): %w", err)
	}

	// Automatically log in after registration
	token, _ := createJWT(user.Email, user.Name, user.Admin)
	http.SetCookie(res, &http.Cookie{
		Name:     "Auth",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24 * 30),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(res, req, "/blog", http.StatusSeeOther)
	return nil
}

func (server *Server) getLogoutHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	http.SetCookie(res, &http.Cookie{
		Name:     "Auth",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(res, req, "/login", http.StatusSeeOther)
	return nil
}

func (server *Server) getProfileHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	user := getSessionUser(req)
	if user == nil {
		http.Redirect(res, req, "/login", http.StatusSeeOther)
		return nil
	}

	fullUser, err := server.storage.GetUserByEmail(user.Email)
	if err != nil {
		return err
	}

	userBlogs, err := server.storage.GetBlogsByAuthor(user.Email)
	if err != nil {
		return err
	}

	type ProfileViewData struct {
		User  *User
		Blogs []Blog
	}

	tmpl, err := parseTemplate("templates/layout.html", "templates/profile.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData("Author Profile // MintSpace", ProfileViewData{
		User:  fullUser,
		Blogs: userBlogs,
	}, user))
}

// ----------------- ADMIN HANDLERS -----------------

func (server *Server) getAdminHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	user := getSessionUser(req)

	stats, err := server.storage.GetStats()
	if err != nil {
		return err
	}

	users, err := server.storage.GetAllUsers()
	if err != nil {
		return err
	}

	blogs, err := server.storage.GetAllBlogs("", true) // admin views all
	if err != nil {
		return err
	}

	type AdminDashboardData struct {
		Stats *Stats
		Users []User
		Blogs []Blog
	}

	tmpl, err := parseTemplate("templates/layout.html", "templates/admin.html")
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(res, "layout", MakeTemplateData("Editorial & System Console // MintSpace", AdminDashboardData{
		Stats: stats,
		Users: users,
		Blogs: blogs,
	}, user))
}

func (server *Server) postToggleAdminHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	targetEmail := req.FormValue("email")
	isAdmin := req.FormValue("is_admin") == "1"

	if targetEmail == "" {
		return fmt.Errorf("email required")
	}

	if err := server.storage.SetUserAdmin(targetEmail, isAdmin); err != nil {
		return err
	}

	http.Redirect(res, req, "/admin", http.StatusSeeOther)
	return nil
}

func (server *Server) postAdminDeleteBlogHandle(res http.ResponseWriter, req *http.Request, _ httprouter.Params) error {
	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID")
	}

	if err := server.storage.DeleteBlog(id); err != nil {
		return err
	}

	http.Redirect(res, req, "/admin", http.StatusSeeOther)
	return nil
}

// ----------------- SEED INITIAL SAMPLE DATA -----------------

func (server *Server) seedDemoPaperIfEmpty() {
	blogs, err := server.storage.GetAllBlogs("", true)
	if err == nil && len(blogs) == 0 {
		sampleMd := `# Content-Addressable Storage and Distributed Archiving in Modern Web Platforms

## Abstract
Traditional web publishing architectures tightly couple content storage to relational primary keys and mutable records. In this treatise, we present an alternative architectural paradigm built on **Content-Addressable Storage (CAS)**, cryptographic hashing ($\mathcal{H} : \{0,1\}^* \to \{0,1\}^{256}$), and zero-friction migration paths toward decentralized storage networks such as IPFS and Arweave.

---

## 1. Introduction & Theoretical Background

Modern scholarly communication demands transparency, immutability, and deterministic verification. When a manuscript is submitted, its integrity must remain verifiable across time and physical host boundaries:

$$\text{CID} = \text{Multihash}(\text{SHA-256}(M))$$

Where $M$ is the canonical Markdown payload. By separating the metadata tier (authorship, visibility controls, peer commentary) from the content payload, the storage layer achieves complete location independence.

### Key Architectural Tenets:
1. **Deduplication**: Identical blog entries generate congruent hashes, eliminating storage redundancy.
2. **Tamper Evidence**: Any alteration to a single character in the markdown file instantly invalidates the content hash pointer in the index.
3. **Decentralized Readiness**: The payload pointer can resolve locally or across peer-to-peer storage protocols seamlessly.

---

## 2. Mathematical Formalism

Let $P$ denote a published manuscript represented by the tuple:

$$P = \langle \mathcal{A}, \mathcal{T}, \mathcal{H}(M), \sigma_{\text{visibility}}, \tau_{\text{created}} \rangle$$

The verification predicate $\mathcal{V}$ is defined as:

$$\mathcal{V}(P, M) = \begin{cases} 
1 & \text{if } \text{SHA256}(M) = P_{\mathcal{H}} \\ 
0 & \text{otherwise} 
\end{cases}$$

---

## 3. Implementation Blueprint in Go

The underlying file storage abstraction decouples the persistence engine from physical disk layouts:

` + "```go" + `
type BlobStorage interface {
    Save(content []byte) (hash string, err error)
    Get(hash string) ([]byte, error)
    Delete(hash string) error
}
` + "```" + `

When distributed storage is integrated, the ` + "`LocalCASStorage`" + ` adapter is substituted with an IPFS or Arweave gateway adapter without requiring schema or API alterations.

---

## 4. Conclusion & Outlook

Content-addressable storage offers mathematical guarantees against unauthorized content modification while providing a natural bridge to decentralized web infrastructure. Future work will investigate zero-knowledge proof verification for access-restricted private manuscripts.

---

### References
1. Benet, J. (2014). *IPFS - Content Addressed, Versioned, P2P File System*. arXiv:1407.3561.
2. Merkle, R. C. (1987). *A Digital Signature Based on a Conventional Encryption Function*. Advances in Cryptology — CRYPTO ’87.
`
		hash, err := server.blobStorage.Save([]byte(sampleMd))
		if err == nil {
			demoBlog := &Blog{
				AuthorEmail:   "admin@mintspace.io",
				Title:         "Content-Addressable Storage and Distributed Archiving in Modern Web Platforms",
				Abstract:      "An architectural paper detailing cryptographic content-addressable storage (CAS), SHA-256 integrity verification, and zero-downtime distributed storage integration for scholarly and technical publishing.",
				CoverImageURL: "",
				ContentHash:   hash,
				IsPrivate:     false,
				DateCreated:   time.Now().Format("2006-01-02"),
			}
			_, _ = server.storage.CreateBlog(demoBlog)
		}
	}
}
