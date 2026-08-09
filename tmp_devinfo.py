import json, urllib.request
req = urllib.request.Request("http://192.168.54.254:7557/devices?limit=1")
resp = urllib.request.urlopen(req)
devs = json.loads(resp.read())
dev = devs[0]
igd = dev.get("InternetGatewayDevice", {})

# WiFi
lan = igd.get("LANDevice", {})
print("=== LANDevice keys ===")
print(list(lan.keys()))
lan1 = lan.get("1", {})
wlan = lan1.get("WLANConfiguration", {})
print("\n=== WLANConfiguration keys ===")
print(list(wlan.keys()))
for i in range(1, 5):
    wl = wlan.get(str(i), {})
    if wl:
        ssid = wl.get("SSID", {}).get("_value", "")
        enable = wl.get("Enable", {}).get("_value", "")
        beacon = wl.get("BeaconType", {}).get("_value", "")
        print(f"  WLAN {i}: SSID={ssid}, Enable={enable}, Beacon={beacon}")

# WAN
wan = igd.get("WANDevice", {})
print("\n=== WANDevice keys ===")
print(list(wan.keys()))
wan1 = wan.get("1", {})
wanc = wan1.get("WANConnectionDevice", {})
print("\n=== WANConnectionDevice keys ===")
print(list(wanc.keys()))
wanc1 = wanc.get("1", {})
ppp = wanc1.get("WANPPPConnection", {})
print("\n=== WANPPPConnection keys ===")
print(list(ppp.keys()))
ppp1 = ppp.get("1", {})
if ppp1:
    for k in ["Username", "Password", "ExternalIPAddress", "ConnectionStatus", "Enable", "PPPoECEnable", "MACAddress"]:
        v = ppp1.get(k, {}).get("_value", "")
        print(f"  {k}: {v}")
    # Also check WANIPConnection
    ipconn = wanc1.get("WANIPConnection", {})
    print("\n=== WANIPConnection keys ===")
    print(list(ipconn.keys()))
    ip1 = ipconn.get("1", {})
    if ip1:
        for k in ["ExternalIPAddress", "AddressingType", "ConnectionStatus", "Enable", "MACAddress"]:
            v = ip1.get(k, {}).get("_value", "")
            print(f"  {k}: {v}")

# DeviceInfo
di = igd.get("DeviceInfo", {})
print("\n=== DeviceInfo ===")
for k in ["SerialNumber", "Manufacturer", "ModelName", "SoftwareVersion", "HardwareVersion", "UpTime", "DeviceStatus"]:
    v = di.get(k, {}).get("_value", "")
    if v:
        print(f"  {k}: {v}")

# ManagementServer
ms = igd.get("ManagementServer", {})
print("\n=== ManagementServer ===")
for k in ["ConnectionRequestURL", "ConnectionRequestUsername", "PeriodicInformInterval", "PeriodicInformEnable"]:
    v = ms.get(k, {}).get("_value", "")
    if v:
        print(f"  {k}: {v}")
