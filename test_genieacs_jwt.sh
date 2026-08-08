#!/bin/bash
JWT_SECRET="PZenv1O+qemswFw9ouj3pwkYzP1QVGQaQkwxaj2NFSo="
TOKEN=$(python3 << 'PYEOF'
import jwt, time
secret = "PZenv1O+qemswFw9ouj3pwkYzP1QVGQaQkwxaj2NFSo="
payload = {
    "userID": "test-admin-id",
    "email": "admin@example.com",
    "role": "SUPER_ADMIN",
    "exp": int(time.time()) + 3600,
    "iss": "salfanet-radius",
}
print(jwt.encode(payload, secret, algorithm="HS256"))
PYEOF
)
echo "Token: ${TOKEN:0:30}..."

echo "--- Test: ListDevices ---"
curl -s http://127.0.0.1:8080/api/settings/genieacs/devices -H "Authorization: Bearer $TOKEN" -o /tmp/test_devices.json 2>&1
echo "Response size: $(wc -c < /tmp/test_devices.json) bytes"
python3 << 'PYEOF'
import json
with open("/tmp/test_devices.json") as f:
    d = json.load(f)
print('success:', d.get('success'))
print('total:', d.get('total'))
devs = d.get('devices', [])
print('device count:', len(devs))
if devs:
    dev = devs[0]
    print('first device:', dev.get('_id'), dev.get('serialNumber'), dev.get('manufacturer'), dev.get('model'), dev.get('status'))
PYEOF

echo "--- Test: DeviceDetail ---"
DEVICE_ID=$(python3 -c "import json; d=json.load(open('/tmp/test_devices.json')); print(d.get('devices',[{}])[0].get('_id',''))" 2>&1)
echo "Device ID: $DEVICE_ID"
if [ -n "$DEVICE_ID" ]; then
  curl -s "http://127.0.0.1:8080/api/settings/genieacs/devices/$DEVICE_ID/detail" -H "Authorization: Bearer $TOKEN" -o /tmp/test_detail.json 2>&1
  echo "Response size: $(wc -c < /tmp/test_detail.json) bytes"
  python3 << 'PYEOF'
import json
with open("/tmp/test_detail.json") as f:
    d = json.load(f)
dev = d.get('device', {})
print('success:', d.get('success'))
print('serial:', dev.get('serialNumber'))
print('mfr:', dev.get('manufacturer'))
print('model:', dev.get('model'))
print('status:', dev.get('status'))
print('pppoeUser:', dev.get('pppoeUsername'))
print('rxPower:', dev.get('rxPower'))
print('ponMode:', dev.get('ponMode'))
PYEOF
fi

echo "--- Test: DeviceParameters ---"
if [ -n "$DEVICE_ID" ]; then
  curl -s "http://127.0.0.1:8080/api/settings/genieacs/devices/$DEVICE_ID/parameters" -H "Authorization: Bearer $TOKEN" -o /tmp/test_params.json 2>&1
  echo "Response size: $(wc -c < /tmp/test_params.json) bytes"
  python3 << 'PYEOF'
import json
with open("/tmp/test_params.json") as f:
    d = json.load(f)
params = d.get('parameters', [])
print('success:', d.get('success'))
print('param count:', len(params))
if params:
    p = params[0]
    print('first param:', p.get('path'), '=', p.get('value'))
PYEOF
fi

echo "--- Test: ListTasks ---"
curl -s "http://127.0.0.1:8080/api/settings/genieacs/tasks" -H "Authorization: Bearer $TOKEN" -o /tmp/test_tasks.json 2>&1
echo "Response size: $(wc -c < /tmp/test_tasks.json) bytes"
python3 << 'PYEOF'
import json
with open("/tmp/test_tasks.json") as f:
    d = json.load(f)
tasks = d.get('tasks', [])
print('success:', d.get('success'))
print('task count:', len(tasks))
PYEOF

echo "--- All done ---"
