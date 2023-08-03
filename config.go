package main

import (
	"fmt"
)

var config Configuration

type Configuration struct {
	Address  string
	Static   string
	Mode     string
	Database string
}

func print(a ...interface{}) {
	fmt.Println(a...)
}

func init() {
	filename := "config.json"
	config := Configuration{}
	LoadJson(filename, &config)
	LogInfo().Printf("Server configuration-> { Addr: %s, StaticDir: %s, Mode: %s}\n", config.Address, config.Static, config.Mode)
}
