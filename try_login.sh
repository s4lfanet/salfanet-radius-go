#!/bin/bash
# Try login with various passwords
for pw in "admin123" "salfanet" "salfanet123" "password" "admin" "Admin123!" "s4lfanet"; do
  RESP=$(curl -s http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d "{\"email\":\"admin@example.com\",\"password\":\"$pw\"}" 2>&1)
  TOKEN=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('accessToken',''))" 2>&1)
  if [ -n "$TOKEN" ]; then
    echo "SUCCESS with password: $pw"
    echo "TOKEN=$TOKEN"
    break
  else
    echo "FAIL with password: $pw"
  fi
done
