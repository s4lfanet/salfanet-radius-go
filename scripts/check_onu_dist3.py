import telnetlib, time
tn = telnetlib.Telnet('136.1.1.100', 23, timeout=15)
tn.read_until(b'Username:', timeout=5)
tn.write(b'salfanet\r\n')
tn.read_until(b'Password:', timeout=5)
tn.write(b'seven789\r\n')
tn.read_until(b'ZXAN#', timeout=5)
tn.write(b'show gpon onu detail-info gpon-onu_0/1/1:28\r\n')
out28 = tn.read_until(b'ZXAN#', timeout=8).decode(errors='ignore')
tn.write(b'show gpon onu detail-info gpon-onu_0/1/1:1\r\n')
out1 = tn.read_until(b'ZXAN#', timeout=8).decode(errors='ignore')
tn.write(b'exit\r\n')
tn.close()
print('=== ONU 28 ===')
for line in out28.split('\n'):
    if 'Distance' in line or 'distance' in line:
        print(line.strip())
print('=== ONU 1 ===')
for line in out1.split('\n'):
    if 'Distance' in line or 'distance' in line:
        print(line.strip())
