package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var wg sync.WaitGroup

type Job struct {
	tcpAddr *net.TCPAddr
	mail    string
	rcpt    string
}

func main() {
	max_chan := 4
	if len(os.Args) < 5 {
		fmt.Println("usgae: ./go ip port rcpt mail_dir [max_threads]")
		return
	}
	service := os.Args[1] + ":" + os.Args[2]
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err)

	files, err := ioutil.ReadDir(os.Args[4])
	if err != nil {
		log.Fatal(err)
	}

	dir := os.Args[4] + "/"
	rcpt := os.Args[3]
	if len(os.Args) == 6 {
		max_chan, _ = strconv.Atoi(os.Args[5])
	}

	jobChan := make(chan Job, max_chan)

	wg.Add(1)
	go worker(jobChan)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		jobChan <- Job{tcpAddr, dir + file.Name(), rcpt}
	}
	close(jobChan)

	wg.Wait()
}

func worker(jobChan <-chan Job) {
	defer wg.Done()
	for job := range jobChan {
		sendMail(job.tcpAddr, job.mail, job.rcpt)
	}
}

func sendMail(tcpAddr *net.TCPAddr, mailFile string, rcpt string) {
	var conn *net.TCPConn
	var err error
	resp := make([]byte, 1024)
	retryCnt := 0
	cmdSet := []string{"helo test.com\r\n", "mail from:<>\r\n",
		"rcpt to: <" + rcpt + ">\r\n", "data\r\n"}

	for retryCnt < 4 {
		conn, err = net.DialTCP("tcp", nil, tcpAddr)
		if err == nil {
			break
		}
		fmt.Println(err.Error())
		retryCnt++
	}
	if conn == nil {
		return
	}
	n, _ := conn.Read(resp)
	fmt.Println(string(resp[:n]))
	for _, cmd := range cmdSet {
		fmt.Print(cmd)
		_, err = conn.Write([]byte(cmd))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		n, _ = conn.Read(resp)
		fmt.Println(string(resp[:n]))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	file, err := os.Open(mailFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	i := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		conn.Write([]byte(strings.Trim(line, "\r\n") + "\r\n"))
		//fmt.Print(i, line)
		i++
	}
	conn.Write([]byte(".\r\n"))
	n, _ = conn.Read(resp)
	fmt.Println(string(resp[:n]))
	conn.Write([]byte("quit\r\n"))
	n, _ = conn.Read(resp)
	fmt.Println(string(resp[:n]))
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
