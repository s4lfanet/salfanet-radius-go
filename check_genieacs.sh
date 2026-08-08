#!/bin/bash
curl -s -u admin:genieacs 'http://192.168.54.254:7557/devices' -o /tmp/genieacs_devices.json
python3 << 'PYEOF'
import json
with open("/tmp/genieacs_devices.json") as f:
    d = json.load(f)
dev = d[0]
vp = dev.get("VirtualParameters", {})
print("=== VirtualParameters ===")
for k in sorted(vp.keys()):
    v = vp[k]
    if isinstance(v, dict):
        print(f"  {k}: {v.get('_value', '')}")
    else:
        print(f"  {k}: {v}")
print("\n=== Top-level keys ===")
for k in sorted(dev.keys()):
    if k not in ("VirtualParameters", "InternetGatewayDevice", "Device"):
        val = dev[k]
        if isinstance(val, dict):
            print(f"  {k}: {json.dumps(val)[:200]}")
        else:
            print(f"  {k}: {val}")
print(f"\nTotal devices: {len(d)}")
PYEOF
