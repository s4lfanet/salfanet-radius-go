#!/bin/bash
echo "=== VPS System Time ==="
echo "UTC: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo "Local: $(date '+%Y-%m-%dT%H:%M:%S%z')"
timedatectl 2>/dev/null | head -5
echo ""

echo "=== GenieACS Server Time (from headers) ==="
curl -s -u admin:genieacs -I 'http://192.168.54.254:7557/devices?_limit=1' 2>&1 | grep -i date
echo ""

echo "=== GenieACS config: threshold/timeout/ping settings ==="
curl -s -u admin:genieacs 'http://192.168.54.254:7557/config' | python3 -c "
import sys, json
config = json.load(sys.stdin)
for item in config:
    key = item.get('key', '')
    val = item.get('value', '')
    kl = key.lower()
    if any(x in kl for x in ['inform', 'timeout', 'ping', 'online', 'offline', 'threshold', 'stale', 'expire', 'session']):
        print(f'{key} = {val}')
" 2>&1
echo ""

echo "=== Device with most recent lastInform ==="
curl -s -u admin:genieacs 'http://192.168.54.254:7557/devices?projection=_id,_lastInform&_limit=5&_sort=_lastInform:desc' | python3 -c "
import sys, json
from datetime import datetime, timezone
now_utc = datetime.now(timezone.utc)
print(f'Python UTC: {now_utc.isoformat()}')
print()
devices = json.load(sys.stdin)
for d in devices:
    li = d.get('_lastInform', '')
    dev_id = d.get('_id', 'N/A')
    if li:
        dt = datetime.fromisoformat(li.replace('Z', '+00:00'))
        diff_sec = (now_utc - dt).total_seconds()
        print(f'{dev_id}: lastInform={li} | {diff_sec:.0f}s ({diff_sec/60:.1f} min) ago')
" 2>&1
