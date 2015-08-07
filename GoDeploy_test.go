package main

import (
	"testing"
	"os/exec"
	"fmt"
	"os"
	"bytes"
	_"time"
)

func TestScript(t *testing.T) {
	runScript()
}

func runScript() {
	scriptName := fmt.Sprintf("%v.ds","1403854457")
	cmd := exec.Command("./GoDeploy","-load","conf_script/"+scriptName)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()		
	msg := ""
	fmt.Println("------------------------------------------------------")
	if err != nil {
		msg = fmt.Sprintf("error %v",fmt.Sprint(err) + "-" + stderr.String())		
	} else {
		msg = out.String()
	}
	fmt.Println(msg)
	fmt.Println("------------------------------------------------------")
}


func saveScript(fileName string,data string) {
	
	dir := "conf_script"
	fmt.Println("saveScript:",fileName)
	os.Mkdir(dir,0777)	
	fo, err := os.Create(dir + "/" + fileName)
	if err != nil {
	     fmt.Printf("File create error %v =>%v\n",fileName,err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
		    fmt.Printf("File close error:%v\n",err)
		}
	}()
	
	if _, err := fo.Write([]byte(data)); err != nil {
		fmt.Printf("File Write Error:%v\n",err)
	}
}

func BenchmarkScript(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runScript()
	}
}
