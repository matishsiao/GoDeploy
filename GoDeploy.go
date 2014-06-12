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
	"sync"
	_ "compress/gzip"
	_ "bytes"
)
const layout = "2006-01-02-15-04-05"

var version string = "0.0.4"
var configInfo ConfigInfo
var help *bool
var envConfig Configs
var clientList map[string]*Client
var clientChan chan string
var processChan chan bool
var cmdIdx int
var cmdScripts []string
var cmdReg = regexp.MustCompile("^cmd+|^help+|^exit+|^file+|^script+|^get+")
var cLock CommandLock 

type CommandLock struct {
	lock *sync.Mutex
	InputLock bool
 	UseScript bool
}

func (cl *CommandLock) KeyboardLock() {
	cl.lock.Lock()
	cl.InputLock = true
	cl.lock.Unlock()
}

func (cl *CommandLock) KeyboardUnLock() {
	cl.lock.Lock()	
	cl.InputLock = false
	cl.lock.Unlock()
}

func (cl *CommandLock) ScriptLock() {
	cl.lock.Lock()
	cl.UseScript = true
	cl.lock.Unlock()
}

func (cl *CommandLock) ScriptUnLock() {
	cl.lock.Lock()	
	cl.UseScript = false
	cl.lock.Unlock()
}

func Init() {
	ok,config := loadConfigs()
	if !ok {
		os.Exit(0)
	}
	setUlimit(10000)
	clientChan = make(chan string)
	processChan = make(chan bool)
	cLock = CommandLock{&sync.Mutex{},false,false}
	
	envConfig = config
	ServerInfo = goInfo.GetInfo()
}


func main() {
	configInfo.FileName = flag.String("config", "./config.json", "set config file path.")
	configInfo.Debug = flag.Bool("debug", false, "show debug trace message.")
	configInfo.Version = flag.Bool("version", false, "GoDeploy version.")
	configInfo.Mode = flag.String("mode", "client", "service mode:server,client default:client")
	configInfo.Load = flag.String("load", "", "load script to run.")
	configInfo.Group = flag.String("group", "", "connect group servers")
	configInfo.Server = flag.String("server", "", "connect specific server")
	configInfo.Record = flag.Bool("record", false, "Record command to script file")
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
		fmt.Printf("-record    Record command to script file\n")
		fmt.Printf("-version   Show version.\n")
		fmt.Printf("-load      Load script and run,with close\n")
		fmt.Printf("-help      Show help information\n")
		os.Exit(0)
	}

	if *configInfo.Version {
		fmt.Println("GoDeploy Version", version)
		os.Exit(0)
	}	
	
	Init()
	
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
			
			time.Sleep(time.Duration(envConfig.Configs.ConnectTimeout) * time.Second)			
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
			go Input()
			
			if *configInfo.Record {
				fmt.Println("start recording..")
			} 		
			if *configInfo.Load != "" {
				go sendScript(*configInfo.Load, true)
			} else {				
				cmdEndPos()	
			}
			
			
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
		//fmt.Printf("rev:%v,idx:%v,revIdx:%v,conn:%v,process:%v,kL:%v,uS:%v\n",data,cmdIdx,rev["cmdIdx"],getConnectionListCount(),getServerProcessedCount(),cLock.InputLock,cLock.UseScript)
		if rev["cmdIdx"] == strconv.Itoa(cmdIdx) && getConnectionListCount() == getServerProcessedCount() && getConnectionListCount() > 0{		
			cmdIdx++
			//fmt.Printf("complied.:%v,idx:%v,revIdx:%v,conn:%v,process:%v,kL:%v,uS:%v\n",data,cmdIdx,rev["cmdIdx"],getConnectionListCount(),getServerProcessedCount(),cLock.InputLock,cLock.UseScript)
			if cLock.UseScript {
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

func Input() {
	for {
		if !cLock.InputLock {
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
	if *configInfo.Record && cmdStr != "exit" {		
		cmdScripts = append(cmdScripts,cmdStr)
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
				
				if *configInfo.Record {
					scripts := ArrayToString(cmdScripts,"\n")					
					fileName := fmt.Sprintf("%s-%s.ds",time.Now().Format(layout),envConfig.Configs.User)					
					SaveFile("script",fileName,[]byte(scripts))
					fmt.Printf("Save record file to script/%v\n",fileName)
				}
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
				cLock.KeyboardLock()
				FileName := cmdStr[strings.Index(cmdStr," ")+1:]							
				file, e := ioutil.ReadFile(FileName)
				if e != nil {
					fmt.Printf("[Error]: %v\n", e)
					if *configInfo.Load != "" {
						fmt.Println("[System]:Exit")
						os.Exit(1)
					}
					cLock.KeyboardUnLock()					
				}
				if file != nil && len(file) > 0 {
					/*var fileZip bytes.Buffer
					w := gzip.NewWriter(&fileZip)
					w.Write(file)
					w.Close()*/
					for _,v := range clientList {
						if v != nil && v.Connected && v.Login {
							go v.SendFile(FileName,file,cmdIdx)
						}
					}
					cLock.KeyboardUnLock()	
				} else {
					fmt.Println("[Error]: File is nil or file size is zero")					
					cLock.KeyboardUnLock()	
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
	FileName := cmdStr[strings.Index(cmdStr," ")+1:]
	fmt.Printf("[Script]: Load script file[%v]\n",FileName)			
	file, e := ioutil.ReadFile(FileName)
	if e != nil {
		fmt.Printf("[Error]: %v\n", e)
		if load {
			sendCmd("exit")
		}
		return
	}					
	var script = strings.Split(string(file),"\n")	
	cLock.KeyboardLock()
	cLock.ScriptLock()
	for k,v := range script {
		if len(v) > 1 {
			fmt.Printf("[Script][Command][%d]:%v \n",k,v)
			if strings.Index(v," ") != -1 && cmdReg.MatchString(v) {
				ok := sendCmd(v)				
				if ok {
					fmt.Printf("[Script][Command]:Wait response from servers\n")					
					nextScript()
				} else {
					fmt.Printf("[Script][Command]:Have error leave now.\n")		
					cmdEndPos()	
					cLock.KeyboardUnLock()
					cLock.ScriptUnLock()		
					return
				}						
			} else {
				fmt.Printf("[Error]: Script Wrong command input.\n")
			}
						
		}						
	}
	cmdEndPos()	
	cLock.KeyboardUnLock()
	cLock.ScriptUnLock()	
	if load {
		sendCmd("exit")
	}
}

func nextScript() {
	for next := range processChan {
		if next {
			fmt.Printf("[Script]:Process success.Send next script\n")
			break
		}
	}
}
