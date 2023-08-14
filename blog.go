package main

type Blog struct {
	Id            ObjectID `bson:"_id"`
	Author        *User    `bson:"author"`
	Title         string   `bson:"title"`
	CoverImageURL string   `bson:"coverImageURL"`
	Premium       bool     `bson:"premium"`
	DateCreated   string   `bson:"datecreated"`
}

type Blogs struct {
	Length int    `bson:"length"`
	Array  []Blog `bson:"array"`
}

type BlogWithContent struct {
	Blog    *Blog
	Content string
}
