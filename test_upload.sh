#!/bin/bash
echo "=== Login response ==="
curl -s -X POST http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d '{"username":"superadmin","password":"admin123"}'
echo
echo "=== Login with email ==="
curl -s -X POST http://127.0.0.1:8080/api/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"admin123"}'
echo
