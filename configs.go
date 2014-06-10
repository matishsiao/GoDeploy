package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
)

func loadConfigs() (bool,Configs) {	
	file, e := ioutil.ReadFile(*configInfo.FileName)
	if e != nil {
		fmt.Printf("Load config error: %v\n", e)
		os.Exit(1)
	}
	
	var config Configs
	err := json.Unmarshal(file, &config)
	if err != nil {
		fmt.Printf("Config load error:%v \n",err)
		return false,config
	}
	return true,config
}

func configWatcher() {
	file, err := os.Open(*configInfo.FileName) // For read access.
	if err != nil {
		fmt.Println(err)
	}
	info, err := file.Stat()
	if err != nil {
		fmt.Println(err)
	}
	if configInfo.Size == 0 {
		configInfo.Size = info.Size()
		configInfo.ModTime = info.ModTime()
	}

	if info.Size() != configInfo.Size || info.ModTime() != configInfo.ModTime {
		fmt.Printf("Config changed.Reolad.\n")
		configInfo.Size = info.Size()
		configInfo.ModTime = info.ModTime()
		ok,config := loadConfigs()
		if ok {
			envConfig = config
		}

	}
	defer file.Close()
}