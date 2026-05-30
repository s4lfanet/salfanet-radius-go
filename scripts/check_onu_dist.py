import socket, time
s = socket.socket()
s.connect(('136.1.1.100', 23))
time.sleep(2)
s.recv(4096)
s.send(b'salfanet\r\n')
time.sleep(1)
s.recv(4096)
s.send(b'seven789\r\n')
time.sleep(2)
s.recv(4096)
s.send(b'show gpon onu detail-info gpon-onu_0/1/1:1\r\n')
time.sleep(3)
data = s.recv(8192).decode(errors='ignore')
s.send(b'show gpon onu detail-info gpon-onu_0/1/1:9\r\n')
time.sleep(3)
data2 = s.recv(8192).decode(errors='ignore')
s.close()
print('=== ONU 1 ===')
for line in data.split('\n'):
    if 'Distance' in line or 'distance' in line:
        print(repr(line))
print('=== ONU 9 ===')
for line in data2.split('\n'):
    if 'Distance' in line or 'distance' in line:
        print(repr(line))
