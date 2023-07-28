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

	Info().Println("Server started with address: ", config.Address)
	server.ListenAndServe()
}
