#!/bin/bash
# ============================================
# QRIS Dinamis + QRIS Mandiri - Auto Update Script
# Upload ZIP ini ke server Ubuntu, extract, lalu jalankan script ini
# ============================================

set -e

# Warna output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  QRIS Dinamis + Mandiri - Update${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Detect project directory
PROJECT_DIR=""
if [ -d "/var/www/salfanet-radius" ]; then
  PROJECT_DIR="/var/www/salfanet-radius"
elif [ -d "/root/salfanet-radius" ]; then
  PROJECT_DIR="/root/salfanet-radius"
elif [ -d "/home/*/salfanet-radius" ]; then
  PROJECT_DIR=$(ls -d /home/*/salfanet-radius 2>/dev/null | head -1)
fi

if [ -z "$PROJECT_DIR" ]; then
  echo -e "${YELLOW}Project directory not found automatically.${NC}"
  read -p "Masukkan path project salfanet-radius: " PROJECT_DIR
fi

if [ ! -d "$PROJECT_DIR" ]; then
  echo -e "${RED}ERROR: Directory $PROJECT_DIR tidak ditemukan!${NC}"
  exit 1
fi

echo -e "${GREEN}Project directory: ${PROJECT_DIR}${NC}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Step 1: Install qrcode.react dependency
echo -e "${YELLOW}[1/5] Installing qrcode.react dependency...${NC}"
cd "$PROJECT_DIR"
npm install qrcode.react --save
echo -e "${GREEN}✅ qrcode.react installed${NC}"
echo ""

# Step 2: Backup existing files
echo -e "${YELLOW}[2/5] Backing up existing files...${NC}"
BACKUP_DIR="$PROJECT_DIR/backups/qris-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

for f in \
  "src/app/pay/[token]/page.tsx" \
  "src/app/isolated/page.tsx" \
  "src/lib/qris.ts" \
  "src/app/api/payment/create/route.ts" \
  "src/app/api/pppoe/users/check-isolation/route.ts" \
  "src/app/api/invoices/by-token/[token]/route.ts" \
  "src/app/api/company/route.ts" \
  "src/app/admin/settings/company/page.tsx" \
  "src/app/admin/payment-gateway/page.tsx" \
  "prisma/schema.prisma"
do
  if [ -f "$PROJECT_DIR/$f" ]; then
    mkdir -p "$BACKUP_DIR/$(dirname $f)"
    cp "$PROJECT_DIR/$f" "$BACKUP_DIR/$f"
    echo "  Backed up: $f"
  fi
done

echo -e "${GREEN}✅ Backup saved to: ${BACKUP_DIR}${NC}"
echo ""

# Step 3: Copy updated files
echo -e "${YELLOW}[3/5] Copying updated files...${NC}"

# Frontend pages
mkdir -p "$PROJECT_DIR/src/app/pay/[token]"
cp "$SCRIPT_DIR/src/app/pay/[token]/page.tsx" "$PROJECT_DIR/src/app/pay/[token]/page.tsx"
echo "  Updated: src/app/pay/[token]/page.tsx"

mkdir -p "$PROJECT_DIR/src/app/isolated"
cp "$SCRIPT_DIR/src/app/isolated/page.tsx" "$PROJECT_DIR/src/app/isolated/page.tsx"
echo "  Updated: src/app/isolated/page.tsx"

# QRIS Mandiri library
mkdir -p "$PROJECT_DIR/src/lib"
cp "$SCRIPT_DIR/src/lib/qris.ts" "$PROJECT_DIR/src/lib/qris.ts"
echo "  Updated: src/lib/qris.ts"

# API routes
cp "$SCRIPT_DIR/src/app/api/payment/create/route.ts" "$PROJECT_DIR/src/app/api/payment/create/route.ts"
echo "  Updated: src/app/api/payment/create/route.ts"

cp "$SCRIPT_DIR/src/app/api/pppoe/users/check-isolation/route.ts" "$PROJECT_DIR/src/app/api/pppoe/users/check-isolation/route.ts"
echo "  Updated: src/app/api/pppoe/users/check-isolation/route.ts"

mkdir -p "$PROJECT_DIR/src/app/api/invoices/by-token/[token]"
cp "$SCRIPT_DIR/src/app/api/invoices/by-token/[token]/route.ts" "$PROJECT_DIR/src/app/api/invoices/by-token/[token]/route.ts"
echo "  Updated: src/app/api/invoices/by-token/[token]/route.ts"

cp "$SCRIPT_DIR/src/app/api/company/route.ts" "$PROJECT_DIR/src/app/api/company/route.ts"
echo "  Updated: src/app/api/company/route.ts"

# Admin settings
cp "$SCRIPT_DIR/src/app/admin/settings/company/page.tsx" "$PROJECT_DIR/src/app/admin/settings/company/page.tsx"
echo "  Updated: src/app/admin/settings/company/page.tsx"

mkdir -p "$PROJECT_DIR/src/app/admin/payment-gateway"
cp "$SCRIPT_DIR/src/app/admin/payment-gateway/page.tsx" "$PROJECT_DIR/src/app/admin/payment-gateway/page.tsx"
echo "  Updated: src/app/admin/payment-gateway/page.tsx"

# Prisma schema
cp "$SCRIPT_DIR/prisma/schema.prisma" "$PROJECT_DIR/prisma/schema.prisma"
echo "  Updated: prisma/schema.prisma"

echo -e "${GREEN}✅ Files updated${NC}"
echo ""

# Step 4: Apply database migration for new QRIS fields
echo -e "${YELLOW}[4/5] Applying database changes...${NC}"
cd "$PROJECT_DIR"
npx prisma db push --accept-data-loss 2>/dev/null || npx prisma db push
echo -e "${GREEN}✅ Database schema updated${NC}"
echo ""

# Step 5: Build & restart
echo -e "${YELLOW}[5/5] Building and restarting...${NC}"
cd "$PROJECT_DIR"
npm run build
echo ""

# Try PM2 restart
if command -v pm2 &> /dev/null; then
  pm2 restart all 2>/dev/null || true
  echo -e "${GREEN}✅ PM2 restarted${NC}"
else
  echo -e "${YELLOW}PM2 not found. Please restart your app manually.${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  ✅ QRIS Update Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Fitur yang ditambahkan:"
echo -e "  • QR Code inline di halaman pembayaran & isolasi"
echo -e "  • Auto-polling status setiap 5 detik (gateway pihak ke-3)"
echo -e "  • Countdown timer 24 menit"
echo -e "  • Animasi sukses pembayaran"
echo -e ""
echo -e "Gateway QRIS Dinamis (pihak ke-3):"
echo -e "  • Tripay (method: QRIS)"
echo -e "  • Duitku (method: SP)"
echo -e ""
echo -e "${GREEN}🆕 QRIS Mandiri (TANPA biaya admin):${NC}"
echo -e "  • Gunakan QRIS dari rekening bank Anda sendiri"
echo -e "  • Aktifkan di: Admin → Settings → Company → QRIS Mandiri"
echo -e "  • Paste kode QRIS statis dari aplikasi bank"
echo -e "  • Sistem konversi otomatis ke QRIS dinamis + nominal"
echo -e "  • Konfirmasi pembayaran manual via admin"
echo ""
