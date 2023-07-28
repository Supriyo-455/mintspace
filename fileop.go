package main

import (
	"log"
	"os"
)

func WriteToFile(path, data string) error {
	err := os.WriteFile(path, []byte(data), 0666)
	return err
}

func ReadFile(path string) error {
	if FileExist(path) {
		file, err := os.OpenFile(path, os.O_RDONLY, 0666)
		if err != nil {
			log.Fatalln("error occured! ", err.Error())
			return err
		}
		// TODO: DO stuff with the file

		file.Close()
	}
	return nil
}

func FileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatalln("error occured! ", err.Error())
	}
	return true
}
