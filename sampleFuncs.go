package main

import "fmt"

func getSampleBlogs() Blogs {
	filename := "sampledb/sampleblogs.json"
	blogs := Blogs{}
	LoadJson(filename, &blogs)
	return blogs
}

func getSampleBlogById(id int) (*Blog, error) {
	blogs := getSampleBlogs()
	for _, blog := range blogs.Array {
		if blog.Id == id {
			return &blog, nil
		}
	}
	return nil, fmt.Errorf("blog with id %d not found", id)
}
