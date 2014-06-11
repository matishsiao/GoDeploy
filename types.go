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
}

type Configs struct {
	Configs ConfigsObject
}

type ConfigsObject struct {
	Server	[]ServerNode
	ServerPort	string
	User		string
	Password	string
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