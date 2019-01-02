package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

type SessionInfo struct {
	from, rcpt        string
	greeting, dataing bool
	mail              *os.File
}

var (
	logFile, _ = os.OpenFile("smtp.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	logger     = log.New(logFile, "", 0)
)

const CRLF = "\n"

func main() {
	service := ":1031"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err, true)
	defer logFile.Close()

	logger.Println("server starts")

	listener, err := net.ListenTCP("tcp", tcpAddr)
	checkError(err, true)
	for {
		conn, err := listener.Accept()
		if err != nil {
			checkError(err, false)
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	request := make([]byte, 1024)
	defer conn.Close()

	t := time.Now()
	mailFile, err := os.Create(t.Format("20060102_150405.000000"))
	if err != nil {
		logger.Println("cannot open mail file to write")
		return
	}
	defer mailFile.Close()

	info := SessionInfo{"", "", false, false, mailFile}
	conn.Write([]byte("220 smtp service\n"))
	for {
		readLen, err := conn.Read(request)
		fmt.Println("request: " + string(request[:readLen]))
		logger.Println("request: " + string(request[:readLen]))
		if err != nil {
			checkError(err, false)
			break
		}
		if readLen == 0 {
			break
		}

		cmd, param, msg := parseRequest(string(request[:readLen]), &info)
		if msg != "" {
			logger.Println(msg)
			conn.Write([]byte(msg))
			continue
		}
		fmt.Printf("cmd:%s, param:%s, msg:%s\n", cmd, param, msg)
		logger.Printf("cmd:%s, param:%s, msg:%s\n", cmd, param, msg)

		msg = checkRequest(cmd, param, &info)
		if msg != "" {
			logger.Println(msg)
			conn.Write([]byte(msg))
			continue
		}

		msg = doRequest(cmd, param, &info)
		conn.Write([]byte(msg))
		logger.Println(msg)
		if cmd == "quit" {
			break
		}
	}
	fmt.Println("client close")
}

func doRequest(cmd string, param string, info *SessionInfo) string {
	msg := ""
	if cmd == "helo" {
		info.greeting = true
		msg = "250 service starts\n"
	} else if cmd == "rset" {
		info.from = ""
		info.rcpt = ""
		info.dataing = false
	} else if cmd == "mail from" {
		info.from = strings.Trim(param, ": <>")
		msg = "250 OK\n"
	} else if cmd == "rcpt to" {
		info.rcpt = strings.Trim(param, ": <>")
		msg = "250 OK\n"
	} else if cmd == "data" {
		if info.dataing {
			info.dataing = true
		} else {
			// output to email file
		}
		msg = "250 OK\n"
	} else if cmd == "quit" {
		msg = "221 close connection"
	} else if cmd == "."+CRLF {
	}
	return msg
}

func checkRequest(cmd string, param string, info *SessionInfo) string {
	msg := ""

	if !info.greeting && cmd != "helo" {
		msg = "503 HELO first\n"
	} else if cmd == "helo" {
		if param == "" || !strings.HasPrefix(param, " ") {
			msg = "501 Invalid argument\n"
			goto END
		}
		fmt.Println("client from " + strings.TrimSpace(param))
	} else if cmd == "mail from" {
		msg = checkAddress(param, true)
	} else if cmd == "rcpt to" {
		if info.from == "" {
			msg = "503 bad sequence of commands\n"
			goto END
		}
		msg = checkAddress(param, false)
	} else if cmd == "data" {
		if info.from == "" {
			msg = "503 MAIL first\n"
		} else if info.rcpt == "" {
			msg = "503 RCPT first\n"
		}
	}

END:
	return msg
}

func checkAddress(param string, canEmpty bool) string {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	if !strings.HasPrefix(param, ":") {
		return "555 Syntax error\n"
	}

	param = strings.Trim(param, ": ")
	if param[0] != '<' || param[len(param)-1] != '>' {
		return "555 Syntax error\n"
	}

	param = strings.Trim(param, "<>")
	if canEmpty && param == "" {
	} else if !re.MatchString(param) {
		return "550 Invalid address\n"
	}

	return ""
}

func parseRequest(request string, info *SessionInfo) (string, string, string) {
	cmd, param, msg := "", "", ""
	tmp := strings.ToLower(request)
	cmdSet := []string{"helo", "mail from", "rcpt to",
		"data", "." + CRLF, "rset", "quit"}

	if !strings.HasSuffix(request, CRLF) {
		msg = "555 Syntax error\n"
		goto END
	}

	if info.dataing {
		return "data", request, ""
	}

	for _, c := range cmdSet {
		if strings.HasPrefix(tmp, c) {
			cmd = c
			param = request[len(c) : len(request)-len(CRLF)]
		}
	}

	if cmd == "" {
		if strings.Contains(tmp, "mail") ||
			strings.Contains(tmp, "rcpt") {
			msg = "555 Syntax error\n"
		} else {
			msg = "502 Unrecognized coomand\n"
		}
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
