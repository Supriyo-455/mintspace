package main

import (
	"context"
)

/*
	Requirements:
		1) Insert User, Blog
		2) Update User, Blog
		3) Delete User, Blog
		4) Get all blogs
		5) Store blog content in "blogs/{userID}/{BlogID}.md" this manner
		6) When deleting the blog, blog content should also be deleted from the file system
		7) Get blogs by ID
*/

type Storage interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	InsertUser(ctx context.Context, user *User) (ObjectID, error)
	InsertBlog(ctx context.Context, blog *Blog) (string, error)
	GetAllBlogs(ctx context.Context) (Blogs, error)
	GetBlogByID(ctx context.Context, id ObjectID) (Blog, error)
	UpdateUserByID(ctx context.Context, id ObjectID, data *User) error
	UpdateBlogByID(ctx context.Context, id ObjectID, data *Blog) error
	DeleteBlogByID(ctx context.Context, id ObjectID) error
	DeleteUserByID(ctx context.Context, id ObjectID) error
}
