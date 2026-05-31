import socket, time

def recv(s, wait=5):
    data = b""
    s.settimeout(0.5)
    start = time.time()
    while time.time() - start < wait:
        try:
            c = s.recv(8192)
            if c: data += c
        except: time.sleep(0.05)
    return data.decode("utf-8", errors="replace")

s = socket.socket()
s.connect(("136.1.1.100", 23))
time.sleep(1); recv(s,1)
s.sendall(b"salfanet\r\n"); time.sleep(0.5); recv(s,0.5)
s.sendall(b"seven789\r\n"); time.sleep(1); recv(s,1)
s.sendall(b"terminal length 0\r\n"); time.sleep(1); recv(s,1)

# show running-config and search for DBA/traffic profile sections
s.sendall(b"show running-config\r\n"); time.sleep(8)
out = recv(s,8)
print("=== show running-config (first 4000 chars) ===")
print(out[:4000])
print("...")
print("=== searching for 'profile' in full output ===")
for line in out.split("\n"):
    if "profile" in line.lower():
        print(repr(line))
s.close()
