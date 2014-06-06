# Introduction:
  GoDeploy is simple deploy tool by Golang.
  
  It can help you to quickly send command or file to your server groups.
  
  one client wtih mutli server.
  
## Version:

version:0.0.2

## Futures

  
  
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
	  	"ServerIP":[	//Deploy receive server ip list		
  			"10.7.9.83",
			  "10.7.9.90"
		  ],
		  "ServerPort":"9000",//receive server port
		  "User":"abc",//user account
		  "Password":"abc123"//user password
	  }
  }
```

## Run:

   Client mode
```sh
   ./GoDeploy -mode client
```

   Server mode
```sh
   ./GoDeploy
```

   Get more informatiion
   
```sh
   ./GoDeploy -help 
```

## Example:

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

   1.cmd: Send command to server.
    
   example:
    
        cmd ls
        
        cmd df -h
        
   2.file: Send file to server.
    
   example:
        
        file test.txt
        
        file /var/tmp/test.txt //this file will save to server same directory.
    
   3.script: Use script to run commands.
    
   example:
    
        script test.dsh
    
   4.status: Show all server status.
    
   5.help: Show help information.
    
   6.exit: Exit appclication.

##License and Copyright
This software is Copyright 2012-2014 Matis Hsiao.
