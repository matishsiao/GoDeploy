package main

import(
	"fmt"
	"net"
	_"log"
	"strings"
	"os/exec"
	"bytes"
	"os"
	"strconv"
	"time"
	"runtime"
	"regexp"
	"github.com/matishsiao/goInfo"
	"io/ioutil"
	"sync"
	_ "io"
	_ "compress/gzip"
)

type SrvClient struct {
	Conn *net.TCPConn
	mu *sync.Mutex
	User string
	Token string
	File bool
	FileObj *FileObject
	Login bool
	Connected bool
}

var srvClientList []*SrvClient
var serverIP string
var ServerInfo *goInfo.GoInfoObject

func (cl *SrvClient) Init(conn *net.TCPConn) {
	cl.Conn = conn
	cl.Connected = true
	WriteToLogFile("Server",fmt.Sprintf("Client:%v connected\n",conn.RemoteAddr()))
	cl.Conn.SetReadDeadline(time.Now().Add(time.Duration(envConfig.Configs.Server.Timeout) * time.Second))
	cl.Read()
}

func (cl *SrvClient) Close() {
	cl.mu.Lock()
	cl.Conn.Close()	
	cl.Connected = false
	cl.mu.Unlock()
	runtime.GC()
	WriteToLogFile("Server",fmt.Sprintf("Client:%v closed.connection:%v\n",cl.Conn.RemoteAddr(),cl.Connected))
}

func (cl *SrvClient) Write(data []byte) {
	
	n,err := cl.Conn.Write(data)
	WriteToLogFile("Server",fmt.Sprintf("write[%v][%d]:%v\n",cl.Conn.RemoteAddr(),n,string(data)))
	if err != nil {
		WriteToLogFile("Server",fmt.Sprintf("[%v][Write Error]:%v [useGoroutine]:%v\n",time.Now().Format("2006-01-02-15-04-05"),err,runtime.NumGoroutine()))	
		cl.Close()
	}
}

func (cl *SrvClient) Read() {
	buf := make([]byte,2048)
	for cl.Connected {
		bytesRead, err := cl.Conn.Read(buf)
	    if err != nil {
	     	WriteToLogFile("Server",fmt.Sprintf("[%v][Read Error]:%v [useGoroutine]:%v\n",time.Now().Format("2006-01-02-15-04-05"),err,runtime.NumGoroutine()))	
	     	cl.Close()
	     	break
	    } else {
	    	cl.Conn.SetReadDeadline(time.Now().Add(time.Duration(envConfig.Configs.Server.Timeout) * time.Second))
	    	data := buf[:bytesRead]
	    	if len(data) > 0 { 
	    		cl.Process(data)
	    	}
	    }
    }
}

func (cl *SrvClient) Process(data []byte) {
	if !cl.File {
			fmt.Printf("[%v][process]:%v [useGoroutine]:%v\n",time.Now().Format("2006-01-02-15-04-05"),string(data),runtime.NumGoroutine())	
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
						 	cl.Close()
						 }
					case "cmd":
						if cl.Login {
							outStr := cl.runCmd(rev,false)							
							for strings.Index(outStr ,"broken pipe") != -1 {
								outStr = cl.runCmd(rev,false)
								time.Sleep(250 * time.Millisecond)
							}						
							if outStr == "" {
								outStr = rev["cmd"] +" success"
							}							
							msg := fmt.Sprintf("action=server&ip=%v&type=cmd&msg=%v&cmdIdx=%s",serverIP,outStr,rev["cmdIdx"])
							cl.Write([]byte(msg))							
						}
					case "sh":
						if cl.Login {
							outStr := cl.runCmd(rev,true)							
							for strings.Index(outStr ,"broken pipe") != -1 {
								outStr = cl.runCmd(rev,true)
								time.Sleep(250 * time.Millisecond)
							}						
							if outStr == "" {
								outStr = rev["cmd"] +" success"
							}							
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
										fileName = rev["file"][strings.LastIndex(rev["file"],"/")+1:]
									} else if strings.LastIndex(rev["file"],"\\") != -1 {
										fileName = rev["file"][strings.LastIndex(rev["file"],"\\")+1:]
									}	
									cl.FileObj = &FileObject{FileName:fileName,FileSize:fileSize,CmdIdx:rev["cmdIdx"]}
								} else {
									msg := fmt.Sprintf("action=server&ip=%v&type=file&msg=%v&cmdIdx=%s",serverIP,"file size not be zero.\n",rev["cmdIdx"])
									cl.Write([]byte(msg))
								}					
							}
						}
					case "get":
						if cl.Login {					
							file, e := ioutil.ReadFile(rev["file"])
							if e != nil {
								fmt.Printf("[Open][Error]: %v\n", e)
							}
							if file != nil && len(file) > 0 {
								msg := fmt.Sprintf("action=file&type=send&msg=%v&fileName=%v&size=%v&cmdIdx=%s","start send",rev["file"],len(file),rev["cmdIdx"])
								cl.Write([]byte(msg))
								time.Sleep(1000 * time.Millisecond)
								file = append(file,0x1a)
								cl.Write(file)
							} else {				
								msg := fmt.Sprintf("action=file&type=error&msg=%v&cmdIdx=%s","[Error]: File is nil or file size is zero",rev["cmdIdx"])								
								cl.Write([]byte(msg))
								return			
							}
							
						}
					case "env":
						if cl.Login {	
							gi := goInfo.GetInfo()
							msg := fmt.Sprintf("action=server&ip=%v&type=env&msg=%v&cmdIdx=%s",serverIP,gi.String(),rev["cmdIdx"])
							cl.Write([]byte(msg))
						}	
					case "gorountine":
						if cl.Login {	
							msg := fmt.Sprintf("action=server&ip=%v&type=gorountine&msg=%v&cmdIdx=%s",serverIP,strconv.Itoa(runtime.NumGoroutine()),rev["cmdIdx"])
							cl.Write([]byte(msg))
						}	
					case "gc":
						if cl.Login {
							for k,_ := range srvClientList {								
								srvClientList[k].Close()
							} 	
						}	
							
				}
			}
		
	} else{
		var databytes []byte
		end := false
		/*for k,v := range data {
			if v == 0x1a {
				end = true
				databytes = append(databytes,data[:k-1]...)
			}
		}*/
		if !end {
			all := int64(len(cl.FileObj.Data) + len(data))
			need := cl.FileObj.FileSize - all 
			if *configInfo.Debug {
				fmt.Printf("need pkt:%v now pkt:%v filesize:%v\n",need,all,cl.FileObj.FileSize)
			}	 
			if all > cl.FileObj.FileSize {
				
				if int64(len(data)) >= need {
					databytes = append(databytes,data[:need]...)
				} 
				end = true
			} else {
				databytes = append(databytes,data...)
				if need == 0 {
					end = true
				}	
			}
		}
		//fmt.Printf("server:% x \nend:%v\n",databytes,end)
		cl.FileObj.Data = append(cl.FileObj.Data,databytes...)		
		if end {
			fileName := cl.FileObj.FileName
			WriteToLogFile("Server",fmt.Sprintf("Server write file:%v\n",fileName))
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


func (cl *SrvClient) runCmd(rev map[string]string,shMode bool) string {
	fmt.Printf("runCmd:%v\n",rev)
	var cmdLine []string
	var cmd *exec.Cmd
	
	sh := false
	if !shMode {
		if strings.Index(rev["cmd"],"|") != -1 {
			sh = true
		} else {
			cmdLine = strings.Split(rev["cmd"]," ")
		}
	}else{
		sh = true
	}
	
	if len(cmdLine) > 1{
		arg := strings.Index(rev["cmd"]," ")+1	
		cmdStr := rev["cmd"]	
		args := strings.Split(cmdStr[arg:]," ")
		cmd = exec.Command(cmdLine[0],args...)
	} else {
		if sh {
			cmd = exec.Command("sh","-c",rev["cmd"])
		} else {
			cmd = exec.Command(cmdLine[0])
		}
	}
	//cmd.Stdin = strings.NewReader("cmdinput")
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
	if cl.User == envConfig.Configs.Auth.User && cl.Token == envConfig.Configs.Auth.Password {
		return true	
	}
	return false
}

func Listen(port string) {
	 
	info,_:=net.InterfaceAddrs()
	var ipReg = regexp.MustCompile(`([0-9]{1,3}\.){3}[0-9]{1,3}$`)
	
	for _,addr := range info{
        ip := strings.Split(addr.String(),"/")[0]       
        if ipReg.MatchString(ip) {
	        if ip != "127.0.0.1" && ip != "0.0.0.0" {
	        	serverIP = ip
	        	break
	        }
	    }
    }
	fmt.Printf("[Server] %v%v start listen.\n",serverIP,port)
	l, err := net.Listen("tcp4", port)
	if err != nil {
		fmt.Printf("Listen Error:%v\n",err)
		os.Exit(1)
		return
	}
	ln := l.(*net.TCPListener)
		
	for {
    	conn, err := ln.AcceptTCP()
    	if err != nil {
    		fmt.Println(err)
    	} else {
    		go ProcessConn(conn)
    	}
	}
}

func ProcessConn(c *net.TCPConn) {
	client := new(SrvClient)
	client.mu = &sync.Mutex{}
	srvClientList = append(srvClientList,client)
	client.Init(c)
	
}