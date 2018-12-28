package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main(argc int, argv []string) int {
	service := ":7788"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err, true)
	for {
		conn, err := net.Accept()
		if err != nil {
			checkError(err, false)
			continue
		}

	}
}

func checkError(err error, fatal bool) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		if fatal {
			os.Exit(1)
		}
	}
}
