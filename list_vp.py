import sys, json
data = json.load(sys.stdin)
for d in data:
    print(d["_id"])
