#!/bin/bash
curl -s -u admin:genieacs 'http://192.168.54.254:7557/config' | python3 << 'PYEOF'
import sys, json
config = json.load(sys.stdin)
for item in config:
    key = item.get('key', '')
    val = item.get('value', '')
    kl = key.lower()
    if any(x in kl for x in ['inform', 'timeout', 'ping', 'online', 'offline', 'threshold', 'stale', 'expire', 'session', 'cwmp', 'connection']):
        print(f'{key} = {val}')
PYEOF
