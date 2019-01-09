import socket
import sys, os

gotMail = {}

def connect(ip, port, user, pwd):
    req = [
        "USER {0}\n".format(user).encode(),
        "PASS {0}\n".format(pwd).encode(),
    ]
    resp = ''
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((ip, port))
    resp = s.recv(1024).decode()
    print('S: %s' % resp.decode())
    if resp != 'OK':
        s.close()
        return None
    for r in req:
        s.send(r)
        resp = s.recv(1024).decode()
        if resp != 'OK':
            print(resp)
            s.close()
            return None

    return s

def listMail(s):
    ls = list()
    s.send("UIDL\n")
    r = s.recv(1024).decode()
    r = request(ip, port, user, pwd, 'UIDL')
    if 'OK' not in r:
        print(r)
        return ls
    a = r.split('\n')
    if len(a) == 1:
        return ls

    for m in r.split('\n')[1:-2]:
        ls.append(m.split(' '))

    return ls

def retrMail(s, msg, path):
    s.send('RETR %s\n' % msg)
    r = s.recv(1024).decode() # TODO may longer than buffer
    if 'OK' not in r:
        print(r)
        return
    with open(path, 'w') as f:
        if r.find('\n') > len(r)-1:
            f.write(r[r.find('\n')+1:])

def delMail(s, msg):
    s.send('DELE %s\n' % msg)
    r = s.recv(1024).decode()
    if 'OK' not in r:
        print(r)

if __name__ == '__main__':
    if len(sys.argv) < 6:
        print("Usage: ./clent ip port user pwd cmd <arg1> <arg2>")

    ip = sys.argv[1]
    port = int(sys.argv[2])
    user = sys.argv[3]
    pwd = sys.argv[4]
    cmd = sys.argv[5].lower()

    sock = connect(ip, port, user, pwd)
    if sock == None:
        exit(0)

    if cmd == 'list':
        a = listMail(sock)
        for m in a:
            print(m[0], m[1])
    elif cmd == 'count':
        a = listMail(sock)
        print('%d messages' % len(a))
    elif cmd == 'get':
        if len(sys.argv) < 8:
            print("should give message number and file path")
            exit(0)
        msg = sys.argv[6]
        output = sys.argv[7]
        retrMail(sock, msg, output)
    elif cmd == 'getall':
        if len(sys.argv) < 7:
            print("should give dir path")
            exit(0)
        Dir = sys.argv[6]
        msgs = listMail(sock)
        if len(msgs) == 0:
            exit(0)
        
        for m in msgs:
            retrMail(sock, m[0], Dir+"/"+m[1])
    elif cmd == 'delete':
        if len(sys.argv) < 7:
            print("should give message number")
            exit(0)
        msg = sys.argv[6]
        delMail(sock, msg)
        
    elif cmd == 'deleteall':
        msgs = listMail(sock)
        for m in msgs:
            delMail(sock, m[0])
