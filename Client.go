package main

import(
	"net"
	"fmt"
	"strings"
	"time"
)

type Client struct {
	Server string
	Conn *net.TCPConn
	Close bool
	Connected bool
	User string
	Pwd string
	Login bool
	ClientChan chan string
	Processing bool
}

func (cl *Client) Init(conn *net.TCPConn) {
	cl.Conn = conn	
}

func (cl *Client) Write(data []byte) {
	if cl.Conn != nil {	
		//fmt.Printf("[%v][write]:%v\n",cl.Server,string(data))
		if string(data) != "health" {
			cl.Processing = true
		}
		_,err := cl.Conn.Write(data)		
		if err != nil {
			fmt.Printf("Client write error:%v\n",err)
			cl.Processing = false
			cl.Conn.Close()		
			cl.checkConneciton(true, false)
			cl.ClientChan <- "write failed"
		}
	} 
}

func (cl *Client) Read() {
	buf := make([]byte,2048)
	for {
		n, error := cl.Conn.Read(buf)
	    if error != nil {
	     	
	    } else {
	    	cl.Process(buf[:n])
	    }
    }
}

func (cl *Client) Process(data []byte) {			
	revMsg := strings.Split(string(data),`&`)
	//cl.Printf(fmt.Sprintf("%v",string(data)))
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
					cl.Conn.Close()		
					cl.checkConneciton(true, false) 
				 }
			case "server":
			default:
		}
	}
	fmt.Printf("[%v]\n",cl.Server)	
	fmt.Printf("      [type]:%v\n",rev["type"])
	fmt.Printf("      [cmdIdx]:%v\n",rev["cmdIdx"])
	strCount := strings.Count(rev["msg"],"\n")
	if strCount > 1 {
		rev["msg"] = strings.Replace(rev["msg"],"\n","\n        ",strCount -1)
		fmt.Printf("      [msg]:\n        %v",rev["msg"])
	} else if strCount == 1{
		fmt.Printf("      [msg]:%v",rev["msg"])
	} else {
		fmt.Printf("      [msg]:%v\n",rev["msg"])
	}
	//fmt.Printf("----------------------------------------------\n")

	cl.Processing = false	
	cl.ClientChan <- string(data)
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

func (cl *Client) Printf(str string){
	fmt.Printf("Client[%v]%v\n",cl.Server,str)	
}

func (cl *Client) checkConneciton(_close bool,_connected bool) {
	cl.Close = _close
	cl.Connected = _connected
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
		fmt.Printf("----------------Client[%v]-------------\n",cl.Server)	
		fmt.Printf("[type]:%v\n","Error")
		fmt.Printf("[msg]:%v\n",err)
		fmt.Printf("----------------------------------------------\n")
		cl.checkConneciton(true, false)	
		cl.ClientChan <- "connect failed"
		return
	}
   	
	cl.checkConneciton(false, true)
	cl.Init(con)
	msg := fmt.Sprintf("action:login,user:%s,pwd:%s",cl.User,cl.Pwd)
	cl.Write([]byte(msg))
	go cl.Read()
}