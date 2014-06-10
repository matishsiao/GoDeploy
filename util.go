package main

import (
	"log"
	"os"
	"fmt"
	"time"
)


func WriteToLogFile(remote string, msg string) {
	logMsg := "[" + remote + "]:" + msg
	log.Println(logMsg)
	t := time.Now()
	fileName := t.Format("20060102") + ".log"
	fmt.Println(logMsg, fileName)
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)

	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	defer f.Close()
	log.SetOutput(f)
	log.Println(logMsg)
}