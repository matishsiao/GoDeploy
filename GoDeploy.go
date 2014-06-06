package main

import (
	"fmt"	
	"flag"
	"time"	
	"bufio"
	"strings"
	"os"
	_ "os/signal"
	"regexp"
	"io/ioutil"
)

var version string = "0.0.2"
var configInfo ConfigInfo
var help *bool
var envConfig Configs
var clientList map[string]*Client
var clientChan chan string
var cmdReg = regexp.MustCompile("^cmd+|^help+|^exit+|^file+|^script+")
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
	clientChan = make(chan string)
	envConfig = config
	switch *configInfo.Mode {
		case "server": 
			go Listen(":"+envConfig.Configs.ServerPort)
		case "client":
			clientList = make(map[string]*Client)
			fmt.Printf(`You can keyin "help" to see more information.`+"\n")
			fmt.Println("GoDeploy:> Start connect to servers.")
			for _,v := range envConfig.Configs.ServerIP {
				cl := &Client{Server:v,User:envConfig.Configs.User,Pwd:envConfig.Configs.Password,ClientChan:clientChan}
				clientList[v] = cl
				go cl.Connect(v+":"+envConfig.Configs.ServerPort)			
			}
			time.Sleep(1500 * time.Millisecond)
			go Input()
			cmdEndPos()	
		default:
			fmt.Printf("No sure mode,Please use -help to see more information.\n")
			os.Exit(0)
	}
	go receiveChan()
	go checkServerHealth()
	
	for {
		configWatcher()
		time.Sleep(500 * time.Millisecond)
	}	
}


func receiveChan() {
	for {
		rev := <-clientChan
		//fmt.Println(rev)	
		if len(rev) > 0 && getConnectionListCount() == getServerProcessedCount() && getConnectionListCount() > 0{
			cmdEndPos()	
		}
	}
}

func getConnectionListCount() int {
	var count int = 0
	for _,v := range clientList {
		if v != nil && v.Connected {
			count++
		}
	}
	return count
}

func getServerProcessedCount() int {
	var count int = 0
	for _,v := range clientList {
		if v != nil && v.Connected && !v.Processing {
			count++
		}
	}
	return count
}

func checkServerHealth() {
	for {
		for _,v := range clientList {
			if v != nil {
				v.Write([]byte("health"))
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}


func Input() {
	for {
   		cmdReader := bufio.NewReader(os.Stdin)
   		
		cmdStr, _ := cmdReader.ReadString('\n')
		cmdStr = strings.Trim(cmdStr, "\r\n")   
		cmd := ""
		if cmdStr != ""{			
			if strings.Index(cmdStr," ") != -1 {
				cmd = cmdStr[:strings.Index(cmdStr," ")]
			} else {
				cmd = cmdStr
			} 
		}
					
		switch cmd {
				case "help":
					fmt.Printf("[Deploy help]\n")
					fmt.Printf("Input command list:\n")
					fmt.Printf("1.cmd: Send command to server\n")
					fmt.Printf("       example:cmd ls\n")					
					fmt.Printf("2.file: Send file to server\n")
					fmt.Printf("       example:file test.txt\n")					
					fmt.Printf("3.script: Use script to run commands.\n")
					fmt.Printf("       example:script test.dsh\n")				
					fmt.Printf("4.status: Show all server status.\n")
					fmt.Printf("5.help: Show help information.\n")
					fmt.Printf("6.exit: Exit appclication.\n")
					cmdEndPos()	
				case "reconnect":
					go reconnect()
				case "exit":
					fmt.Printf("Good bye.Have nice day.\n")
					os.Exit(0)
				case "status":
					fmt.Printf("[Server status]\n")
					for _,v := range clientList {
						if v != nil {
							fmt.Printf("[%s][Connection]:%v\n",v.Server,v.Connected)
						}
					}
					cmdEndPos()	
				case "file":				
					FileName := cmdStr[strings.Index(cmdStr," ")+1:]
					//fmt.Printf("send file:%v\n",FileName)			
					file, e := ioutil.ReadFile(FileName)
					if e != nil {
						fmt.Printf("Load file error: %v\n", e)
					}
					if file != nil && len(file) > 0 {
						for _,v := range clientList {
							if v != nil && v.Connected {
								go v.SendFile(FileName,file)
							}
						}
					} else {
						fmt.Println("file error: file is nil or file size is zero")
						cmdEndPos()				
					}
				case "script":
					go sendScript(cmdStr)
				default:
				if strings.Index(cmdStr," ") != -1 && cmdReg.MatchString(cmdStr) {
					for _,v := range clientList {
						if v != nil && v.Connected {
							v.InputCmd(cmdStr)
						}
					}
				} else {
					fmt.Printf("Wrong command input. You can use help to see more.\n")		
					cmdEndPos()				
				}				
		}		
			
	}	
}

func cmdEndPos(){
	fmt.Printf("GoDeploy:>")	
}

func reconnect() {
	for _,v := range clientList {
		if v != nil && !v.Connected {
			go v.Connect(v.Server+":"+envConfig.Configs.ServerPort)	
		}
	}
	time.Sleep(1000 * time.Millisecond)
	cmdEndPos()	
}

func sendScript(cmdStr string) {
	FileName := cmdStr[strings.Index(cmdStr," ")+1:]
	fmt.Printf("load script file:%v\n",FileName)			
	file, e := ioutil.ReadFile(FileName)
	if e != nil {
		fmt.Printf("Load script error: %v\n", e)
	}
					
	var script = strings.Split(string(file),"\n")
	for k,v := range script {
		fmt.Printf("[%d]:%v \n",k,v)
		if strings.Index(v," ") != -1 && cmdReg.MatchString(v) {
			for _,cv := range clientList {
				if cv != nil && cv.Connected {
					cv.InputCmd(v)				
				}				
			}							
		} else {
			fmt.Printf("Script Wrong command input.\n")
		}
		time.Sleep(1 * time.Second)						
	}
}
