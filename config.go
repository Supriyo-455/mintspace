package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var config Configuration

type Configuration struct {
	Address string
	Static  string
}

// Convenience function for printing to stdout
func print(a ...interface{}) {
	fmt.Println(a...)
}

func init() {
	loadConfig()
}

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalln("Cannot open config file", err)
	}
	decoder := json.NewDecoder(file)
	config = Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalln("Cannot get configuration from file", err)
	}
}
