import socket, time

def recv(s, wait=3):
    data = b""
    s.settimeout(0.5)
    start = time.time()
    while time.time() - start < wait:
        try:
            c = s.recv(4096)
            if c: data += c
        except: time.sleep(0.05)
    return data.decode("utf-8", errors="replace")

s = socket.socket()
s.connect(("136.1.1.100", 23))
s.settimeout(5)
time.sleep(1); recv(s,1)
s.sendall(b"salfanet\r\n"); time.sleep(0.5); recv(s,0.5)
s.sendall(b"seven789\r\n"); time.sleep(1); recv(s,1)
s.sendall(b"terminal length 0\r\n"); time.sleep(1); recv(s,1)

cmds = [
    "show running-config pon-onu-mng gpon-onu_1/1/1:1",
    "show running-config interface gpon-olt_1/1/1",
    "show running-config interface gpon-onu_1/1/1:1",
    "show gpon onu detail-info gpon-onu_1/1/1:1",
]
for cmd in cmds:
    s.sendall((cmd + "\r\n").encode()); time.sleep(3)
    out = recv(s,3)
    print(f"\n=== {cmd} ===")
    print(out[:3000])

s.close()
