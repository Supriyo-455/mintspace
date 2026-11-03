package main

import "html/template"

type Blog struct {
	Id            int64  `json:"id"`
	AuthorEmail   string `json:"authorEmail"`
	AuthorName    string `json:"authorName"`
	Title         string `json:"title"`
	Abstract      string `json:"abstract"`
	CoverImageURL string `json:"coverImageURL"`
	ContentHash   string `json:"contentHash"`
	Content       string `json:"content,omitempty"`
	IsPrivate     bool   `json:"isPrivate"`
	LikesCount    int    `json:"likesCount"`
	CommentsCount int    `json:"commentsCount"`
	DateCreated   string `json:"dateCreated"`
	UserHasLiked  bool   `json:"userHasLiked"`
}

type Comment struct {
	Id           int64     `json:"id"`
	BlogId       int64     `json:"blogId"`
	ParentId     int64     `json:"parentId"`
	UserEmail    string    `json:"userEmail"`
	AuthorName   string    `json:"authorName"`
	Content      string    `json:"content"`
	DateCreated  string    `json:"dateCreated"`
	LikesCount   int       `json:"likesCount"`
	UserHasLiked bool      `json:"userHasLiked"`
	Replies      []Comment `json:"replies"`
}

type BlogCreateRequest struct {
	Title     string `json:"title"`
	Abstract  string `json:"abstract"`
	ImageURL  string `json:"imageurl"`
	Content   string `json:"content"`
	IsPrivate bool   `json:"isPrivate"`
}

type BlogDetailView struct {
	Blog           *Blog
	RenderedHTML   template.HTML
	Comments       []Comment
	CurrentSession *SessionUser
}

type Stats struct {
	TotalUsers    int
	TotalBlogs    int
	TotalPublic   int
	TotalPrivate  int
	TotalComments int
	TotalLikes    int
}
