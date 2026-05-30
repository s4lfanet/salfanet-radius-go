#!/bin/bash
set -e

APP_DIR="/var/www/salfanet-radius"
DB="salfanet_radius"

echo "=== Deploy v2.52.84 ==="
cd "$APP_DIR"

echo "[1/5] Git pull..."
git pull origin master

echo "[2/5] DB migration: add odpId column..."
MYSQL="mysql -u salfanet_user -psalfanetradius123 $DB"
COL_EXISTS=$($MYSQL -sN -e "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='$DB' AND TABLE_NAME='olt_onu_status' AND COLUMN_NAME='odpId';")
if [ "$COL_EXISTS" = "0" ]; then
  $MYSQL -e "ALTER TABLE olt_onu_status ADD COLUMN odpId VARCHAR(191) NULL AFTER customerId;" 2>&1
  echo "Column odpId added"
else
  echo "Column odpId already exists, skip"
fi
IDX_EXISTS=$($MYSQL -sN -e "SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='$DB' AND TABLE_NAME='olt_onu_status' AND INDEX_NAME='idx_onu_odpId';")
if [ "$IDX_EXISTS" = "0" ]; then
  $MYSQL -e "ALTER TABLE olt_onu_status ADD INDEX idx_onu_odpId (odpId);" 2>&1
  echo "Index idx_onu_odpId added"
else
  echo "Index idx_onu_odpId already exists, skip"
fi
echo "DB OK"

echo "[3/5] Build Go binary..."
go build -o bin/server ./cmd/server/
echo "Go build OK"

echo "[4/5] Restart Go service..."
systemctl restart salfanet-api
sleep 2
systemctl is-active salfanet-api && echo "salfanet-api: running" || echo "salfanet-api: FAILED"

echo "[5/5] Build frontend & restart PM2..."
npm run build
pm2 restart salfanet-radius
sleep 2
pm2 status salfanet-radius --no-color 2>/dev/null | grep salfanet || true

echo "=== Deploy v2.52.84 DONE ==="
