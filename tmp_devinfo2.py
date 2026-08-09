import json, urllib.request
req = urllib.request.Request("http://192.168.54.254:7557/devices?limit=1")
resp = urllib.request.urlopen(req)
devs = json.loads(resp.read())
dev = devs[0]
igd = dev.get("InternetGatewayDevice", {})
lan = igd.get("LANDevice", {})
wlan = lan.get("1", {}).get("WLANConfiguration", {})
for i in ["1", "2", "5"]:
    wl = wlan.get(i, {})
    if wl:
        print(f"\n=== WLAN {i} all keys ===")
        for k in sorted(wl.keys()):
            v = wl[k]
            if isinstance(v, dict) and "_value" in v:
                print(f"  {k}: {v['_value']} (writable={v.get('_writable', '')})")
            elif isinstance(v, dict) and v.get("_object"):
                print(f"  {k}: [object] keys={list(v.keys())[:5]}")

# Also check WANConnectionDevice 2
wan = igd.get("WANDevice", {})
wanc = wan.get("1", {}).get("WANConnectionDevice", {})
for i in ["1", "2"]:
    w = wanc.get(i, {})
    if w:
        print(f"\n=== WANConnectionDevice.{i} keys ===")
        print(list(w.keys()))
        ppp = w.get("WANPPPConnection", {})
        ppp1 = ppp.get("1", {})
        if ppp1:
            for k in sorted(ppp1.keys()):
                v = ppp1[k]
                if isinstance(v, dict) and "_value" in v:
                    print(f"  PPP.{k}: {v['_value']}")
        ip = w.get("WANIPConnection", {})
        ip1 = ip.get("1", {})
        if ip1:
            for k in sorted(ip1.keys()):
                v = ip1[k]
                if isinstance(v, dict) and "_value" in v:
                    print(f"  IP.{k}: {v['_value']}")
