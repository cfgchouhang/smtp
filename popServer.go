package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Job struct {
	conn net.Conn
}

type Mail struct {
	uid     string
	size    int
	deleted bool
}

var (
	logFile, _ = os.OpenFile("pop.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	logger     = log.New(logFile, "", 0)
	mailDir    = ""
	authPath   = ""
)

func main() {
	max_conns := 2
	if len(os.Args) < 4 {
		fmt.Println("usage: ./go port auth_file mail_dir [max_sessions]")
		return
	}

	service := ":" + os.Args[1]
	authPath = os.Args[2]
	mailDir = os.Args[3]
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

	auth := getAuth(authPath)

	for m := 1; m <= max_conns; m++ {
		go worker(jobChan, releaseChan, auth)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err, true)
	for {
		conn, err := listener.Accept()
		if err != nil {
			checkError(err, false)
			continue
		}
		job := Job{conn}
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
func worker(jobChan <-chan Job, releaseChan <-chan Job, auth map[string]string) {
	for job := range jobChan {
		handleClient(job.conn, auth)
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

func getList(dir string, mbox string) ([]Mail, int) {
	var mails []Mail
	size := 0
	files, err := ioutil.ReadDir(dir + "/" + mbox)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		mails = append(mails, Mail{file.Name(), int(file.Size()), false})
		size += int(file.Size())
	}
	return mails, size
}

func checkMsgNum(conn net.Conn, param string, total int) int {
	var err error
	i := -1
	if i, err = strconv.Atoi(param); err != nil {
		conn.Write([]byte("-ERR invalid argument\n"))
	} else if i >= total {
		conn.Write([]byte("-ERR no such message\n"))
	}
	return i
}
func handleClient(conn net.Conn, auth map[string]string) {
	//conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	defer conn.Close()

	var mails []Mail
	size := 0
	cmd, param, user := "", "", ""
	cmdSet := []string{"user", "pass", "stat",
		"list", "uidl", "retr", "dele", "quit"}
	state := "auth"

	conn.Write([]byte("+OK POP3 server ready\n"))
	printLog("incoming client: " + conn.RemoteAddr().String())
	mails, size = getList(mailDir, user)

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
		request = strings.ToLower(request)
		tmp := strings.Split(request, " ")
		if len(tmp) > 1 {
			param = tmp[1]
		}

		for _, c := range cmdSet {
			if tmp[0] == c {
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
			} else if param == "" {
				conn.Write([]byte("-ERR no password\n"))
				continue
			}
			if auth[user] == param {
				conn.Write([]byte("+OK\n"))
				state = "trans"
			} else {
				conn.Write([]byte("-ERR wrong password\n"))
				state = "auth"
			}
		} else if cmd == "stat" || cmd == "uidl" || cmd == "list" {
			if state != "trans" {
				conn.Write([]byte("-ERR invalid state\n"))
				continue
			}
			if cmd == "stat" {
				conn.Write([]byte(fmt.Sprintf("+OK %d %d\n", len(mails), size)))
			} else if cmd == "uidl" {
				if param != "" {
					i := 0
					if i = checkMsgNum(conn, param, len(mails)); i < 0 {
						continue
					}
					conn.Write([]byte(fmt.Sprintf("+OK %d %d\n", i, mails[i].uid)))
				} else {
					conn.Write([]byte("+OK\n"))
					for i, m := range mails {
						if !m.deleted {
							conn.Write([]byte(fmt.Sprintf("%d %d\n", i, m.uid)))
						}
					}
				}
			} else if cmd == "list" {
				if param != "" {
					i := 0
					if i = checkMsgNum(conn, param, len(mails)); i < 0 {
						continue
					}
					conn.Write([]byte(fmt.Sprintf("+OK %d %d\n", i, mails[i].size)))
				} else {
					conn.Write([]byte(fmt.Sprintf("+OK %d %d\n", len(mails), size)))
					for i, m := range mails {
						if !m.deleted {
							conn.Write([]byte(fmt.Sprintf("%d %d\n", i, m.size)))
						}
					}
				}
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
