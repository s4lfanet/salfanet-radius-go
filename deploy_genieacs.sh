#!/bin/bash
cd /var/www/salfanet-radius
echo "--- Git pull ---"
git pull origin master 2>&1 | tail -5
export PATH=$PATH:/usr/local/go/bin
echo "--- Building Go binary ---"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server 2>&1 | tail -5
echo "Go build exit: $?"
echo "--- Restarting API ---"
systemctl restart salfanet-api
sleep 2
systemctl status salfanet-api --no-pager 2>&1 | head -5
echo "--- Health check ---"
curl -s http://127.0.0.1:8080/api/health
echo ""
echo "--- Test: ListDevices endpoint ---"
curl -s http://127.0.0.1:8080/api/settings/genieacs/devices 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); print('success:',d.get('success'),'total:',d.get('total'),'first device:',d.get('devices',[{}])[0].get('_id','none') if d.get('devices') else 'empty')" 2>&1
echo "--- Test: DeviceDetail endpoint ---"
DEVICE_ID=$(curl -s http://127.0.0.1:8080/api/settings/genieacs/devices 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('devices',[{}])[0].get('_id',''))" 2>&1)
echo "Device ID: $DEVICE_ID"
if [ -n "$DEVICE_ID" ]; then
  curl -s "http://127.0.0.1:8080/api/settings/genieacs/devices/$DEVICE_ID/detail" 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin); dev=d.get('device',{}); print('serial:',dev.get('serialNumber'),'manufacturer:',dev.get('manufacturer'),'model:',dev.get('model'),'status:',dev.get('status'))" 2>&1
fi
echo "--- All done ---"
