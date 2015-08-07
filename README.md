# Introduction:
  GoDeploy is simple deploy tool by Golang.
  
  It can help you to quickly send commands or file to your servers group.
  
  one client with mutli servers.
  
## Version:

version:0.0.3

## Futures

-config    Set cofing file path. Default value:./config.json

-debug     Show debug trace message. Default value:false

-mode      Service mode:server,client default:client

-group     Connect specific group servers

-server    Connect specific server

-version   Show version

-load      Load script and run,with exit

-help      Show help information
  
## Install:
```sh
  go get github.com/matishsiao/GoDeploy
  go test
  go build
```

## Configuration format:
```sh
    {
	"Configs":{
		"Server":[			
			{"Ip":"10.7.9.90","Group":"dev"},
			{"Ip":"10.7.9.83","Group":"dev"},
			{"Ip":"10.7.9.163","Group":"prd"}
		],
		"ServerPort":"9000",
		"User":"abc",
		"Password":"abc123"
	}
   }
```

## Run:

   Client mode(default mode)
```sh
   ./GoDeploy
```

   Server mode
```sh
   ./GoDeploy -mode server
```

   Get more informatiion
   
```sh
   ./GoDeploy -help 
```

## Client Example:

```sh   
   cmd ls
   cmd whoami
   file test.txt
   script add.dsh
   help
   status
   reconnect
```

## Commands:
	
	1.cmd:		Send command to server.
	
       			example:cmd ls
       			
	2.env:		Show all server os information.
	
	3.exit:		Exit appclication.
	
	4.file: 	Send file to server,It will save to server site file/ directory.
	
       			example:file test.txt
	
	5.get:		get file from all connect servers,the file will save to file/.
	
	6.help:		Show help information.
	
	7.script: 	Use script to run commands.
	
	  	    	example:script test.dsh
	  	    	
	8.status:	Show all server status.
	
##License and Copyright

This software is Copyright 2012-2014 Matis Hsiao.
