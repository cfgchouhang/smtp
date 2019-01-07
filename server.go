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
	conn  net.Conn
	valid bool
}

var (
	logFile, _ = os.OpenFile("smtp.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	logger     = log.New(logFile, "", 0)
	dir        = ""
)

func main() {
	max_conns := 2
	if len(os.Args) < 3 {
		fmt.Println("usage: ./go port mail_dir [max_sessions]")
		return
	}

	service := ":" + os.Args[1]
	dir = os.Args[2]
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err, true)
	defer logFile.Close()

	if len(os.Args) == 4 {
		max_conns, _ = strconv.Atoi(os.Args[3])
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
			conn.Write([]byte("421 reached session limits, try latter\n"))
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

func resetSession(info *SessionInfo) {
	info.from = ""
	info.rcpt = ""
	info.dataing = false
	info.needReset = false
	info.mail = nil
}

func handleClient(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	defer conn.Close()

	info := SessionInfo{"", "", false, false, false, nil}
	resetSession(&info)
	conn.Write([]byte("220 smtp service\n"))
	printLog("incoming client: " + conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)
	for {
		request, err := reader.ReadString('\n')
		if err != nil {
			printLog(err.Error())
			break
		}
		printLog("client request:" + request)

		cmd, param, msg := parseRequest(request, &info)
		if msg != "" {
			printLog(msg)
			conn.Write([]byte(msg + "\n"))
			continue
		}
		logger.Printf("cmd:%s, param:%s, msg:%s\n", cmd, param, msg)

		msg = checkRequest(cmd, param, &info)
		if msg != "" {
			printLog(msg)
			conn.Write([]byte(msg + "\n"))
			continue
		}

		msg = doRequest(cmd, param, &info)
		if msg != "" {
			printLog(msg)
			conn.Write([]byte(msg + "\n"))
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

func doRequest(cmd string, param string, info *SessionInfo) string {
	msg := ""
	if cmd == "helo" {
		info.greeting = true
		printLog("client greeting" + strings.TrimSpace(param))
		msg = "250 service starts"
	} else if cmd == "rset" {
		resetSession(info)
		msg = "250 OK, session reset"
	} else if cmd == "mail from" {
		info.from = strings.Trim(param, ": ")
		msg = "250 OK"
	} else if cmd == "rcpt to" {
		info.rcpt = strings.Trim(param, ": ")
		msg = "250 OK"
	} else if cmd == "data" {
		if !info.dataing {
			info.dataing = true
			msg = "354 Start mail input"
			mailFile, err := os.Create(dir + "/" + time.Now().Format("20060102_150405.000000.eml"))
			if err != nil {
				checkError(err, true)
			}
			info.mail = mailFile
			//fmt.Fprintf(info.mail, "MAIL FROM:%s\nRCPT TO:%s\n", info.from, info.rcpt)
		} else {
			logger.Println(param)
			info.mail.WriteString(param + "\r\n")
		}
	} else if cmd == "end" {
		info.needReset = true
		info.dataing = false
		info.mail.Close()
		info.mail = nil
		msg = "250 Mail data transfer completed"
	} else if cmd == "quit" {
		msg = "221 Close connection"
	}
	return msg
}

func checkRequest(cmd string, param string, info *SessionInfo) string {
	msg := ""

	if !info.greeting && cmd != "helo" && cmd != "quit" {
		msg = "503 HELO first"
	} else if cmd == "helo" {
		if !strings.HasPrefix(param, " ") || strings.TrimSpace(param) == "" {
			msg = "501 Invalid argument"
			goto END
		}
	} else if cmd == "mail from" {
		msg = checkAddress(param, true)
	} else if cmd == "rcpt to" {
		if info.from == "" {
			msg = "503 bad sequence of commands"
			goto END
		}
		msg = checkAddress(param, false)
	} else if cmd == "data" {
		if info.from == "" {
			msg = "503 MAIL first"
		} else if info.rcpt == "" {
			msg = "503 RCPT first"
		}
	}

END:
	return msg
}

func checkAddress(param string, canEmpty bool) string {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	if !strings.HasPrefix(param, ":") {
		return "555 Syntax error"
	}

	param = strings.Trim(param, ": ")
	if param[0] != '<' || param[len(param)-1] != '>' {
		return "555 Syntax error"
	}

	param = strings.Trim(param, "<>")
	if canEmpty && param == "" {
	} else if !re.MatchString(param) {
		return "550 Invalid address"
	}

	return ""
}

func parseRequest(request string, info *SessionInfo) (string, string, string) {
	cmd, param, msg := "", "", ""
	tmp := strings.ToLower(request)
	cmdSet := []string{"helo", "mail from", "rcpt to",
		"data", "rset", "quit"}

	if !strings.HasSuffix(request, "\r\n") &&
		!strings.HasSuffix(request, "\n") {
		msg = "555 Syntax error"
		goto END
	}

	if info.dataing {
		if tmp != ".\r\n" && tmp != ".\n" {
			param = strings.Trim(request, "\r\n")
			return "data", param, ""
		}
		return "end", "", ""
	}

	for _, c := range cmdSet {
		if strings.HasPrefix(tmp, c) {
			cmd = c
			param = strings.Trim(request[len(c):], "\r\n")
		}
	}

	if cmd == "" {
		if strings.Contains(tmp, "mail") ||
			strings.Contains(tmp, "rcpt") {
			msg = "555 Syntax error"
		} else {
			msg = "502 Unrecognized coomand"
		}
	}
	if info.needReset && cmd != "rset" && cmd != "quit" {
		msg = "503 Reset session first"
	}

END:
	return cmd, param, msg
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
