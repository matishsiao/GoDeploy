# Introduction:
  This GoDeploy is simple deploy tool by Golang.
  
## Version:

version:0.0.1

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
	  	"ServerIP":[	//Delpoy receive server ip list		
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
