import socket, time, sys

def telnet_cmd(s, cmd, wait=2):
    s.sendall((cmd + '\r\n').encode())
    time.sleep(wait)
    data = b''
    s.settimeout(0.5)
    while True:
        try:
            chunk = s.recv(4096)
            if chunk:
                data += chunk
            else:
                break
        except:
            break
    return data.decode('utf-8', errors='replace')

s = socket.socket()
s.connect(('136.1.1.100', 23))
s.settimeout(3)
time.sleep(1)
data = s.recv(4096)
s.sendall(b'admin\r\n'); time.sleep(0.5)
try: data = s.recv(4096)
except: pass
s.sendall(b'admin\r\n'); time.sleep(1)
try: data = s.recv(4096)
except: pass
out = telnet_cmd(s, 'terminal length 0', 1)
out = telnet_cmd(s, 'show gpon onu detail-info gpon-onu_1/1/1:1', 3)
print('=DETAIL=')
print(out[:3000])
out2 = telnet_cmd(s, 'show running-config interface gpon-onu_1/1/1:1', 3)
print('=RUNCONFIG=')
print(out2[:3000])
out3 = telnet_cmd(s, 'show interface gpon-onu_1/1/1:1', 2)
print('=INTERFACE=')
print(out3[:2000])
out4 = telnet_cmd(s, 'show gpon onu state gpon-olt_1/1/1', 2)
print('=STATE=')
print(out4[:2000])
s.close()
