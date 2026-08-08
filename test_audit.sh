#!/bin/bash
# Generate JWT and test all GenieACS endpoints
BASE="http://127.0.0.1:8080"
JWT_SECRET="PZenv1O+qemswFw9ouj3pwkYzP1QVGQaQkwxaj2NFSo="

# Generate JWT token
TOKEN=$(python3 -c "
import jwt, time
payload = {
    'userId': '1',
    'email': 'admin@example.com',
    'role': 'admin',
    'exp': int(time.time()) + 3600,
    'iat': int(time.time())
}
token = jwt.encode(payload, '$JWT_SECRET', algorithm='HS256')
print(token)
" 2>&1)

echo "Token: ${TOKEN:0:30}... (len: ${#TOKEN})"
AUTH="Authorization: Bearer $TOKEN"
DEV_ID="00259E-HG8145V5-4857544390D5ADAA"

echo ""
echo "=== Settings GenieACS Endpoints ==="

echo "--- ListDevices ---"
curl -s -H "$AUTH" "$BASE/api/settings/genieacs/devices" > /tmp/r1.json
python3 -c "
import json
d=json.load(open('/tmp/r1.json'))
devs=d.get('devices',[])
print('success:', d.get('success'), 'total:', d.get('total'), 'count:', len(devs))
if devs:
    print('first status:', devs[0].get('status','?'))
    print('first serial:', devs[0].get('serialNumber','?'))
"

echo ""
echo "--- DeviceDetail ---"
curl -s -H "$AUTH" "$BASE/api/settings/genieacs/devices/$DEV_ID/detail" > /tmp/r2.json
python3 -c "
import json
d=json.load(open('/tmp/r2.json'))
print('success:', d.get('success'))
dev=d.get('device',{})
print('serial:', dev.get('serialNumber','?'))
print('status:', dev.get('status','?'))
print('pppoeUser:', dev.get('pppoeUsername','?'))
"

echo ""
echo "=== GenieACS Proxy Endpoints ==="

echo "--- GetDevice ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/devices/$DEV_ID" > /tmp/r3.json
python3 -c "
import json
d=json.load(open('/tmp/r3.json'))
print('has data:', 'data' in d)
data=d.get('data',{})
print('has _id:', '_id' in data)
print('_lastInform:', str(data.get('_lastInform','?'))[:30])
"

echo ""
echo "--- DeviceTasks ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/devices/$DEV_ID/tasks" > /tmp/r4.json
python3 -c "
import json
d=json.load(open('/tmp/r4.json'))
print('has data:', 'data' in d, 'is array:', isinstance(d.get('data'), list))
"

echo ""
echo "--- AllParameters ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/devices/$DEV_ID/all-parameters" > /tmp/r5.json
python3 -c "
import json
d=json.load(open('/tmp/r5.json'))
data=d.get('data',[])
print('has data:', 'data' in d, 'is array:', isinstance(data, list), 'count:', len(data))
if data:
    print('first param:', data[0].get('path','?'), '=', str(data[0].get('value','?'))[:50])
    print('has writable:', 'writable' in data[0])
"

echo ""
echo "--- ListPresets ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/presets" > /tmp/r6.json
python3 -c "
import json
d=json.load(open('/tmp/r6.json'))
print('success:', d.get('success'), 'data count:', len(d.get('data',[])))
"

echo ""
echo "--- ListProvisions ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/provisions" > /tmp/r7.json
python3 -c "
import json
d=json.load(open('/tmp/r7.json'))
print('success:', d.get('success'), 'data count:', len(d.get('data',[])))
"

echo ""
echo "--- ListFaults ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/faults" > /tmp/r8.json
python3 -c "
import json
d=json.load(open('/tmp/r8.json'))
print('success:', d.get('success'), 'data count:', len(d.get('data',[])))
"

echo ""
echo "--- ListFiles ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/files" > /tmp/r9.json
python3 -c "
import json
d=json.load(open('/tmp/r9.json'))
print('success:', d.get('success'), 'data count:', len(d.get('data',[])))
"

echo ""
echo "--- ListConfig ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/config" > /tmp/r10.json
python3 -c "
import json
d=json.load(open('/tmp/r10.json'))
print('success:', d.get('success'), 'data count:', len(d.get('data',[])))
"

echo ""
echo "--- ListVirtualParameters ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/virtual-parameters" > /tmp/r11.json
python3 -c "
import json
d=json.load(open('/tmp/r11.json'))
print('success:', d.get('success'), 'data count:', len(d.get('data',[])))
"

echo ""
echo "--- ListTasks ---"
curl -s -H "$AUTH" "$BASE/api/genieacs/tasks" > /tmp/r12.json
python3 -c "
import json
d=json.load(open('/tmp/r12.json'))
print('success:', d.get('success'), 'task count:', len(d.get('tasks',[])))
"

echo ""
echo "=== All tests done ==="
