#!/bin/bash
set -e
echo "Updating OLT Monitoring Optimizations..."

PROJECT_DIR="/var/www/salfanet-radius"
if [ ! -d "$PROJECT_DIR" ]; then
  PROJECT_DIR="/root/salfanet-radius"
fi

if [ ! -d "$PROJECT_DIR" ]; then
  echo "Project directory not found!"
  exit 1
fi

cd "$PROJECT_DIR"
cp -r update-olt-opt/internal/* internal/

echo "Rebuilding Go backend..."
go build -o bin/salfanet-radius cmd/server/main.go

echo "Restarting service..."
systemctl restart salfanet-radius || echo "Please restart the service manually"

echo "Done!"
