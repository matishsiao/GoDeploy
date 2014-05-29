package main

import(
	"net"
	"fmt"
	"strings"
	"time"
	"io/ioutil"
)

type Client struct {
	Server string
	Conn *net.TCPConn
	User string
	Pwd string
	Login bool
}

func (cl *Client) Init(conn *net.TCPConn) {
	cl.Conn = conn	
}

func (cl *Client) Write(data []byte) {
	if cl.Conn != nil {	
		cl.Conn.Write(data)
	} else {
		cl.Printf("Client Connection is nil.")
	}
}

func (cl *Client) Read() {
	buf := make([]byte,2048)
	for {
		n, error := cl.Conn.Read(buf)
	    if error != nil {
	     	return 
	    }
	    cl.Process(buf[:n])
    }
}

func (cl *Client) Process(data []byte) {			
	revMsg := strings.Split(string(data),`,`)
	cl.Printf(fmt.Sprintf("%v",string(data)))
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
				 if rev["status"] == "success" {
				 	cl.Login = true				 	
				 } else {
					cl.Conn.Close()			 
				 }
			case "server":
			
		}
		
	}
	//cl.Printf(fmt.Sprintf("client rev:%v\n",rev))
}

func (cl *Client) InputCmd(cmdStr string) {
	//cl.Printf(fmt.Sprintf("Input Rev:%v\n",cmdStr))	
	if cmdStr != "" {			
		cmd := cmdStr[:strings.Index(cmdStr," ")]
		switch cmd {
			case "file":				
				FileName := cmdStr[strings.Index(cmdStr," ")+1:]
				cl.Printf(fmt.Sprintf("send file:%v",FileName))			
				file, e := ioutil.ReadFile(FileName)
				if e != nil {
					fmt.Printf("Load config error: %v\n", e)
				}
				msg := fmt.Sprintf("action:%s,user:%s,cmd:%s,file:%s,size:%v",cmd,cl.User,"start",FileName,len(file))
				//cl.Printf(fmt.Sprintf("Input action:%s,user:%s,cmd:%s,file:%s,size:%v\n",cmd,cl.User,"start",FileName,len(file)))				
				cl.Write([]byte(msg))
				time.Sleep(1000 * time.Millisecond)
				cl.Write(file)										
			default:
				msg := fmt.Sprintf("action:%s,user:%s,cmd:%s",cmd,cl.User,cmdStr[strings.Index(cmdStr," ")+1:])
				cl.Write([]byte(msg))
		}
	}
}

func (cl *Client) Printf(str string){
	fmt.Printf("Client[%v]:%v\n",cl.Server,str)	
}

func (cl *Client) Connect(addr string) {
	serverAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		cl.Printf(fmt.Sprintf("Connect serverAddr Error:%v\n",err))	
		return
	}	
	con, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {		
		cl.Printf(fmt.Sprintf("Connect Error:%v\n",err))	
		return
	}
	cl.Init(con)
	msg := fmt.Sprintf("action:login,user:%s,pwd:%s",cl.User,cl.Pwd)
	cl.Write([]byte(msg))
	go cl.Read()
}