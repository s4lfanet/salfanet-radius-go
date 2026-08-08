#!/bin/bash
cd /var/www/salfanet-radius

# Get auth token
TOKEN=$(curl -s http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"admin123"}' 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('accessToken','') or d.get('token',''))" 2>&1)
echo "Token: ${TOKEN:0:20}..."

if [ -z "$TOKEN" ]; then
  echo "Trying alternate login..."
  TOKEN=$(curl -s http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' 2>&1)
  echo "Raw response: $TOKEN"
  exit 1
fi

echo "--- Test: ListDevices ---"
RESP=$(curl -s http://127.0.0.1:8080/api/settings/genieacs/devices -H "Authorization: Bearer $TOKEN" 2>&1)
echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print('success:',d.get('success'),'total:',d.get('total')); devs=d.get('devices',[]); print('first 3:'); [print('  ',dev.get('_id'),dev.get('serialNumber'),dev.get('manufacturer'),dev.get('model')) for dev in devs[:3]]" 2>&1

echo "--- Test: DeviceDetail ---"
DEVICE_ID=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('devices',[{}])[0].get('_id',''))" 2>&1)
echo "Device ID: $DEVICE_ID"
if [ -n "$DEVICE_ID" ]; then
  curl -s "http://127.0.0.1:8080/api/settings/genieacs/devices/$DEVICE_ID/detail" -H "Authorization: Bearer $TOKEN" 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); dev=d.get('device',{}); print('serial:',dev.get('serialNumber'),'mfr:',dev.get('manufacturer'),'model:',dev.get('model'),'status:',dev.get('status'),'pppoeUser:',dev.get('pppoeUsername'),'rxPower:',dev.get('rxPower'))" 2>&1
fi

echo "--- Test: Settings page device count ---"
curl -s http://127.0.0.1:8080/api/settings/genieacs -H "Authorization: Bearer $TOKEN" 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); print('settings:',d.get('settings'))" 2>&1

echo "--- All done ---"
