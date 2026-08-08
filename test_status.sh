#!/bin/bash
# Focused test: VPS time vs GenieACS lastInform
echo "=== VPS Time ==="
echo "UTC: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo "Local: $(date '+%Y-%m-%dT%H:%M:%S%z')"
echo ""

# Get first 5 devices with lastInform
echo "=== First 5 devices: lastInform vs now ==="
curl -s -u admin:genieacs 'http://192.168.54.254:7557/devices?projection=_id,_lastInform&_limit=5' | python3 -c "
import sys, json
from datetime import datetime, timezone
now_utc = datetime.now(timezone.utc)
print(f'Python UTC now: {now_utc.isoformat()}')
print()
devices = json.load(sys.stdin)
for d in devices:
    li = d.get('_lastInform', '')
    dev_id = d.get('_id', 'N/A')
    if li:
        try:
            dt = datetime.fromisoformat(li.replace('Z', '+00:00'))
            diff_sec = (now_utc - dt).total_seconds()
            mins = diff_sec / 60
            status = 'Online' if mins <= 15 else 'Offline'
            print(f'{dev_id}:')
            print(f'  lastInform: {li}')
            print(f'  diff: {diff_sec:.0f}s ({mins:.1f} min)')
            print(f'  Our status: {status}')
            print()
        except Exception as e:
            print(f'{dev_id}: lastInform={li} (error: {e})')
    else:
        print(f'{dev_id}: no lastInform')
" 2>&1

echo ""
echo "=== Our API response for same devices ==="
TOKEN=$(python3 -c "
import jwt, time
payload = {'userId': '1', 'email': 'admin@example.com', 'role': 'admin', 'exp': int(time.time()) + 3600, 'iat': int(time.time())}
print(jwt.encode(payload, 'PZenv1O+qemswFw9ouj3pwkYzP1QVGQaQkwxaj2NFSo=', algorithm='HS256'))
")
curl -s -H "Authorization: Bearer $TOKEN" 'http://127.0.0.1:8080/api/settings/genieacs/devices' | python3 -c "
import sys, json
d = json.load(sys.stdin)
devs = d.get('devices', [])[:5]
for dev in devs:
    print(f'{dev.get(\"_id\",\"?\")}: status={dev.get(\"status\",\"?\")} lastInform={dev.get(\"lastInform\",\"?\")}')
" 2>&1
