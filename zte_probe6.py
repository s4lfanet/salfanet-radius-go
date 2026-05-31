import socket, time

def recv(s, wait=2):
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
s.settimeout(3)
time.sleep(1); recv(s,1)
s.sendall(b"salfanet\r\n"); time.sleep(0.5); recv(s,0.5)
s.sendall(b"seven789\r\n"); time.sleep(1); recv(s,1)
s.sendall(b"terminal length 0\r\n"); time.sleep(1); recv(s,1)

# Try tab-complete style to discover commands for gpon onu
s.sendall(b"show gpon onu ?\r\n"); time.sleep(2)
out = recv(s,2)
print("=== show gpon onu ? ===")
print(out[:3000])

s.sendall(b"show gpon onu traffic-statistic gpon-onu_1/1/1:1\r\n"); time.sleep(2)
out2 = recv(s,2)
print("=== show gpon onu traffic-statistic ===")
print(out2[:2000])

s.sendall(b"show gpon service-port gpon-onu_1/1/1:1\r\n"); time.sleep(2)
out3 = recv(s,2)
print("=== show gpon service-port ===")
print(out3[:2000])

s.close()
