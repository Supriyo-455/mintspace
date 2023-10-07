package main

type Blog struct {
	Id            int64  `json:"_id"`
	AuthorEmail   string `json:"authorEmail"`
	Title         string `json:"title"`
	CoverImageURL string `json:"coverImageURL"`
	Content       string `json:"content"`
	Premium       bool   `json:"premium"`
	DateCreated   string `json:"dateCreated"`
}

type Blogs struct {
	Length int    `json:"length"`
	Array  []Blog `json:"array"`
}

type BlogWithContent struct {
	Blog    *Blog
	Content string
}

type BlogCreateRequest struct {
	Title    string `json:"title"`
	ImageURL string `json:"imageurl"`
	Content  string `json:"content"`
}
