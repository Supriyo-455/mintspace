package main

import (
	"fmt"
)

type Configuration struct {
	Address string
	Mode    string
}

func print(a ...interface{}) {
	fmt.Println(a...)
}

var config *Configuration = new(Configuration)

func init() {
	filename := "config.json"
	LoadJson(filename, config)
	LogInfo().Printf("Server configuration-> { Addr: %s, Mode: %s}\n", config.Address, config.Mode)
}
