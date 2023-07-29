package main

import (
	"net/http"
)

func main() {
	print(config.Address, config.Static)

	TestMongo(ConnectToMongo())

	router := NewRouter()

	server := http.Server{
		Addr:    config.Address,
		Handler: router.mux,
	}

	Info().Println("Server started with address: ", config.Address)
	err := server.ListenAndServe()
	if err != nil {
		Error().Fatalln(err)
	}
}
