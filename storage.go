package main

type Storage interface {
	Connect() error
	Disconnect()
	InitSchema() error

	// User methods
	CreateUser(user *User) error
	DeleteUser(email string) error
	GetUserByEmail(email string) (*User, error)
	GetAllUsers() ([]User, error)
	SetUserAdmin(email string, isAdmin bool) error

	// Blog methods
	CreateBlog(blog *Blog) (int64, error)
	DeleteBlog(id int64) error
	GetBlogById(id int64) (*Blog, error)
	GetAllBlogs(viewerEmail string, isAdmin bool) ([]Blog, error)
	GetBlogsByAuthor(authorEmail string) ([]Blog, error)

	// Like methods
	ToggleLike(blogId int64, userEmail string) (hasLiked bool, newCount int, err error)
	HasUserLiked(blogId int64, userEmail string) (bool, error)

	// Comment methods
	CreateComment(comment *Comment) (int64, error)
	GetCommentById(id int64) (*Comment, error)
	GetCommentsByBlogId(blogId int64, viewerEmail string) ([]Comment, error)
	DeleteComment(id int64) error
	ToggleCommentLike(commentId int64, userEmail string) (hasLiked bool, newCount int, err error)

	// Metrics
	GetStats() (*Stats, error)
}
