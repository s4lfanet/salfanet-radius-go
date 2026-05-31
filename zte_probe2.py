import socket, time, sys

def recv_all(s, wait=2):
    data = b''
    s.settimeout(0.5)
    start = time.time()
    while time.time() - start < wait:
        try:
            chunk = s.recv(4096)
            if chunk: data += chunk
        except: time.sleep(0.05)
    return data.decode('utf-8', errors='replace')

s = socket.socket()
s.connect(('136.1.1.100', 23))
s.settimeout(3)
time.sleep(1)
recv_all(s, 1)
s.sendall(b'salfanet\r\n'); time.sleep(0.5)
recv_all(s, 0.5)
s.sendall(b'seven789\r\n'); time.sleep(1)
recv_all(s, 1)
s.sendall(b'terminal length 0\r\n'); time.sleep(1)
recv_all(s, 1)
s.sendall(b'show gpon onu detail-info gpon-onu_1/1/1:1\r\n'); time.sleep(3)
out = recv_all(s, 3)
print('=DETAIL=')
print(out[:4000])
s.sendall(b'show running-config interface gpon-onu_1/1/1:1\r\n'); time.sleep(3)
out2 = recv_all(s, 3)
print('=RUNCONFIG=')
print(out2[:3000])
s.sendall(b'show interface gpon-onu_1/1/1:1\r\n'); time.sleep(2)
out3 = recv_all(s, 2)
print('=INTERFACE=')
print(out3[:2000])
s.close()
