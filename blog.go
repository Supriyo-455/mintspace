package main

type Blog struct {
	Id            int    `json:"id"`
	Author        *User  `json:"author"`
	Title         string `json:"title"`
	CoverImageURL string `json:"coverImageURL"`
	Premium       bool   `json:"premium"`
}

type Blogs struct {
	Length int    `json:"length"`
	Array  []Blog `json:"array"`
}
