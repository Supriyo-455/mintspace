package main

type Blog struct {
	Id            int64  `json:"_id"`
	Author        *User  `json:"author"`
	Title         string `json:"title"`
	CoverImageURL string `json:"coverImageURL"`
	Premium       bool   `json:"premium"`
	DateCreated   string `json:"datecreated"`
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
