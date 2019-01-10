package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SessionInfo struct {
	from, rcpt        string
	greeting, dataing bool
	needReset         bool
	mail              *os.File
}

type Job struct {
	conn net.Conn
}

var (
	logFile, _ = os.OpenFile("pop.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	logger     = log.New(logFile, "", 0)
	dir        = ""
)

func main() {
	max_conns := 2
	if len(os.Args) < 4 {
		fmt.Println("usage: ./go port auth_file mail_dir [max_sessions]")
		return
	}

	service := ":" + os.Args[1]
	dir = os.Args[2]
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err, true)
	defer logFile.Close()

	if len(os.Args) == 5 {
		max_conns, _ = strconv.Atoi(os.Args[4])
	}

	printLog("server starts")

	jobChan := make(chan Job, max_conns)
	releaseChan := make(chan Job, max_conns)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	go func() {
		<-sigc
		close(jobChan)
		printLog("server closed")
		os.Exit(0)
	}()

	for m := 1; m <= max_conns; m++ {
		go worker(jobChan, releaseChan)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err, true)
	for {
		conn, err := listener.Accept()
		if err != nil {
			checkError(err, false)
			continue
		}
		job := Job{conn, true}
		if tryEnqueue(job, releaseChan) {
			jobChan <- job
		} else {
			printLog("session is full, reject client: " + conn.RemoteAddr().String())
			conn.Write([]byte("ERR reached session limits, try latter\n"))
			conn.Close()
		}
	}
}

func tryEnqueue(job Job, Chan chan<- Job) bool {
	select {
	case Chan <- job:
		return true
	default:
		return false
	}
}
func worker(jobChan <-chan Job, releaseChan <-chan Job) {
	for job := range jobChan {
		handleClient(job.conn)
		<-releaseChan
	}
}

func getAuth(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	m := make(map[string]string)
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		tmp := strings.Split(line, " ")
		m[tmp[0]] = tmp[1]
	}
	return m
}

func handleClient(conn net.Conn, auth map[string]string) {
	//conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	defer conn.Close()

	cmd, param, user := "", "", ""
	tmp := strings.ToLower(request)
	cmdSet := []string{"user", "pass", "stat",
		"list", "uidl", "retr", "dele", "quit"}
	state := "auth"

	conn.Write([]byte("+OK POP3 server ready\n"))
	printLog("incoming client: " + conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)
	for {
		cmd, param = "", ""
		request, err := reader.ReadString('\n')
		if err != nil {
			printLog(err.Error())
			break
		}
		request = strings.Trim(request, "\r\n")
		printLog("client request:" + request)
		tmp := strings.Split(request, " ")
		if len(tmp) > 1 {
			param = tmp[1]
		}

		for _, c := range cmdSet {
			if tmp == c {
				cmd = c
				break
			}
		}
		if cmd == "" {
			conn.Write([]byte("-ERR\n"))
			continue
		}
		if cmd == "user" {
			if state != "auth" {
				conn.Write([]byte("-ERR already authenticated\n"))
				continue
			} else if param == "" {
				conn.Write([]byte("-ERR wrong user\n"))
				continue
			}
			if val, ok := auth[param]; ok {
				user = param
				state = "authu"
				conn.Write([]byte("+OK\n"))
			} else {
				conn.Write([]byte("-ERR user not exists\n"))
			}
		} else if cmd == "pass" {
			if state != "authu" {
				conn.Write([]byte("-ERR wrong sequence\n"))
				continue
			}
			if auth[user] == param {
				conn.Write([]byte("+OK\n"))
				state = "trans"
			} else {
				conn.Write([]byte("+ERR wrong password\n"))
				state = "auth"
			}
		}

		if cmd == "quit" {
			break
		}
	}
	printLog("client closed session")
	if info.mail != nil {
		printLog("delete non-completed data: " + info.mail.Name())
		os.Remove(info.mail.Name())
	}
}

func checkError(err error, fatal bool) {
	if err != nil {
		if fatal {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(1)
		}
	}
}

func printLog(msg string) {
	fmt.Println(msg)
	logger.Println(msg)
}
