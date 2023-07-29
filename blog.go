package main

type Blog struct {
	Id            int64  `json:"id"`
	Author        *User  `json:"author"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	CoverImageURL string `json:"coverImageURL"`
	Premium       bool   `json:"premium"`
}

type Blogs struct {
	Length int    `json:"length"`
	Array  []Blog `json:"array"`
}
