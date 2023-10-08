package main

import (
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		LogError().Fatal("Error loading .env file")
	}
}

func main() {
	server := NewServer()
	server.Run()
}
