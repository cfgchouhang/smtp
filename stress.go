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
	"time"
)

var wg sync.WaitGroup

type Job struct {
	mail string
	rcpt string
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
		time.Sleep(0 * time.Millisecond)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		jobChan <- Job{dir + file.Name(), rcpt}
	}
	close(jobChan)
	wg.Wait()
}

func worker(id int, jobChan <-chan Job, tcpAddr *net.TCPAddr) {
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	resp := make([]byte, 1024)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	n, _ := conn.Read(resp)
	fmt.Println(string(resp[:n]))

	sendCommand("helo test.com\r\n", resp, conn)
	for job := range jobChan {
		fmt.Printf("worker%d starts sending mail %s\n", id, job.mail)
		sendMail(conn, job.mail, job.rcpt)
		fmt.Printf("worker%d finishs sending mail %s\n", id, job.mail)
		sendCommand("rset\r\n", resp, conn)
	}
	sendCommand("quit\r\n", resp, conn)

	wg.Done()
}

func sendCommand(cmd string, resp []byte, conn *net.TCPConn) bool {
	n, err := conn.Write([]byte(cmd))
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	n, err = conn.Read(resp)
	r := string(resp[:n])
	fmt.Println(r)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

func sendMail(conn *net.TCPConn, mailFile string, rcpt string) {
	var err error
	resp := make([]byte, 1024)
	cmdSet := []string{"helo test.com\r\n", "mail from:<>\r\n",
		"rcpt to: <" + rcpt + ">\r\n", "data\r\n"}

	for _, cmd := range cmdSet {
		if !sendCommand(cmd, resp, conn) {
			return
		}
	}

	file, err := os.Open(mailFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		conn.Write([]byte(strings.Trim(line, "\r\n") + "\r\n"))
	}
	sendCommand(".\r\n", resp, conn)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
