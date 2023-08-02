package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var config Configuration

type Configuration struct {
	Address       string
	Static        string
	Mode          string
	MongoURI      string
	MongoDatabase string
}

func print(a ...interface{}) {
	fmt.Println(a...)
}

func init() {
	loadConfig()
	LogInfo().Printf("Server configuration-> { Addr: %s, StaticDir: %s, Mode: %s}\n", config.Address, config.Static, config.Mode)
}

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		LogError().Fatalln("Cannot open config file", err)
	}
	decoder := json.NewDecoder(file)
	config = Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		LogError().Fatalln("Cannot get configuration from file", err)
	}
}
