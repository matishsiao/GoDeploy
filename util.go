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

func SaveFile(dir string,fileName string,data []byte) bool {
	os.Mkdir(dir,0777)
	fo, err := os.Create(dir +"/"+ fileName)
	if err != nil {
		fmt.Printf("File create error:%v\n",err)			
		return false
	}
	// close fo on exit and check for its returned error
	
	
		
	if _, err := fo.Write(data); err != nil {
		fmt.Printf("File Write Error:%v\n",err)
		return false
	}
	
	if err := fo.Close(); err != nil {
	   fmt.Printf("File close error:%v\n",err)
	   return false
	}
	
	return true
}

func ArrayToString(data []string,point string) string {
	var result string
	var dataLen int = len(data)
	for k,v := range data {
		result += v
		if k < dataLen - 1{
			result += point
		}
	}
	return result
}