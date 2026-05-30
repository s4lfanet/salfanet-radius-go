import socket, time
s = socket.socket()
s.settimeout(10)
s.connect(('136.1.1.100', 23))
time.sleep(2)
try: data = s.recv(4096)
except: data = b''
print('BANNER:', repr(data[:200]))
s.send(b'salfanet\r\n')
time.sleep(1)
try: data = s.recv(4096)
except: data = b''
print('AFTER_USER:', repr(data[:200]))
s.send(b'seven789\r\n')
time.sleep(2)
try: data = s.recv(4096)
except: data = b''
print('AFTER_PASS:', repr(data[:200]))
s.send(b'show gpon onu detail-info gpon-onu_0/1/1:28\r\n')
time.sleep(4)
try: data = s.recv(8192)
except: data = b''
print('DETAIL_ONU28:', data.decode(errors='ignore'))
s.close()
