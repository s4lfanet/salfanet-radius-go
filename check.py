import sys, json
d = json.load(sys.stdin)[0]
igd = d.get("InternetGatewayDevice", {})
mgmt = igd.get("ManagementServer", {})
for key in ["ConnectionRequestURL", "ConnectionRequestUsername", "ConnectionRequestPassword", "Username", "Password"]:
    val = mgmt.get(key, {})
    if isinstance(val, dict):
        print(f"{key}: {val.get('_value', 'N/A')}")
    else:
        print(f"{key}: {val}")
print("---")
print("lastInform:", d.get("_lastInform", "?"))
