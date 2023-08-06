package main

import (
	"net/http"
)

func main() {
	router := NewRouter()

	server := http.Server{
		Addr:    config.Address,
		Handler: router.mux,
	}

	print("Server config:", config)
	err := server.ListenAndServe()
	if err != nil {
		LogError().Fatalln(err)
	}
}
