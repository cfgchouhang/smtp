import datetime,os,random,string

for i in range(10):
    a = ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(5))
    with open(a+".mail", 'w') as f:
        f.write("msg "+str(i)+"\n")
        f.write("中文\n")
        for j in range(random.randint(2, 10)):
            f.write(''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(8))+"\n")
