package main

/*
	Requirements:
		1) Insert User, Blog
		2) Update User, Blog
		3) Delete Blog
		4) Get user by email, user name, id
		5) Get blog by title, id, author
*/

type Storage interface {
	Connect() error
	Disconnect()
	CheckUserTable() error
	CreateUser(user *User) error
	DeleteUser(email string) error
	GetUserByEmail(email string) (*User, error)
	GetAllBlogs() (*Blogs, error)
	CreateBlog(blog *Blog) (int64, error)
	DeleteBlog(id int64) error
	GetBlogById(id int64) (*Blog, error)
	// GetBlogsByTitle(title string) (Blogs, error)
	// GetBlogsByAuthorName(authorName string) (Blogs, error)
}
