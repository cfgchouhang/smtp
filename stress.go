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
	max_workers := 2
	max_jobs := 100
	if len(os.Args) < 5 {
		fmt.Println("usgae: ./go ip port rcpt mail_dir [max_workers]")
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
		max_workers, _ = strconv.Atoi(os.Args[5])
	}

	jobChan := make(chan Job, max_jobs)

	for w := 1; w <= max_workers; w++ {
		wg.Add(1)
		go worker(w, jobChan, tcpAddr)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		jobChan <- Job{tcpAddr, dir + file.Name(), rcpt}
	}
	close(jobChan)
	wg.Wait()
}

func worker(id int, jobChan <-chan Job) {
	conn, err = net.DialTCP("tcp", nil, job.tcpAddr)
	if err == nil {
		fmt.Println(err.Error())
		return
	}
	n, _ := conn.Read(resp)
	fmt.Println(string(resp[:n]))

	for job := range jobChan {
		fmt.Printf("worker%d starts sending mail %s\n", id, job.mail)
		sendMail(conn, job.mail, job.rcpt)
		fmt.Printf("worker%d finishs sending mail %s\n", id, job.mail)
	}
	conn.Write([]byte("quit\r\n"))
	n, _ = conn.Read(resp)
	fmt.Println(string(resp[:n]))

	wg.Done()
}

func sendMail(conn *net.TCPConn, mailFile string, rcpt string) {
	var conn *net.TCPConn
	var err error
	resp := make([]byte, 1024)
	cmdSet := []string{"helo test.com\r\n", "mail from:<>\r\n",
		"rcpt to: <" + rcpt + ">\r\n", "data\r\n"}

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
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
