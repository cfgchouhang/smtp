#!/usr/bin/python3
import socket, dns.resolver
import sys, os

def get_mx(domain):
    min = 5000
    mx = None

    for x in dns.resolver.query(domain, 'MX'):
        a = x.to_text().split(' ')
        if int(a[0]) < min:
            min = int(a[0])
            mx = a[1]

    return mx

if __name__ == '__main__':
    if len(sys.argv) < 4:
        print('usage: ./client rcpt@domain from@domain mail_file')
        exit(0)
    
    From = sys.argv[2]
    rcpt = sys.argv[1]
    data = sys.argv[3]
    
    if not os.path.exists(data):
        print('mail file not exist')
        exit(0)

    with open(data) as f:
        data = f.read()
    
    mx = get_mx(rcpt[rcpt.find('@')+1:])

    ip = socket.gethostbyname(mx)
    print('F %s\nR %s' % (From, rcpt))
    print(mx, ip)

    commands = [
        b'HELO openfind.com\r\n',
        'MAIL FROM:<{0}>\r\n'.format(From).encode(),
        'RCPT TO:<{0}>\r\n'.format(rcpt).encode(),
        b'DATA\r\n',
    ]

    #data = 'From:""<sender@mydomain.com>\r\nTo:""< friend@example.com>\r\nSubject: test\r\n\r\nhello test\r\n'

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.connect((ip, 25))
        resp = s.recv(1024)
        print(resp)

        for act in commands:
            print('C: %s' % act)
            s.send(act)
            resp = s.recv(1024)
            print('S: %s' % resp)
        
        for line in data.split('\r\n'):
            s.send('{0}\r\n'.format(line).encode())
        s.send(b'\r\n.\r\n')
        resp = s.recv(1024)
        print('S: %s' % resp.decode())

        s.send(b'quit\r\n')
        resp = s.recv(1024)
        print('S: %s' % resp.decode())
