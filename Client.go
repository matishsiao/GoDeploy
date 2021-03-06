package main

import(
	"net"
	"fmt"
	"strings"
	"time"
	"strconv"
	"os"
)

type Client struct {
	Server string
	Conn *net.TCPConn
	IsClose bool
	Connected bool
	User string
	Pwd string
	Login bool
	Receive []byte
	Msg string
	ClientChan chan string
	File bool
	FileObj *FileObject
	Processing bool
}

func (cl *Client) Init(conn *net.TCPConn) {
	cl.Conn = conn	
}

func (cl *Client) Write(data []byte) {
	if cl.Conn != nil {	
		//fmt.Printf("[%v][write]:%v\n",cl.Server,string(data))
		
		cl.Processing = true
		//WriteToLogFile("Write",string(data))	
		_,err := cl.Conn.Write(data)		
		if err != nil {
			fmt.Printf("[Client][Error]:Write error%v\n",err)
			cl.Close()
			cl.ClientChan <- "write failed"
		}
	} 
}

func (cl *Client) Read() {	
    buf := make([]byte,10240)
	for {
		n, err := cl.Conn.Read(buf)
	    if err != nil {
	     	fmt.Printf("[Client][Error]:Read error %v,client exit.\n",err)
	     	cl.Close()
	     	return
	    } else {	    	
	    	cl.Process(buf[:n])
	    }
    }
    WriteToLogFile("ReadError","Connection break.")	
}

func (cl *Client) Process(data []byte) {	
	//WriteToLogFile("Read",string(data))
	if !cl.File {
		cl.Receive = append(cl.Receive,data...)		
		revMsg := strings.Split(string(cl.Receive),`&`)		
		rev := make(map[string]string)
		for _,v := range revMsg {
			msg :=  strings.Split(v,`=`)
			if len(msg) == 2 {			
				rev[msg[0]] = msg[1]
			}
		}
		if v,ok := rev["action"]; ok {
			switch v {
				case "login":
					 if rev["status"] == "success" {
					 	cl.Login = true				 	
					 } else {
						cl.Close()
					 }
				case "server":
				default:
			}			
		}
		if _,idxOk := rev["cmdIdx"]; idxOk {
			cl.Msg = string(cl.Receive)[strings.Index(string(cl.Receive),`&msg=`)+5:strings.Index(string(cl.Receive),`&cmdIdx`)]
			cl.Receive = []byte{}
			if *configInfo.Load == "" {
				fmt.Printf("[%v]\n",cl.Server)	
				fmt.Printf("      [type]:%v\n",rev["type"])
				fmt.Printf("      [cmdIdx]:%v\n",rev["cmdIdx"])
				strCount := strings.Count(cl.Msg,"\n")
				if strCount > 1 {
					cl.Msg = strings.Replace(cl.Msg,"\n","\n        ",strCount -1)
					fmt.Printf("      [msg]:\n        %v",cl.Msg)
				} else if strCount == 1{
					fmt.Printf("      [msg]:%v",cl.Msg)
				} else {
					fmt.Printf("      [msg]:%v\n",cl.Msg)
				}
			} else {
				//str := strings.Replace(cl.Msg,"\n","\r",-1)
				//fmt.Printf("[script]client:%v,type:%v,cmdIdx:%v,msg:%v\n",cl.Server,rev["type"],rev["cmdIdx"],str)
				fmt.Printf("[script]client:%v,type:%v,cmdIdx:%v,msg:%v\n",cl.Server,rev["type"],rev["cmdIdx"],cl.Msg)
				
			}
			
		}
		cl.Processing = false	
		cl.ClientChan <- string(data)
	} else {		
		if cl.FileObj.FileSize == 0 {
			revMsg := strings.Split(string(data),`&`)		
			rev := make(map[string]string)
			for _,v := range revMsg {
				msg :=  strings.Split(v,`=`)
				if len(msg) == 2 {			
					rev[msg[0]] = msg[1]
				}
			}
			if v,ok := rev["action"]; ok {
				switch v {
					case "file":
						if rev["type"] == "send" {
							fileSize, err := strconv.ParseInt(rev["size"],10,64)
							if err != nil {
								fmt.Printf("conv error:%v\n",err)
							}
							cl.FileObj.FileSize = fileSize
						} else {
							
							if *configInfo.Load == "" {
								fmt.Printf("[%v]\n",cl.Server)	
								fmt.Printf("      [type]:%v\n",rev["type"])
								fmt.Printf("      [cmdIdx]:%v\n",rev["cmdIdx"])
								fmt.Printf("      [msg]:%v\n",rev["msg"])
							} else {
								str := strings.Replace(cl.Msg,"\n","\r",-1)
								fmt.Printf("[script]client:%v,type:%v,cmdIdx:%v,msg:%v\n",cl.Server,rev["type"],rev["cmdIdx"],str)
							}
							cl.File = false
							cl.Processing = false
							msg := fmt.Sprintf("action=server&ip=%v&type=get&msg=%v&cmdIdx=%s",cl.Server,fmt.Sprintf("Server get file %v failed",cl.FileObj.FileName),cl.FileObj.CmdIdx)
							cl.FileObj = nil
							cl.ClientChan <- string(msg)	
						}
					default:
				}
			}
		} else {
			var databytes []byte
			end := false
			for k,v := range data {
				if v == 0x1a {
					end = true
					databytes = append(databytes,data[:k-1]...)
				}
			}
			if !end {
				databytes = append(databytes,data...)
			}
			//fmt.Printf("server:% x \nend:%v\n",databytes,end)
			cl.FileObj.Data = append(cl.FileObj.Data,databytes...)		
			if end {		
			//if int64(len(cl.FileObj.Data)) >= cl.FileObj.FileSize {
				fileName := fmt.Sprintf("F%s-%s",cl.Server,cl.FileObj.FileName)
				fmt.Printf("Server write file:%v\n",fileName)
				os.Mkdir("file",0777)
				dir := "file/"		
				if ServerInfo.OS == "windows" {
					dir = "file\\"
				}		
				fo, err := os.Create(dir + fileName)
		    	if err != nil {
		    	    fmt.Printf("File create error:%v\n",err)
					cl.File = false
					cl.Processing = false
					msg := fmt.Sprintf("action=server&ip=%v&type=get&msg=%v&cmdIdx=%s",cl.Server,fmt.Sprintf("Server write file %v failed",fileName),cl.FileObj.CmdIdx)
					cl.FileObj = nil
					cl.ClientChan <- string(msg)	
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
				cl.Processing = false
				msg := fmt.Sprintf("action=server&ip=%v&type=get&msg=%v&cmdIdx=%s",cl.Server,fmt.Sprintf("Server write file %v success",fileName),cl.FileObj.CmdIdx)
				cl.FileObj = nil
				cl.ClientChan <- string(msg)
			}
		}
	}
}

func (cl *Client) InputCmd(cmdStr string,Idx int) {		
	if cmdStr != "" && cl.Connected {					
		var cmdInfo,cmd string
		if strings.Index(cmdStr," ")+1 != 0 {
			cmd = cmdStr[:strings.Index(cmdStr," ")]
			cmdInfo = cmdStr[strings.Index(cmdStr," ")+1:]
		} else {
			cmd = cmdStr
		}
		msg := fmt.Sprintf("action:%s,user:%s,cmd:%s,cmdIdx:%d",cmd,cl.User,cmdInfo,Idx)
		cl.Write([]byte(msg))
	} 
}

func (cl *Client) SendFile(fileName string,data []byte,Idx int) {	
	if cl.Connected {
		msg := fmt.Sprintf("action:%s,user:%s,cmd:%s,file:%s,size:%v,cmdIdx:%d","file",cl.User,"start",fileName,len(data),Idx)								
		cl.Write([]byte(msg))
		time.Sleep(1000 * time.Millisecond)
		cl.Write(data)
	} 
}

func (cl *Client) GetFile(fileName string,Idx int) {	
	if cl.Connected {
		msg := fmt.Sprintf("action:%s,user:%s,cmd:%s,file:%s,cmdIdx:%d","get",cl.User,"start",fileName,Idx)								
		cl.Write([]byte(msg))
		cl.File = true
		if ServerInfo.OS != "windows" && strings.LastIndex(fileName,"/") != -1 {
			fileName = fileName[strings.LastIndex(fileName,"/")+1:]
		} else if strings.LastIndex(fileName,"\\") != -1 {
			fileName = fileName[strings.LastIndex(fileName,"\\")+1:]
		}
		cl.FileObj = &FileObject{FileName:fileName,CmdIdx:strconv.Itoa(Idx)}
	} 
}

func (cl *Client) Printf(str string){
	fmt.Printf("Client[%v]%v\n",cl.Server,str)	
}

func (cl *Client) Close() {	
	cl.Processing = false
	cl.Conn.Close()		
	cl.checkConneciton(true, false)
}

func (cl *Client) checkConneciton(_close bool,_connected bool) {
	cl.IsClose = _close
	cl.Connected = _connected
	//WriteToLogFile("checkConneciton",fmt.Sprintf("Close:%v Connected:%v",_close,_connected))	
}

func (cl *Client) Connect(addr string) {
	cl.checkConneciton(true, false)
	serverAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		cl.Printf(fmt.Sprintf("Connect serverAddr Error:%v",err))		
		cl.checkConneciton(true, false)
		cl.ClientChan <- "connect serverAddr error"
		return
	}	
	con, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {	
		if *configInfo.Load == "" {
			fmt.Printf("----------------Client[%v]-------------\n",cl.Server)	
			fmt.Printf("[type]:%v\n","Error")
			fmt.Printf("[msg]:%v\n",err)
			fmt.Printf("----------------------------------------------\n")
		} else {
			fmt.Printf("[script]client:%v,type:%v,msg:%v\n",cl.Server,"Error",err)
		}
		cl.checkConneciton(true, false)	
		cl.ClientChan <- "connect failed"
		return
	}
   	
	cl.checkConneciton(false, true)
	time.Sleep(time.Second)
	cl.Init(con)
	cl.File = false
	msg := fmt.Sprintf("action:login,user:%s,pwd:%s",cl.User,cl.Pwd)
	//cl.Conn.SetReadDeadline(time.Now().Add(time.Duration(envConfig.Configs.Server.Timeout) * time.Second))
	go cl.Read()
	cl.Write([]byte(msg))	
}