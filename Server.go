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
	"github.com/matishsiao/goInfo"
)

type SrvClient struct {
	Conn *net.TCPConn
	User string
	Token string
	File bool
	FileObj *FileObject
	Login bool
}

var srvClientList map[string]*SrvClient
var serverIP string
var ServerInfo *goInfo.GoInfoObject

func (cl *SrvClient) Init(conn *net.TCPConn) {
	cl.Conn = conn
	ServerInfo = goInfo.GetInfo()
	cl.Read()
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
    }
}

func (cl *SrvClient) Process(data []byte) {
	if !cl.File {
		if string(data) != "health" {
			fmt.Printf("[%v][process]:%v\n",time.Now().Unix(),string(data))	
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
						 	cl.Write([]byte("action=login&status=success&type=system&msg=login success&cmdIdx=-1"))
						 } else {
						 	cl.Write([]byte("action=login&status=failed&type=system&msg=account not correct.&cmdIdx=-1"))
						 	cl.Conn.Close()
						 }
					case "cmd":
						if cl.Login {
							outStr := cl.runCmd(rev)							
							for strings.Index(outStr ,"broken pipe") != -1 {
								outStr = cl.runCmd(rev)
								time.Sleep(250 * time.Millisecond)
							}						
							if outStr == "" {
								outStr = rev["cmd"] +" success"
							}
							//fmt.Println("Server out string:",outStr)
							msg := fmt.Sprintf("action=server&ip=%v&type=cmd&msg=%v&cmdIdx=%s",serverIP,outStr,rev["cmdIdx"])
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
									fileName := rev["file"]
									if ServerInfo.OS != "windows" && strings.LastIndex(rev["file"],"/") != -1 {
										fileName = rev["file"][strings.LastIndex(rev["file"],"/"):]
									} else if strings.LastIndex(rev["file"],"\\") != -1 {
										fileName = rev["file"][strings.LastIndex(rev["file"],"\\"):]
									}	
									cl.FileObj = &FileObject{FileName:fileName,FileSize:fileSize,CmdIdx:rev["cmdIdx"]}
								} else {
									msg := fmt.Sprintf("action=server&ip=%v&type=file&msg=%v&cmdIdx=%s",serverIP,"file size not be zero.\n",rev["cmdIdx"])
									cl.Write([]byte(msg))
								}					
							}
						}
					case "env":
						if cl.Login {	
							gi := goInfo.GetInfo()
							msg := fmt.Sprintf("action=server&ip=%v&type=env&msg=%v&cmdIdx=%s",serverIP,gi.String(),rev["cmdIdx"])
							cl.Write([]byte(msg))
						}				
				}
			}
		}
	} else if string(data) != "health" {
		cl.FileObj.Data = append(cl.FileObj.Data,data...)		
		if int64(len(cl.FileObj.Data)) >= cl.FileObj.FileSize {
			fileName := cl.FileObj.FileName
			//fmt.Printf("Server write file:%v\n",fileName)
			os.Mkdir("file",0777)
			dir := "file/"
			if ServerInfo.OS == "windows" {
				dir = "file\\"
			}
			fo, err := os.Create(dir + fileName)
	    	if err != nil {
	    	    fmt.Printf("File create error:%v\n",err)
	    	 	cl.File = false
				fmt.Printf("Server write file done:%v\n",fileName)
				msg := fmt.Sprintf("action=server&ip=%v&type=file&msg=%v&cmdIdx=%s",serverIP,fmt.Sprintf("error,server write file %v failed",fileName),cl.FileObj.CmdIdx)
				cl.Write([]byte(msg))
				cl.FileObj = nil 
				return
	    	}
		    // close fo on exit and check for its returned error
		    defer func() {
		        if err := fo.Close(); err != nil {
		            fmt.Printf("File close error:%v\n",err)
		        }
		    }()

	        if _, err := fo.Write(cl.FileObj.Data); err != nil {
	           fmt.Printf("File Write Error:%v\n",err)
	        }
	    	
			cl.File = false
			
			msg := fmt.Sprintf("action=server&ip=%v&type=file&msg=%v&cmdIdx=%s",serverIP,fmt.Sprintf("Server write file %v success",fileName),cl.FileObj.CmdIdx)
			cl.Write([]byte(msg))
			cl.FileObj = nil
		}
	}
}


func (cl *SrvClient) runCmd(rev map[string]string) string {
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
	cmd.Stdin = strings.NewReader("cmdinput")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("error %v",fmt.Sprint(err) + "-" + stderr.String())		
	} else {
		msg = out.String()
	} 
	return msg
}

func (cl *SrvClient) CheckUser() bool {
	if cl.User == envConfig.Configs.User && cl.Token == envConfig.Configs.Password {
		return true	
	}
	return false
}

func Listen(port string) {
	 
	info,_:=net.InterfaceAddrs()
	var ipReg = regexp.MustCompile("[0-9]{1,3}.{3}[0-9]{1,3}")
	
	for _,addr := range info{
        ip := strings.Split(addr.String(),"/")[0]       
        if ipReg.MatchString(addr.String()) {
	        if ip != "127.0.0.1" && ip != "0.0.0.0" {
	        	serverIP = ip
	        	break
	        }
	    }
    }
	fmt.Printf("[Server]:%v:%v start listen.\n",serverIP,port)
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
    	go ProcessConn(conn)
	}
}

func ProcessConn(c *net.TCPConn) {
	client := new(SrvClient)
	client.Init(c)
}