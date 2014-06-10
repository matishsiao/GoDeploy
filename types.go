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
}

type Configs struct {
	Configs ConfigsObject
}

type ConfigsObject struct {
	ServerIP	[]string
	ServerPort	string
	User		string
	Password	string
}

type FileObject struct {
	FileName string
	FileSize int64
	CmdIdx 	string
	Data	[]byte
}