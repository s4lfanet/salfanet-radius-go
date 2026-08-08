#!/bin/bash
# Get a NextAuth session token from the database
SESSION_TOKEN=$(mysql -u salfanet_user -phBH0ReZP9WzmG9iE81Cg salfanet_radius -e "SELECT sessionToken FROM sessions LIMIT 1" -s -N 2>/dev/null)
echo "Session token: ${SESSION_TOKEN:0:20}..."

if [ -z "$SESSION_TOKEN" ]; then
  echo "No sessions found in DB. Checking if sessions table exists..."
  mysql -u salfanet_user -phBH0ReZP9WzmG9iE81Cg salfanet_radius -e "SHOW TABLES LIKE '%session%'" 2>/dev/null
  exit 1
fi

echo "--- Test: ListDevices with NextAuth cookie ---"
RESP=$(curl -s http://127.0.0.1:8080/api/settings/genieacs/devices -H "Cookie: next-auth.session-token=$SESSION_TOKEN" 2>&1)
echo "$RESP" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print('success:', d.get('success'))
print('total:', d.get('total'))
devs = d.get('devices', [])
print('device count:', len(devs))
if devs:
    dev = devs[0]
    print('first device:', dev.get('_id'), dev.get('serialNumber'), dev.get('manufacturer'), dev.get('model'), dev.get('status'))
" 2>&1

echo "--- Test: DeviceDetail ---"
DEVICE_ID=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('devices',[{}])[0].get('_id',''))" 2>&1)
echo "Device ID: $DEVICE_ID"
if [ -n "$DEVICE_ID" ]; then
  curl -s "http://127.0.0.1:8080/api/settings/genieacs/devices/$DEVICE_ID/detail" -H "Cookie: next-auth.session-token=$SESSION_TOKEN" 2>&1 | python3 -c "
import sys, json
d = json.load(sys.stdin)
dev = d.get('device', {})
print('success:', d.get('success'))
print('serial:', dev.get('serialNumber'))
print('mfr:', dev.get('manufacturer'))
print('model:', dev.get('model'))
print('status:', dev.get('status'))
print('pppoeUser:', dev.get('pppoeUsername'))
print('rxPower:', dev.get('rxPower'))
print('ponMode:', dev.get('ponMode'))
" 2>&1
fi

echo "--- Test: DeviceParameters ---"
if [ -n "$DEVICE_ID" ]; then
  curl -s "http://127.0.0.1:8080/api/settings/genieacs/devices/$DEVICE_ID/parameters" -H "Cookie: next-auth.session-token=$SESSION_TOKEN" 2>&1 | python3 -c "
import sys, json
d = json.load(sys.stdin)
params = d.get('parameters', [])
print('success:', d.get('success'))
print('param count:', len(params))
if params:
    p = params[0]
    print('first param:', p.get('path'), '=', p.get('value'))
" 2>&1
fi

echo "--- All done ---"
