package main

import(
	"fmt"
	"net"
	"log"
	"strings"
	"os/exec"
	"bytes"
	"os"
	"strconv"
	"time"
	"regexp"
	
)

type SrvClient struct {
	Conn *net.TCPConn
	User string
	Token string
	File bool
	FileObj *FileObject
	Login bool
	Outgoing chan []byte
	Incoming chan []byte
}

var srvClientList map[string]*SrvClient
var clientOutgoing chan []byte
var clientIncoming chan []byte
var serverIP string

func (cl *SrvClient) Init(conn *net.TCPConn,ch chan []byte,inch chan []byte) {
	cl.Conn = conn
	cl.Outgoing = ch
	cl.Incoming = inch
	cl.Read()
	go cl.Receiver()
}

func (cl *SrvClient) Receiver() {
	for {
		buf := <- cl.Incoming
		cl.Write(buf)
	}
}

func (cl *SrvClient) Write(data []byte) {
	cl.Conn.Write(data)
}

func (cl *SrvClient) Read() {
	buf := make([]byte,2048)
	for {
		 bytesRead, error := cl.Conn.Read(buf)
	    if error != nil {
	     	return 
	    }
	    data := buf[:bytesRead]
	    cl.Process(data)
	    //cl.Outgoing <- data
    }
}

func (cl *SrvClient) Process(data []byte) {
	if !cl.File {
		if string(data) != "health" {
			fmt.Printf("[%v][process]:%v \n",time.Now().Unix(),string(data))		
			revMsg := strings.Split(string(data),`,`)
			rev := make(map[string]string)
			for _,v := range revMsg {
				msg :=  strings.Split(v,`:`)
				if len(msg) == 2 {			
					rev[msg[0]] = msg[1]
				}
			}
			if v,ok := rev["action"]; ok {
				switch v {
					case "login":					
						 cl.User = rev["user"]
						 cl.Token = rev["pwd"]					 
						 if cl.CheckUser() {
						 	cl.Login = true
						 	cl.Write([]byte("action:login,status:success"))
						 } else {
						 	cl.Write([]byte("action:login,status:failed,msg:account not correct."))
						 	cl.Conn.Close()
						 }
					case "cmd":
						if cl.Login {
							cmdLine := strings.Split(rev["cmd"]," ")
							var cmd *exec.Cmd
							if len(cmdLine) > 1{
								arg := strings.Index(rev["cmd"]," ")+1	
								cmdStr := rev["cmd"]	
								args := strings.Split(cmdStr[arg:]," ")
								cmd = exec.Command(cmdLine[0],args...)
							} else {
								cmd = exec.Command(cmdLine[0])
							}
							cmd.Stdin = strings.NewReader("some input")
							var out bytes.Buffer
							var stderr bytes.Buffer
							cmd.Stdout = &out
							cmd.Stderr = &stderr
							err := cmd.Run()
							if err != nil {							
								msg := fmt.Sprintf("action:Server,ip:%v,msg:[Error]%v",serverIP,fmt.Sprint(err) + "-" + stderr.String())
								cl.Write([]byte(msg))
								return
							}
							//fmt.Printf("Server in all caps: %v\n", out.String())
							outStr := out.String()
							if outStr == "" {
								outStr = "done."
							}
							msg := fmt.Sprintf("action:Server,ip:%v,msg:[Cmd]\n%v",serverIP,outStr)
							cl.Write([]byte(msg))
						}
					case "file":
						if cl.Login {	
							if rev["cmd"] == "start" {					
								fileSize, err := strconv.ParseInt(rev["size"],10,64)
								if err != nil {
									fmt.Printf("conv error:%v\n",err)
								}
								if fileSize != 0 {
									fmt.Printf("start save file:%v\n",rev["file"])
									cl.File = true
									cl.FileObj = &FileObject{FileName:rev["file"],FileSize:fileSize}
								} else {
									msg := fmt.Sprintf("action:Server,ip:%v,msg:[Error]\n%v",serverIP,"File size not be zero.\n")
									cl.Write([]byte(msg))
								}					
							}
						}
					case "input":
						if cl.Login {	
							os.Stdout.Write([]byte(rev["cmd"]+"\n"))
							fmt.Printf("input all caps\n")
						}				
				}
			}
		}
	} else if string(data) != "health" {
		cl.FileObj.Data = append(cl.FileObj.Data,data...)		
		if int64(len(cl.FileObj.Data)) >= cl.FileObj.FileSize {
			fileName := cl.FileObj.FileName
			fmt.Printf("Server write file:%v\n",fileName)
			fo, err := os.Create(fileName)
	    	if err != nil {
	    	    fmt.Printf("fo Error:%v\n",err)
	    	 	cl.File = false
				fmt.Printf("Server write file done:%v\n",fileName)
				msg := fmt.Sprintf("action:Server,ip:%v,msg:[Cmd]%v",serverIP,fmt.Sprintf("Server write file %v",fileName))
				cl.Write([]byte(msg))
				cl.FileObj = nil 
	    	}
		    // close fo on exit and check for its returned error
		    defer func() {
		        if err := fo.Close(); err != nil {
		            fmt.Printf("fo Error:%v\n",err)
		        }
		    }()

	        if _, err := fo.Write(cl.FileObj.Data); err != nil {
	           fmt.Printf("fo Write Error:%v\n",err)
	        }
	    	
			cl.File = false
			fmt.Printf("Server write file done:%v\n",fileName)
			msg := fmt.Sprintf("action:Server,ip:%v,msg:[Cmd]%v",serverIP,fmt.Sprintf("Server write file %v",fileName))
			cl.Write([]byte(msg))
			cl.FileObj = nil
		}
	}
}

func (cl *SrvClient) CheckUser() bool {
	if cl.User == envConfig.Configs.User && cl.Token == envConfig.Configs.Password {
		return true	
	}
	return false
}

func Listen(port string) {
	clientOutgoing = make(chan []byte) 
	clientIncoming = make(chan []byte) 
	info,_:=net.InterfaceAddrs()
	var ipReg = regexp.MustCompile("[0-9]{1,3}.{3}[0-9]{1,3}")
	
	for _,addr := range info{
        ip := strings.Split(addr.String(),"/")[0]       
        if ipReg.MatchString(addr.String()) {
	        if ip != "127.0.0.1" {
	        	serverIP = ip
	        	break
	        }
	    }
    }
	fmt.Printf("Server:%v Listen Port:%v\n",serverIP,port)
	l, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("Listen Error:%v\n",err)
		return
	}
	ln := l.(*net.TCPListener)
		
	for {
    	conn, err := ln.AcceptTCP()
    	if err != nil {
    		log.Fatal(err)
    	}
    	go ProcessConn(conn,clientOutgoing,clientIncoming)
	}
}

func ProcessConn(c *net.TCPConn,ch chan []byte,inch chan []byte) {
	client := new(SrvClient)
	client.Init(c,ch,inch)
	//srvClientList = append(srvClientList,client)
}