# Panduan Instalasi di VPS (Ubuntu Server)

Panduan ini akan menjelaskan langkah-langkah untuk menginstal aplikasi Provisioning OLT ZTE C320 berbasis Laravel ini pada VPS (terutama dengan OS Ubuntu Server).

## 1. Persiapan Server VPS
Pastikan VPS Anda sudah terinstal web server, PHP, Database, Composer, dan Node.js. Jika menggunakan Ubuntu, jalankan perintah berikut:

```bash
# Update sistem
sudo apt update && sudo apt upgrade -y

# Install Nginx
sudo apt install nginx -y

# Install PHP (contoh menggunakan PHP 8.1/8.2 beserta ekstensi yang dibutuhkan)
sudo apt install php php-cli php-fpm php-mysql php-xml php-mbstring php-curl php-zip php-bcmath -y

# Install MariaDB/MySQL
sudo apt install mariadb-server -y

# Install Composer
curl -sS https://getcomposer.org/installer | php
sudo mv composer.phar /usr/local/bin/composer

# Install Node.js dan NPM (versi 20.x)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
```

## 2. Setup Database
Buat database baru untuk aplikasi dan siapkan user database.

1. Masuk ke console MySQL:
   ```bash
   sudo mysql -u root -p
   ```
2. Buat database dan user (Ganti `password_rahasia` dengan password yang aman):
   ```sql
   CREATE DATABASE olt_db;
   CREATE USER 'olt_user'@'localhost' IDENTIFIED BY 'password_rahasia';
   GRANT ALL PRIVILEGES ON olt_db.* TO 'olt_user'@'localhost';
   FLUSH PRIVILEGES;
   EXIT;
   ```

## 3. Upload Source Code & Import Database
Pindahkan seluruh source code project ini ke VPS Anda, misalnya diletakkan di `/var/www/olt-provisioning`. 

Setelah source code berada di VPS, lakukan import database dari file `olt.sql` yang ada di dalam root folder project:

```bash
# Sesuaikan path direktori jika berbeda
mysql -u olt_user -p olt_db < /var/www/olt-provisioning/olt.sql
```

## 4. Konfigurasi Aplikasi (Laravel)
Masuk ke direktori project dan setup environment.

```bash
cd /var/www/olt-provisioning

# Install dependensi PHP (abaikan require-dev)
composer install --optimize-autoloader --no-dev

# Salin file .env (jika belum ada)
cp .env.example .env
```

Edit file `.env` (`nano .env`) dan pastikan konfigurasi koneksi database Anda benar:
```env
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=olt_db
DB_USERNAME=olt_user
DB_PASSWORD=password_rahasia
```

Kemudian jalankan perintah konfigurasi dasar Laravel:
```bash
# Generate APP_KEY
php artisan key:generate

# Mengatur hak akses folder (penting agar tidak error permission)
sudo chown -R www-data:www-data /var/www/olt-provisioning
sudo chmod -R 775 /var/www/olt-provisioning/storage
sudo chmod -R 775 /var/www/olt-provisioning/bootstrap/cache

# Install dependensi Node.js & build aset frontend
npm install
npm run build
```

## 5. Konfigurasi Nginx (Web Server)
Buat file virtual host (server block) baru di Nginx:

```bash
sudo nano /etc/nginx/sites-available/olt-provisioning
```

Masukkan konfigurasi berikut (jangan lupa ganti `domain_atau_ip_vps_anda` dengan IP atau Domain milik Anda, dan sesuaikan path `php8.1-fpm.sock` jika Anda menggunakan versi PHP lain):

```nginx
server {
    listen 80;
    server_name domain_atau_ip_vps_anda;
    root /var/www/olt-provisioning/public;

    add_header X-Frame-Options "SAMEORIGIN";
    add_header X-Content-Type-Options "nosniff";

    index index.php;

    charset utf-8;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location = /favicon.ico { access_log off; log_not_found off; }
    location = /robots.txt  { access_log off; log_not_found off; }

    error_page 404 /index.php;

    location ~ \.php$ {
        # PERHATIAN: Pastikan versi sock sesuai dengan PHP yang Anda install
        fastcgi_pass unix:/var/run/php/php8.1-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.(?!well-known).* {
        deny all;
    }
}
```

Simpan file tersebut (CTRL+X, lalu Y). Aktifkan konfigurasi dan restart Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/olt-provisioning /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

Aplikasi sekarang sudah terinstal dan bisa diakses melalui web browser!
