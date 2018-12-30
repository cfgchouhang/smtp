package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

type SessionInfo struct {
	from, rcpt        string
	greeting, dataing bool
}

const CRLF = "\n"

func main() {
	service := ":1031"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	checkError(err, true)

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
	conn.SetReadDeadline(time.Now().Add(1 * time.Minute))
	request := make([]byte, 1024)
	defer conn.Close()

	info := SessionInfo{"", "", false, false}
	conn.Write([]byte("220 smtp service\n"))
	for {
		readLen, err := conn.Read(request)
		fmt.Println("request: " + string(request[:readLen]))
		if err != nil {
			checkError(err, false)
			break
		}
		if readLen == 0 {
			break
		}

		cmd, param, msg := parseRequest(string(request[:readLen]), &info)
		if msg != "" {
			conn.Write([]byte(msg))
			continue
		}
		fmt.Printf("cmd:%s, param:%s, msg:%s\n", cmd, param, msg)

		msg = checkRequest(cmd, param, &info)
		if msg != "" {
			conn.Write([]byte(msg))
			continue
		}

		/*msg = doRequest(cmd, param, &info)
		  conn.Write([]byte(msg))*/
	}
	fmt.Print("connection close\n")
}

func doRequest(cmd string, param string, info *SessionInfo) string {
	msg := ""
	if cmd == "helo" {
	} else if cmd == "rset" {
	} else if cmd == "mail from" {
	} else if cmd == "rcpt to" {
	} else if cmd == "data" {
	} else if cmd == "quit" {
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
		msg = checkAddress(param)
	} else if cmd == "rcpt to" {
		if info.from == "" {
			msg = ""
			goto END
		}
		msg = checkAddress(param)
	}

END:
	return msg
}

func checkAddress(param string) string {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	if !strings.HasPrefix(param, ":") {
		return "555 Syntax error\n"
	}

	param = strings.Trim(param, ": ")
	if param[0] != '<' || param[len(param)-1] != '>' {
		return "555 Syntax error\n"
	}

	param = strings.Trim(param, "<>")
	if !re.MatchString(param) {
		return "550 Invalid address\n"
	}

	return ""
}

func parseRequest(request string, info *SessionInfo) (string, string, string) {
	cmd, param, msg := "", "", ""
	tmp := strings.ToLower(request)
	cmdSet := []string{"helo", "mail from", "rcpt to",
		"data", "." + CRLF, "rset"}

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
