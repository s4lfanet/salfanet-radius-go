#!/bin/bash

# Auto Installer for OLT Provisioning Laravel App on Ubuntu 20.04/22.04+
# Harap jalankan script ini sebagai root (Gunakan: sudo ./install.sh)

if [ "$EUID" -ne 0 ]; then
  echo "Error: Silakan jalankan script ini sebagai root (sudo ./install.sh)"
  exit 1
fi

echo "=========================================================="
echo "  Auto Installer: OLT ZTE C320 Provisioning (Laravel) "
echo "=========================================================="
echo ""

# 1. Setup Variabel dari Input User
read -p "Masukkan Domain atau IP VPS untuk aplikasi ini (misal: 192.168.1.50): " APP_DOMAIN
read -p "Masukkan Password Baru untuk Database 'olt_user': " DB_PASS

APP_DIR="/var/www/olt-provisioning"
DB_NAME="olt_db"
DB_USER="olt_user"

echo ""
echo "Memulai instalasi... Proses ini akan memakan waktu beberapa menit."
echo "=========================================================="

# 2. Update Sistem & Install Paket yang Dibutuhkan
echo "[1/6] Mengupdate sistem dan menginstal dependensi dasar..."
apt update && apt upgrade -y
apt install -y software-properties-common curl unzip git mariadb-server nginx

# Menginstal PHP dan ekstensinya dari repositori bawaan Ubuntu
# Tanpa metapackage 'php' agar tidak otomatis menginstal Apache2 (karena kita menggunakan Nginx)
apt install -y php-cli php-fpm php-mysql php-xml php-mbstring php-curl php-zip php-bcmath

# Pastikan Apache2 mati jika terlanjur terinstal agar port 80 tidak bentrok dengan Nginx
systemctl stop apache2 2>/dev/null
systemctl disable apache2 2>/dev/null

# Install Composer
echo "[2/6] Menginstal Composer & Node.js..."
curl -sS https://getcomposer.org/installer | php
mv composer.phar /usr/local/bin/composer

# Install Node.js 20.x
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs

# 3. Setup Database & Import
echo "[3/6] Mengonfigurasi Database MariaDB/MySQL..."
mysql -u root <<EOF
CREATE DATABASE IF NOT EXISTS ${DB_NAME};
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';
FLUSH PRIVILEGES;
EOF

# 4. Copy File Aplikasi
echo "[4/6] Menyalin file aplikasi ke ${APP_DIR}..."
mkdir -p $APP_DIR

# Mengkopi semua file (termasuk file hidden seperti .env.example) dari direktori saat ini ke /var/www
cp -r ./* $APP_DIR/
cp -r ./.[!.]* $APP_DIR/ 2>/dev/null

cd $APP_DIR

if [ -f "olt.sql" ]; then
    echo "--> Mengimport file database (olt.sql)..."
    mysql -u root ${DB_NAME} < olt.sql
else
    echo "--> Peringatan: File olt.sql tidak ditemukan, abaikan import database."
fi

# 5. Konfigurasi Aplikasi (Laravel & Node)
echo "[5/6] Mengonfigurasi Laravel dan Menginstal Dependensi (Composer & NPM)..."
export COMPOSER_ALLOW_SUPERUSER=1
composer install --optimize-autoloader --no-dev

# Setup .env
if [ -f ".env.example" ]; then
    cp .env.example .env
fi

# Replace konfigurasi DB di .env
sed -i "s/^DB_CONNECTION=.*/DB_CONNECTION=mysql/" .env
sed -i "s/^DB_HOST=.*/DB_HOST=127.0.0.1/" .env
sed -i "s/^DB_PORT=.*/DB_PORT=3306/" .env
sed -i "s/^DB_DATABASE=.*/DB_DATABASE=${DB_NAME}/" .env
sed -i "s/^DB_USERNAME=.*/DB_USERNAME=${DB_USER}/" .env
sed -i "s/^DB_PASSWORD=.*/DB_PASSWORD=${DB_PASS}/" .env

php artisan key:generate

# Buat folder cache/session/views yang dibutuhkan Laravel
mkdir -p $APP_DIR/storage/framework/{cache/data,sessions,views}
mkdir -p $APP_DIR/storage/logs
mkdir -p $APP_DIR/bootstrap/cache

# Hak akses folder
chown -R www-data:www-data $APP_DIR
chmod -R 775 $APP_DIR/storage
chmod -R 775 $APP_DIR/bootstrap/cache

# Build asset (Vite/Tailwind)
npm install
npm run build

# 6. Konfigurasi Nginx
echo "[6/6] Mengonfigurasi Web Server (Nginx)..."

# Mencari socket PHP-FPM yang aktif secara dinamis
PHP_SOCK=$(find /var/run/php -name "php*-fpm.sock" | head -n 1)
if [ -z "$PHP_SOCK" ]; then
    PHP_SOCK="/var/run/php/php-fpm.sock" # Fallback
fi

cat > /etc/nginx/sites-available/olt-provisioning <<EOF
server {
    listen 80;
    server_name ${APP_DOMAIN};
    root ${APP_DIR}/public;

    add_header X-Frame-Options "SAMEORIGIN";
    add_header X-Content-Type-Options "nosniff";

    index index.php;

    charset utf-8;

    location / {
        try_files \$uri \$uri/ /index.php?\$query_string;
    }

    location = /favicon.ico { access_log off; log_not_found off; }
    location = /robots.txt  { access_log off; log_not_found off; }

    error_page 404 /index.php;

    location ~ \.php$ {
        fastcgi_pass unix:${PHP_SOCK};
        fastcgi_param SCRIPT_FILENAME \$realpath_root\$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.(?!well-known).* {
        deny all;
    }
}
EOF

# Aktifkan konfigurasi
ln -s /etc/nginx/sites-available/olt-provisioning /etc/nginx/sites-enabled/ 2>/dev/null
rm -f /etc/nginx/sites-enabled/default

# Pastikan Apache2 mati agar tidak bentrok port 80
systemctl stop apache2 2>/dev/null
systemctl disable apache2 2>/dev/null

# Restart PHP-FPM & Nginx
PHP_FPM_SERVICE=$(systemctl list-units --type=service --all | grep -o 'php[0-9\.]*-fpm.service' | head -n 1)
if [ -n "$PHP_FPM_SERVICE" ]; then
    systemctl restart $PHP_FPM_SERVICE
fi
nginx -t
systemctl restart nginx

echo ""
echo "=========================================================="
echo " Instalasi Selesai! "
echo " Aplikasi dapat diakses melalui: http://${APP_DOMAIN}"
echo " Database Name: ${DB_NAME}"
echo " Database User: ${DB_USER}"
echo " Database Pass: ${DB_PASS}"
echo " Direktori App: ${APP_DIR}"
echo "=========================================================="
echo ""
