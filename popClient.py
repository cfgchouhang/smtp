import socket
import sys, os

gotMail = {}

def my_print(s):
    print(s, end='')

def connect(ip, port, user, pwd):
    req = [
        "USER {0}\n".format(user).encode(),
        "PASS {0}\n".format(pwd).encode(),
    ]
    resp = ''
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((ip, port))
    resp = s.recv(1024).decode()
    my_print('S: %s' % resp)
    if 'OK' not in resp:
        s.close()
        return None
    for r in req:
        s.send(r)
        resp = s.recv(1024).decode()
        if 'OK' not in resp:
            my_print("S: %s" % resp)
            s.close()
            return None

    return s

def request(sock, cmd, multi_line=False):
    sock.send(b"{0}\n".format(cmd))
    r = ''
    while '\n' not in r:
        r += sock.recv(1024).decode()

    if 'OK' not in r:
        return r
    
    if multi_line:
        while '\n.\n' not in r:
            r += sock.recv(1024).decode()
    return r

def listMail(s):
    ls = list()
    s.send(b"UIDL\n")
    r = s.recv(1024).decode()
    
    while '\n' not in r:
        r += s.recv(1024).decode()

    if 'OK' not in r:
        my_print(r)
        return ls

    mails = int(r.split('\n')[0].split(' ')[1])

    a = r.split('\n')
    while len(a) != mails+3: # first response line + mails# lines + .
        r += s.recv(1024).decode()
        a = r.split('\n')
    

    #print(r)
    a = r.split('\n')
    if len(a) == 1:
        my_print(r)
        return ls

    for m in a[1:-2]:
        ls.append(m.split(' '))

    return ls

def retrMail(s, msg, path):
    s.send('RETR {0}\n'.format(msg).encode())
    r = ''
    while '\n' not in r:
        r += sock.recv(1024).decode()

    if 'OK' not in r:
        my_print("S: %s " % r)
        return
    
    while '\n.\n' not in r:
        r += sock.recv(1024).decode()

    with open(path, 'w') as f:
        a = r.split('\n')
        if len(a) > 2:
            my_print("get message %s\n%s" % (msg, r[r.find('\n')+1:-2]))
            f.write(r[r.find('\n')+1:-2])

def delMail(s, msg):
    s.send('DELE {0}\n'.format(msg).encode())
    r = s.recv(1024).decode()
    while '\n' not in r:
        r += sock.recv(1024).decode()
    my_print("S: %s" % r)

def quit(s):
    s.send(b'QUIT\n')
    r = s.recv(1024).decode()
    while '\n' not in r:
        r += sock.recv(1024).decode()
    my_print("S: %s" % r)

if __name__ == '__main__':
    if len(sys.argv) < 6:
        my_print("Usage: ./clent ip port user pwd cmd <arg1> <arg2>")
        exit(0)

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
        print("%d messages" % len(a))
        for m in a:
            print(m[0], m[1])
    elif cmd == 'count':
        a = listMail(sock)
        print('%d messages' % len(a))
    elif cmd == 'get':
        if len(sys.argv) < 8:
            my_print("should give message number and file path\n")
            exit(0)
        msg = int(sys.argv[6])
        path = sys.argv[7]
        retrMail(sock, msg, path)
    elif cmd == 'getall':
        if len(sys.argv) < 7:
            my_print("should give dir path\n")
            exit(0)
        Dir = sys.argv[6]
        msgs = listMail(sock)
        if len(msgs) == 0:
            print("no messages")
            exit(0)
        
        for m in msgs:
            retrMail(sock, m[0], Dir+"/"+m[1])
    elif cmd == 'delete':
        if len(sys.argv) < 7:
            my_print("should give message number")
            exit(0)
        msg = sys.argv[6]
        delMail(sock, msg)
        quit(sock)
    elif cmd == 'deleteall':
        msgs = listMail(sock)
        for m in msgs:
            delMail(sock, m[0])
        quit(sock)
    else:
        print("no such command")
