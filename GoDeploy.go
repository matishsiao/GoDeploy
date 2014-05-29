package main

import (
	"fmt"	
	"flag"
	"time"	
	"bufio"
	"strings"
	"os"
	"regexp"
)

var version string = "0.0.1"
var configInfo ConfigInfo
var help *bool
var envConfig Configs
var clientList map[string]*Client

func main() {
	configInfo.FileName = flag.String("config", "./config.json", "set config file path.")
	configInfo.Debug = flag.Bool("debug", false, "show debug trace message.")
	configInfo.Version = flag.Bool("version", false, "GoDeploy version.")
	configInfo.Mode = flag.String("mode", "server", "service mode:server,client default:server")
	help = flag.Bool("help", false, "Show help information.")
	flag.Parse()
	
	if *help {
		fmt.Printf("GoDeploy Help:\n")
		fmt.Println("-version", version)		
		fmt.Printf("-config    Set cofing file path. Default value:%v\n", *configInfo.FileName)
		fmt.Printf("-debug     Show debug trace message. Default value:%v\n", *configInfo.Debug)
		fmt.Printf("-mode      Service mode:server,client default:server\n")
		fmt.Printf("-version   Show version.\n")
		fmt.Printf("-help      Show help information.\n")
		os.Exit(0)
	}

	if *configInfo.Version {
		fmt.Println("GoDeploy Version", version)
		os.Exit(0)
	}	
	
	ok,config := loadConfigs()
	if !ok {
		os.Exit(0)
	}
	envConfig = config
	switch *configInfo.Mode {
		case "server": 
			go Listen(":"+envConfig.Configs.ServerPort)
		case "client":
			clientList = make(map[string]*Client)
			for _,v := range envConfig.Configs.ServerIP {
				cl := &Client{Server:v,User:envConfig.Configs.User,Pwd:envConfig.Configs.Password}
				clientList[v] = cl
				go cl.Connect(v+":"+envConfig.Configs.ServerPort)			
			}
			go Input()
			fmt.Printf(`You can keyin "help" to see more information.`+"\n")
		default:
			fmt.Printf("No sure mode,Please use -help to see more information.\n")
			os.Exit(0)
	}
	
	
	for {
		configWatcher()
		time.Sleep(500 * time.Millisecond)
	}	
}

func Input() {
	var cmdReg = regexp.MustCompile("^cmd+|^help+|^exit+|^file+")
	for {
   		cmdReader := bufio.NewReader(os.Stdin)
		cmdStr, _ := cmdReader.ReadString('\n')
		cmdStr = strings.Trim(cmdStr, "\r\n")   		
		if cmdStr != "" {	
			switch cmdStr {
				case "help":
					fmt.Printf("Input command:\n")
					fmt.Printf("1.cmd: Send command to server\n")
					fmt.Printf("example:\n")
					fmt.Printf("   cmd ls\n")
					fmt.Printf("2.file: Send file to server\n")
					fmt.Printf("example:\n")
					fmt.Printf("   file test.txt\n")
					fmt.Printf("3.help: Show help information.\n")
					fmt.Printf("4.exit: Leave appclication.\n")
				case "exit":
					fmt.Printf("GoDeploy good bye.\n")
					os.Exit(0)
				default:
				if strings.Index(cmdStr," ") != -1 && cmdReg.MatchString(cmdStr) {
					for _,v := range clientList {
						v.InputCmd(cmdStr)
					}
				} else {
					fmt.Printf("Wrong command input.\n")
				}
			}			
		}
	}
	
}