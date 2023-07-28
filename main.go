package main

import (
	"net/http"
)

func main() {
	print(config.Address, config.Static)

	router := NewRouter()

	server := http.Server{
		Addr:    config.Address,
		Handler: router.mux,
	}

	print("Listening on port: ", config.Address)
	server.ListenAndServe()
}
