import json, urllib.request
script = 'return {writable: false, value: ["test", "xsd:string"]};'
data = json.dumps({"script": script}).encode()
req = urllib.request.Request("http://192.168.54.254:7557/virtual_parameters/testVP3", data=data, headers={"Content-Type": "application/json"}, method="PUT")
try:
    r = urllib.request.urlopen(req)
    print("Status:", r.status)
    print(r.read().decode())
except Exception as e:
    print("Error:", e)
    if hasattr(e, "read"):
        print(e.read().decode())
