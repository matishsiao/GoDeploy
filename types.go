package main

import (
	"time"
)

type ConfigInfo struct {
	FileName *string
	Size     int64
	ModTime  time.Time
	Debug    *bool
	Version  *bool
	Mode	*string	
	Load	*string
	Group	*string
	Server	*string
	Record 	*bool
}

type Configs struct {
	Configs ConfigsObject
}

type ConfigsObject struct {
	Server		ConfServer
	Auth		ConfAuth
	Client		ConfClient
}

type ConfServer struct {
	List	[]ServerNode
	Port	string
	Timeout int
}

type ConfAuth struct {
	User		string
	Password	string
}

type ConfClient struct {
	Timeout int
}

type ServerNode struct {
	Ip string
	Group string
}

type FileObject struct {
	FileName string
	FileSize int64
	CmdIdx 	string
	Data	[]byte
}