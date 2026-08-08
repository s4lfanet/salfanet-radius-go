#!/bin/bash
# Test different query field names for tasks
echo "--- Direct GenieACS: all tasks ---"
curl -s -u admin:genieacs 'http://192.168.54.254:7557/tasks' -w '\nHTTP:%{http_code}\n' 2>&1 | tail -3

echo ""
echo "--- Direct GenieACS: query with device ---"
curl -s -u admin:genieacs 'http://192.168.54.254:7557/tasks?query=%5B%5B%22device%22%2C%22%3D%22%2C%2200259E-HG8145V5-4857544390D5ADAA%22%5D%5D' -w '\nHTTP:%{http_code}\n' 2>&1 | tail -3

echo ""
echo "--- Direct GenieACS: query with deviceId ---"
curl -s -u admin:genieacs 'http://192.168.54.254:7557/tasks?query=%5B%5B%22deviceId%22%2C%22%3D%22%2C%2200259E-HG8145V5-4857544390D5ADAA%22%5D%5D' -w '\nHTTP:%{http_code}\n' 2>&1 | tail -3
