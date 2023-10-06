package main

import (
	"log"
	"os"
)

const (
	LogsDirpath = "logs"
	LogsError   = "logs.txt"
	LogsInfo    = "logs.txt"
	LogsWarn    = "logs.txt"
)

func getLogfilePath(logType string) string {
	return LogsDirpath + "/" + logType
}

func LogInfo() *log.Logger {
	file, err := os.OpenFile(getLogfilePath(LogsInfo), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	return log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func LogWarn() *log.Logger {
	file, err := os.OpenFile(getLogfilePath(LogsWarn), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	return log.New(file, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func LogError() *log.Logger {
	file, err := os.OpenFile(getLogfilePath(LogsError), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	return log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
