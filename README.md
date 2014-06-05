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

## Example:
```sh   
   cmd ls
   cmd whoami
   file test.txt
   script add.dsh
```

##License and Copyright
This software is Copyright 2012-2014 Matis Hsiao.
