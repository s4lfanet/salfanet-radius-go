import socket, time

def recv(s, wait=2):
    data = b''
    s.settimeout(0.5)
    start = time.time()
    while time.time() - start < wait:
        try:
            c = s.recv(4096)
            if c: data += c
        except: time.sleep(0.05)
    return data.decode('utf-8', errors='replace')

s = socket.socket()
s.connect(('136.1.1.100', 23))
s.settimeout(3)
time.sleep(1); recv(s, 1)
s.sendall(b'salfanet\r\n'); time.sleep(0.5); recv(s, 0.5)
s.sendall(b'seven789\r\n'); time.sleep(1); recv(s, 1)
s.sendall(b'terminal length 0\r\n'); time.sleep(1); recv(s, 1)

cmds = [
    'show gpon onu tr069-mode gpon-onu_1/1/1:1',
    'show gpon onu optical-info gpon-olt_1/1/1',
    'show running-config interface gpon-olt_1/1/1',
    'show qos policy DOWNSTREAM',
]
for cmd in cmds:
    s.sendall((cmd + '\r\n').encode()); time.sleep(2)
    out = recv(s, 2)
    print(f'=== {cmd} ===')
    print(out[:1500])
s.close()