import socket
import sys, os

uids_file = "uids.txt"
uids = list()

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

def listMail(s):
    ls = list()
    s.send(b"UIDL\r\n")
    r = s.recv(1024).decode()
    
    while '\n' not in r:
        r += s.recv(1024).decode().replace('\r', '')

    if 'OK' not in r:
        my_print(r)
        return ls

    #while '\n.\n' not in r and '\r\n.\r\n' not in r:
    #    r += sock.recv(1024).decode().replace('\r', '')

    mails = int(r.split('\n')[0].split(' ')[1])

    a = r.split('\n')
    while mails > 0 and len(a) != mails+3: # first response line + mails# lines + .
        r += s.recv(1024).decode().replace('\r', '')
        a = r.split('\n')
    

    #print(r)
    a = r.split('\n')
    if len(a) == 1:
        my_print(r)
        return ls

    for m in a[1:-2]:
        a = m.split(' ')
        if a[1] in uids:
            continue
        ls.append(m.split(' '))

    return ls

def retrMail(s, msg, path, uid):
    s.send('RETR {0}\r\n'.format(msg).encode())
    r = ''
    while '\n' not in r:
        r += sock.recv(1024).decode().replace('\r', '')

    if 'OK' not in r:
        my_print("S: %s " % r)
        return
    
    while '\n.\n' not in r and '\n.\r\n' not in r:
        r += sock.recv(1024).decode().replace('\r', '')

    with open(path, 'w') as f:
        a = r.split('\n')
        if len(a) > 2:
            my_print("get message %s\n%s" % (msg, r[r.find('\n')+1:-2]))
            with open(uids_file, 'a') as t:
                t.write(uid+'\n')
            f.write(r[r.find('\n')+1:-2])

def delMail(s, msg):
    s.send('DELE {0}\r\n'.format(msg).encode())
    r = s.recv(1024).decode()
    while '\n' not in r:
        r += sock.recv(1024).decode()
    my_print("S: %s" % r)

def quit(s):
    s.send(b'QUIT\r\n')
    r = s.recv(1024).decode()
    while '\n' not in r:
        r += sock.recv(1024).decode()
    my_print("S: %s" % r)

if __name__ == '__main__':
    if len(sys.argv) < 6:
        print("Usage: ./clent ip port user pwd cmd <arg1> <arg2>")
        exit(0)

    ip = sys.argv[1]
    port = int(sys.argv[2])
    user = sys.argv[3]
    pwd = sys.argv[4]
    cmd = sys.argv[5].lower()

    sock = connect(ip, port, user, pwd)
    if sock == None:
        exit(0)

    uids_file += "."+user
    
    if os.path.exists(uids_file):
        with open(uids_file) as f:
            s = f.read()
            for u in s.split('\n'):
                uids.append(u)

    msgs = listMail(sock)
    if cmd == 'list':
        print("%d messages" % len(msgs))
        for m in msgs:
            print(m[0], m[1])
        quit(sock)
    elif cmd == 'count':
        print('%d messages' % len(msgs))
        quit(sock)
    elif cmd == 'get':
        if len(sys.argv) < 8:
            my_print("should give message number and file path\n")
            exit(0)
        msg = int(sys.argv[6])
        path = sys.argv[7]
        if msg > len(msgs):
            print("no %d message" % msg)
            exit(0)
        retrMail(sock, msg, path, msgs[msg-1][1])
        quit(sock)
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
            retrMail(sock, m[0], Dir+"/"+m[1], m[1])
        quit(sock)
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
