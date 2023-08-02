package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
	Init(ctx *context.Context) error
	Close(ctx *context.Context) error
	InsertUser(ctx *context.Context, user *User) (primitive.ObjectID, error)
	InsertBlog(ctx *context.Context, blog *Blog) (primitive.ObjectID, error)
	GetAllBlogs(ctx *context.Context) ([]Blog, error)
	GetBlogByID(ctx *context.Context, id primitive.ObjectID) (Blog, error)
	UpdateUserByID(ctx *context.Context, id primitive.ObjectID, data *User) error
	UpdateBlogByID(ctx *context.Context, id primitive.ObjectID, data *Blog) error
	DeleteBlogByID(ctx *context.Context, id primitive.ObjectID) error
	DeleteUserByID(ctx *context.Context, id primitive.ObjectID) error
}
