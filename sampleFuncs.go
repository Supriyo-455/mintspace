package main

func getSampleBlogs() Blogs {
	filename := "sampledb/sampleblogs.json"
	blogs := &Blogs{}
	LoadJson(filename, blogs)
	return *blogs
}
