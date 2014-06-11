package main

import (
	"fmt"	
	"flag"
	"time"	
	"bufio"
	"strings"
	"os"
	"syscall"
	_ "os/signal"
	"regexp"
	"io/ioutil"
	"strconv"
	"github.com/matishsiao/goInfo"
)

var version string = "0.0.3"
var configInfo ConfigInfo
var help *bool
var envConfig Configs
var clientList map[string]*Client
var clientChan chan string
var processChan chan bool
var cmdIdx int
var scriptStatus bool
var cmdReg = regexp.MustCompile("^cmd+|^help+|^exit+|^file+|^script+|^get+")
func main() {
	configInfo.FileName = flag.String("config", "./config.json", "set config file path.")
	configInfo.Debug = flag.Bool("debug", false, "show debug trace message.")
	configInfo.Version = flag.Bool("version", false, "GoDeploy version.")
	configInfo.Mode = flag.String("mode", "client", "service mode:server,client default:client")
	configInfo.Load = flag.String("load", "", "load script to run.")
	configInfo.Group = flag.String("group", "", "connect group servers")
	configInfo.Server = flag.String("server", "", "connect specific server")
	help = flag.Bool("help", false, "Show help information.")
	flag.Parse()
	
	if *help {
		fmt.Printf("GoDeploy Help:\n")
		fmt.Println("-version", version)		
		fmt.Printf("-config    Set cofing file path. Default value:%v\n", *configInfo.FileName)
		fmt.Printf("-debug     Show debug trace message. Default value:%v\n", *configInfo.Debug)
		fmt.Printf("-mode      Service mode:server,client default:client\n")
		fmt.Printf("-group     Connect specific group servers\n")
		fmt.Printf("-server    Connect specific server\n")
		fmt.Printf("-version   Show version.\n")
		fmt.Printf("-load      Load script and run,with close\n")
		fmt.Printf("-help      Show help information\n")
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
	setUlimit(10000)
	clientChan = make(chan string)
	processChan = make(chan bool)
	envConfig = config
	ServerInfo = goInfo.GetInfo()
	switch *configInfo.Mode {
		case "server": 
			go Listen(":"+envConfig.Configs.ServerPort)
		case "client":
			clientList = make(map[string]*Client)
			fmt.Printf(`You can keyin "help" to see more information.`+"\n")
			fmt.Println("Start connect to servers.")
			if *configInfo.Server == "" {
				for _,v := range envConfig.Configs.Server {
					fmt.Printf("[Server][%v]:%v\n",v.Group,v.Ip)
					if *configInfo.Group != "" {
						if v.Group == *configInfo.Group {
							connect(v)
						}
					} else {
						connect(v)
					}
				}
			} else {
				var ipReg = regexp.MustCompile(`([0-9]{1,3}\.){3}[0-9]{1,3}$`)
				 fmt.Println(ipReg.FindAllStringSubmatch(*configInfo.Server, -1))
        		if ipReg.MatchString(*configInfo.Server) {        			
					connect(ServerNode{Ip:*configInfo.Server,Group:"Specific"})
	    		} else {
	    			fmt.Println("[Error]: Server ip format has wrong.Tcp4 accept only.")
	    			os.Exit(0)
	    		}
				
			}
			fmt.Println("connecting...")
			
			time.Sleep(3 * time.Second)			
			fmt.Println("----------------[Server status]---------------")	
			for _,v := range clientList {
				if v != nil {
					fmt.Printf("[%s][Connection]:%v [Login]:%v\n",v.Server,v.Connected,v.Login)
					if !v.Login {
						v.Processing = false
					} 
				}
			}
			fmt.Printf("----------------------------------------------\n\n")
			
			go receiveChan()
			go checkServerHealth()
						
			if *configInfo.Load != "" {
				go sendScript(*configInfo.Load, true)
			} else {				
				cmdEndPos()	
			}
			go Input()
			
		default:
			fmt.Printf("No sure mode,Please use -help to see more information.\n")
			os.Exit(0)
	}
	
	
	
	
	for {
		configWatcher()
		if getConnectionListCount() == 0 &&  *configInfo.Mode != "server" {			
			fmt.Println("[Error] no servers has connected.exit")
			os.Exit(0)
		}
		time.Sleep(500 * time.Millisecond)
	}	
}

func connect(v ServerNode) {
	cl := &Client{Server:v.Ip,User:envConfig.Configs.User,Pwd:envConfig.Configs.Password,ClientChan:clientChan}
	clientList[v.Ip] = cl
	go cl.Connect(v.Ip+":"+envConfig.Configs.ServerPort)
}

func receiveChan() {
	for {
		data := <-clientChan
		revMsg := strings.Split(data,`&`)		
		rev := make(map[string]string)
		for _,v := range revMsg {
			msg :=  strings.Split(v,`=`)
			if len(msg) == 2 {			
				rev[msg[0]] = msg[1]
			}
		}
		//fmt.Println("rev:",rev,cmdIdx,getConnectionListCount(),getServerProcessedCount())
		if rev["cmdIdx"] == strconv.Itoa(cmdIdx) && getConnectionListCount() == getServerProcessedCount() && getConnectionListCount() > 0{		
			cmdIdx++
			if scriptStatus {
				processChan <- true
			}			
			cmdEndPos()	
		}
	}
}

func getConnectionListCount() int {
	var count int = 0
	for _,v := range clientList {
		if v != nil && v.Connected && v.Login {
			count++
		}
	}
	return count
}

func getServerProcessedCount() int {
	var count int = 0
	for _,v := range clientList {
		if v != nil && v.Connected  && v.Login && !v.Processing {
			count++
		}		
	}
	return count
}

func checkServerHealth() {
	for {
		for _,v := range clientList {
			if v != nil && !v.Processing {
				v.Write([]byte("health"))
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}


func Input() {
	for {
		if !scriptStatus {
	   		cmdReader := bufio.NewReader(os.Stdin)
			cmdStr, _ := cmdReader.ReadString('\n')
			cmdStr = strings.Trim(cmdStr, "\r\n")   
			sendCmd(cmdStr)
		}
	}	
}

func sendCmd(cmdStr string) bool {	
	cmd := ""
	if cmdStr != "" {			
		if strings.Index(cmdStr," ") != -1 {
			cmd = cmdStr[:strings.Index(cmdStr," ")]
		} else {
			cmd = cmdStr
		} 
	} else {
		cmdEndPos()
		return false
	}
	
	switch cmd {
			case "help":
				fmt.Printf("[Deploy help]\n")
				fmt.Printf("Input command list:\n")
				fmt.Printf("1.cmd:		Send command to server\n")
				fmt.Printf("       			Example:cmd ls\n")
				fmt.Printf("2.env:		Show all server os information.\n")
				fmt.Printf("3.exit:		Exit appclication.\n")					
				fmt.Printf("4.file: 	Send file to server\n")
				fmt.Printf("       			Example:file test.txt\n")	
				fmt.Printf("5.get:		Get file from all connect servers\n")
				fmt.Printf("6.help:		Show help information.\n")								
				fmt.Printf("7.script: 	Use script to run commands.\n")
				fmt.Printf("	  	    	Example:script test.dsh\n")				
				fmt.Printf("8.status:	Show all server status.\n")				
				
				
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
				file, e := ioutil.ReadFile(FileName)
				if e != nil {
					fmt.Printf("[Error]: %v\n", e)
					if *configInfo.Load != "" {
						fmt.Println("[System]:Exit")
						os.Exit(1)
					}
					scriptStatus = false					
				}
				if file != nil && len(file) > 0 {
					for _,v := range clientList {
						if v != nil && v.Connected && v.Login {
							go v.SendFile(FileName,file,cmdIdx)
						}
					}
				} else {
					fmt.Println("[Error]: File is nil or file size is zero")					
					scriptStatus = false
					return false
					cmdEndPos()				
				}
			case "env":
				for _,v := range clientList {
					if v != nil && v.Connected {
						v.InputCmd(cmdStr,cmdIdx)
					}
				}
			case "script":
				sendScript(cmdStr, false)
			case "cmd":
				if strings.Index(cmdStr,"cmd ") != -1 && cmdReg.MatchString(cmdStr) {					
					for _,v := range clientList {
						if v != nil && v.Connected && v.Login {
							v.InputCmd(cmdStr,cmdIdx)
						}
					}
				}
			case "get":
				if strings.Index(cmdStr,"get ") != -1 && cmdReg.MatchString(cmdStr) {	
					FileName := cmdStr[strings.Index(cmdStr," ")+1:]			
					for _,v := range clientList {
						if v != nil && v.Connected && v.Login {
							v.GetFile(FileName,cmdIdx)
						}
					}
				}				
			default:			
			fmt.Printf("[Error]: Wrong command input. You can use help to see more.\n")		
			cmdEndPos()				
							
	}	
	return true
}

func setUlimit(number uint64) {
	var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        fmt.Println("[Error]: Getting Rlimit ", err)
    }    
    rLimit.Max = number
    rLimit.Cur = number
    err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        fmt.Println("[Error]: Setting Rlimit ", err)
    }
    err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        fmt.Println("[Error]: Getting Rlimit ", err)
    }
    
}

func cmdEndPos(){
	if *configInfo.Load == "" {
		fmt.Printf("GoDeploy:>")
	}	
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



func sendScript(cmdStr string, load bool) {
	scriptStatus = true
	FileName := cmdStr[strings.Index(cmdStr," ")+1:]
	fmt.Printf("[Script]: Load script file[%v]\n",FileName)			
	file, e := ioutil.ReadFile(FileName)
	if e != nil {
		fmt.Printf("[Error]: %v\n", e)
	}
					
	var script = strings.Split(string(file),"\n")
	time.Sleep(1500 * time.Millisecond)
	
	for k,v := range script {		
		if len(v) > 1 {
			fmt.Printf("[Script][Command][%d]:%v \n",k,v)
			if strings.Index(v," ") != -1 && cmdReg.MatchString(v) {
				ok := sendCmd(v)				
				if ok {
					nextScript()
				} else {
					fmt.Printf("[Script][Command]:Have error leave now.\n")		
					cmdEndPos()	
					scriptStatus = false			
					return
				}						
			} else {
				fmt.Printf("[Error]: Script Wrong command input.\n")
			}
						
		}						
	}
	cmdEndPos()	
	scriptStatus = false
	if load {
		sendCmd("exit")
	}
}

func nextScript() {
	for {
		next := <- processChan
		if next {
			fmt.Printf("[Script]: Process success.Send next script.\n")		
			break
		} else {
			break
		}
	}
}
