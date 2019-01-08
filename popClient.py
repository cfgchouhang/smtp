import socket
import sys, os

def request(ip, port, user, pwd, cmd):
    req = [
        "USER {0}\n".format(user).encode(),
        "PASS {0}\n".format(pwd).encode(),
        "{0}\n".format(cmd)
    ]
    resp = ''
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.connect((ip, port))
        resp = s.recv(1024).decode()
        print('S: %s' % resp.decode())
        if resp != 'OK':
            return resp
        for r in req:
            s.send(r)
            resp = s.recv(1024).decode()
            if resp != 'OK':
                return resp
    
    return resp


if __name__ == '__main__':
    if len(sys.argv) < 6:
        print("Usage: ./clent ip port user pwd cmd <arg1> <arg2>")

    ip = sys.argv[1]
    port = int(sys.argv[2])
    user = sys.argv[3]
    pwd = sys.argv[4]
    cmd = sys.argv[5].lower()

    if cmd == 'count':
    elif cmd == 'get':
        if len(sys.argv) < 7:
            print("should give message id")
            exit(0)
        msg = sys.argv[6]
        request(ip, port, user, pwd, 'LIST')
    elif cmd == 'getall':
    elif cmd == 'delete':
    elif cmd == 'deleteall':
