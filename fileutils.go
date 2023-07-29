package main

import (
	"encoding/json"
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
			Error().Fatalln(err.Error())
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
		Error().Fatalln(err.Error())
	}
	return true
}

func LoadJson(filename string, key interface{}) {
	inFile, err := os.Open(filename)
	if err != nil {
		Error().Fatalln(err.Error())
	}

	decoder := json.NewDecoder(inFile)

	err = decoder.Decode(key)
	if err != nil {
		Error().Fatalln(err.Error())
	}

	inFile.Close()
}
