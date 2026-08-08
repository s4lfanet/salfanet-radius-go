#!/bin/bash
echo "=== VPS Time ==="
date -u
echo ""

echo "=== GenieACS HTTP Date Header ==="
curl -s -u admin:genieacs -D - 'http://192.168.54.254:7557/devices?_limit=1' -o /dev/null 2>&1 | grep -i "^date:"
echo ""

echo "=== GenieACS Config (filtered) ==="
curl -s -u admin:genieacs 'http://192.168.54.254:7557/config' | python3 -c "
import sys, json
config = json.load(sys.stdin)
for item in config:
    key = item.get('key', '')
    val = item.get('value', '')
    kl = key.lower()
    if any(x in kl for x in ['inform', 'timeout', 'ping', 'online', 'offline', 'threshold', 'stale', 'expire', 'session', 'cwmp']):
        print(f'{key} = {val}')
" 2>&1
echo ""

echo "=== Most recent 3 devices by lastInform ==="
curl -s -u admin:genieacs 'http://192.168.54.254:7557/devices?projection=_id,_lastInform&_limit=3' | python3 -c "
import sys, json
from datetime import datetime, timezone
now_utc = datetime.now(timezone.utc)
print(f'Python UTC: {now_utc.isoformat()}')
devices = json.load(sys.stdin)
for d in devices:
    li = d.get('_lastInform', '')
    dev_id = d.get('_id', 'N/A')
    if li:
        dt = datetime.fromisoformat(li.replace('Z', '+00:00'))
        diff_sec = (now_utc - dt).total_seconds()
        print(f'{dev_id}: lastInform={li} | {diff_sec:.0f}s ({diff_sec/60:.1f} min) ago')
" 2>&1
