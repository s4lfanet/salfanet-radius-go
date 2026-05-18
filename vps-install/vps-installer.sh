#!/bin/bash
# ============================================================================
# SALFANET RADIUS - Main Installer
# ============================================================================
# Satu installer untuk semua environment:
#   - Public VPS (DigitalOcean, Vultr, Hetzner, Linode, dll)
#   - Proxmox LXC Container (local / intranet)
#   - Proxmox VM / KVM Local
#   - Bare Metal / Server Fisik Lokal
#
# Usage:
#   bash vps-installer.sh              # Interactive (recommended)
#   bash vps-installer.sh --env lxc    # Force LXC mode
#   bash vps-installer.sh --env vm     # Force VM mode
#   bash vps-installer.sh --env vps    # Force VPS mode
#   bash vps-installer.sh --env bare   # Force bare metal mode
# ============================================================================

set -e
set -o pipefail

# Set TERM default for non-interactive environments (prevents `clear` from failing)
export TERM="${TERM:-xterm}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Parse CLI args
export UNATTENDED="${UNATTENDED:-false}"
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --env)         export DEPLOY_ENV="$2";        shift ;;
        --ip)          export FORCE_IP="$2";          shift ;;
        --domain)      export VPS_DOMAIN="$2";        shift ;;
        --db-pass)     export DB_PASSWORD="$2";       shift ;;
        --unattended)  export UNATTENDED="true" ;;
        --vpn)         export INSTALL_VPN_SERVERS="true" ;;
        --no-vpn)      export INSTALL_VPN_SERVERS="false" ;;
        --help|-h)
            echo "Usage: bash vps-installer.sh [--env vps|lxc|vm|bare] [--ip IP] [--domain DOMAIN] [--db-pass PASS] [--unattended] [--vpn] [--no-vpn]"
            echo ""
            echo "  --env vps|lxc|vm|bare  Force environment type (default: auto-detect)"
            echo "  --ip ADDRESS           Force IP address"
            echo "  --domain DOMAIN        Set domain for SSL (skips prompt)"
            echo "  --db-pass PASS         Set DB password (default: from common.sh)"
            echo "  --unattended           Non-interactive mode, use all defaults"
            echo "  --vpn                  Install WireGuard + L2TP VPN servers (VPS only, unattended)"
            echo "  --no-vpn               Skip VPN server prompt (unattended)"
            exit 0
            ;;
    esac
    shift
done

source "$SCRIPT_DIR/common.sh"

# ============================================================================
# STEP 0: PILIH ENVIRONMENT
# ============================================================================

select_environment() {
    print_step "Step 0: Pilih Environment"

    # Deteksi otomatis dulu
    detect_environment

    echo -e "${CYAN}Environment yang terdeteksi: ${WHITE}${DEPLOY_ENV_LABEL}${NC}"
    echo ""
    echo "Pilih environment yang sesuai:"
    echo ""
    echo -e "  ${WHITE}1)${NC} Public VPS / Cloud Server"
    echo -e "     ${YELLOW}DigitalOcean, Vultr, Hetzner, Google Cloud, AWS, dll${NC}"
    echo -e "     IP Publik, UFW aktif, akses dari internet"
    echo ""
    echo -e "  ${WHITE}2)${NC} Proxmox LXC Container"
    echo -e "     ${YELLOW}Container lokal di Proxmox, IP private / VLAN${NC}"
    echo -e "     UFW dilewati, akses dari jaringan lokal"
    echo ""
    echo -e "  ${WHITE}3)${NC} Proxmox VM / KVM / VirtualBox"
    echo -e "     ${YELLOW}Virtual machine lokal, IP private${NC}"
    echo -e "     UFW aktif, akses dari jaringan lokal"
    echo ""
    echo -e "  ${WHITE}4)${NC} Bare Metal / Server Fisik Lokal"
    echo -e "     ${YELLOW}Server fisik di kantor/rumah, IP private${NC}"
    echo -e "     UFW aktif, akses dari jaringan lokal"
    echo ""

    # Tentukan default berdasarkan deteksi
    local DEFAULT_CHOICE="1"
    case "$DEPLOY_ENV" in
        lxc)  DEFAULT_CHOICE="2" ;;
        vm)   DEFAULT_CHOICE="3" ;;
        bare) DEFAULT_CHOICE="4" ;;
        vps)  DEFAULT_CHOICE="1" ;;
    esac

    if [ "${UNATTENDED}" = "true" ]; then
        ENV_CHOICE="$DEFAULT_CHOICE"
        echo "[unattended] Menggunakan environment: ${DEPLOY_ENV_LABEL}"
    else
        read -t 20 -p "Pilih [1/2/3/4] (default: ${DEFAULT_CHOICE} - ${DEPLOY_ENV_LABEL}): " ENV_CHOICE </dev/tty \
            || ENV_CHOICE="$DEFAULT_CHOICE"
    fi
    echo ""

    case "${ENV_CHOICE:-$DEFAULT_CHOICE}" in
        1) export DEPLOY_ENV="vps"  ;;
        2) export DEPLOY_ENV="lxc"  ;;
        3) export DEPLOY_ENV="vm"   ;;
        4) export DEPLOY_ENV="bare" ;;
        *) export DEPLOY_ENV="$DEPLOY_ENV" ;;
    esac

    apply_environment_settings
    print_success "Environment: ${DEPLOY_ENV_LABEL}"

    if [ "$IS_CONTAINER" = "true" ]; then
        echo ""
        print_info "LXC Container terdeteksi:"
        print_info "  - UFW firewall: DILEWATI (konfigurasi di Proxmox host)"
        print_info "  - Firewall Proxmox: tambahkan rules di datacenter firewall"
        print_info "  - Port yang perlu dibuka: 80, 443, 1812/udp, 1813/udp, 3799/udp"
    fi
    echo ""
}

# ============================================================================
# INITIALIZATION
# ============================================================================

initialize_installer() {
    print_banner
    initialize_user_selection

    # Deteksi IP
    local DETECTED_IP=""
    local IP_TYPE=""

    # Gunakan FORCE_IP jika disediakan via --ip
    if [ -n "${FORCE_IP:-}" ]; then
        DETECTED_IP="$FORCE_IP"
        IP_TYPE="Manual/Forced"
    else
        DETECTED_IP=$(detect_ip_address)
        # Tentukan tipe IP
        if [[ "$DETECTED_IP" =~ ^(10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.|127\.) ]]; then
            case "$DEPLOY_ENV" in
                lxc)  IP_TYPE="Private - Proxmox LXC" ;;
                vm)   IP_TYPE="Private - Local VM" ;;
                bare) IP_TYPE="Private - Local Server" ;;
                *)    IP_TYPE="Private/Local" ;;
            esac
        else
            IP_TYPE="Public"
        fi
    fi

    show_installation_info "$DETECTED_IP" "$IP_TYPE"

    # Konfirmasi / override IP
    echo -e "IP yang akan digunakan: ${CYAN}${DETECTED_IP}${NC} (${IP_TYPE})"
    echo ""
    if [ "${UNATTENDED}" = "true" ]; then
        IP_CONFIRM="y"
        echo "[unattended] Menggunakan IP: ${DETECTED_IP}"
    else
        read -t 15 -p "Gunakan IP ini? [Y/n/custom]: " IP_CONFIRM </dev/tty || IP_CONFIRM="y"
    fi
    echo ""

    if [[ "$IP_CONFIRM" =~ ^[Nn]$ ]]; then
        read -p "Masukkan IP address: " MANUAL_IP </dev/tty
        if [[ $MANUAL_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            export VPS_IP="$MANUAL_IP"
            print_success "Menggunakan IP manual: $VPS_IP"
        else
            print_error "Format IP tidak valid, pakai IP terdeteksi: $DETECTED_IP"
            export VPS_IP="$DETECTED_IP"
        fi
    elif [[ "$IP_CONFIRM" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        export VPS_IP="$IP_CONFIRM"
        print_success "Menggunakan IP custom: $VPS_IP"
    else
        export VPS_IP="$DETECTED_IP"
        print_success "Menggunakan IP: $VPS_IP"
    fi

    # Untuk LXC / VM: info tambahan akses
    if [[ "$DEPLOY_ENV" =~ ^(lxc|vm|bare)$ ]]; then
        echo ""
        print_info "Aplikasi akan diakses dari jaringan lokal:"
        print_info "  URL: http://${VPS_IP}"
        print_info "  Pastikan perangkat di jaringan yang sama"
        if [ "$DEPLOY_ENV" = "lxc" ]; then
            print_info "  Atau setup port-forwarding di Proxmox host"
        fi
    fi

    echo ""

    # ---- Domain & SSL (hanya untuk VPS publik) ----
    # Preserve VPS_DOMAIN if already set via --domain CLI flag
    export VPS_DOMAIN="${VPS_DOMAIN:-}"
    export VPS_USE_SSL="false"
    if [ "${DEPLOY_ENV}" = "vps" ]; then
        echo -e "${CYAN}[Domain & SSL]${NC}"
        echo -e "  Jika kamu punya domain (misal: radius.hotspotapp.net), masukkan di sini."
        echo -e "  Installer akan otomatis setup SSL via Let's Encrypt (Certbot)."
        echo -e "  ${YELLOW}Pastikan DNS domain sudah pointing ke IP ${VPS_IP} sebelum lanjut.${NC}"
        echo -e "  Kosongkan jika ingin akses pakai IP saja."
        echo ""
        if [ "${UNATTENDED}" = "true" ]; then
            DOMAIN_INPUT="${VPS_DOMAIN:-}"
            [ -n "$DOMAIN_INPUT" ] && echo "[unattended] Domain: ${DOMAIN_INPUT}" || echo "[unattended] Domain dilewati (akses via IP)"
        else
            read -t 30 -p "Masukkan domain (kosong = skip): " DOMAIN_INPUT </dev/tty || DOMAIN_INPUT=""
        fi
        echo ""
        if [ -n "$DOMAIN_INPUT" ]; then
            # Validasi format domain sederhana
            if [[ "$DOMAIN_INPUT" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z]{2,})+$ ]]; then
                export VPS_DOMAIN="$DOMAIN_INPUT"
                echo ""
                if [ "${UNATTENDED}" = "true" ]; then
                    SSL_EMAIL="admin@${VPS_DOMAIN}"
                else
                    read -t 15 -p "Email untuk SSL certificate (Let's Encrypt): " SSL_EMAIL </dev/tty || SSL_EMAIL=""
                fi
                export VPS_SSL_EMAIL="${SSL_EMAIL:-admin@${VPS_DOMAIN}}"
                export VPS_USE_SSL="true"
                print_success "Domain: ${VPS_DOMAIN}"
                print_info "SSL email: ${VPS_SSL_EMAIL}"
            else
                print_warning "Format domain tidak valid, dilewati."
            fi
        else
            print_info "Domain dilewati, akses pakai IP: ${VPS_IP}"
        fi
    fi

    export INSTALL_INFO_FILE="${APP_DIR}/INSTALLATION_INFO.txt"
    export NEXTAUTH_SECRET=$(generate_secret)
    print_info "Initialization selesai"
}

# ============================================================================
# KONFIRMASI SEBELUM MULAI
# ============================================================================

confirm_installation() {
    echo ""
    echo -e "${WHITE}=====================================================${NC}"
    echo -e "${WHITE}  Ringkasan Instalasi${NC}"
    echo -e "${WHITE}=====================================================${NC}"
    echo -e "  Environment : ${GREEN}${DEPLOY_ENV_LABEL}${NC}"
    echo -e "  IP Address  : ${GREEN}${VPS_IP}${NC}"
    if [ -n "${VPS_DOMAIN:-}" ]; then
    echo -e "  Domain      : ${GREEN}${VPS_DOMAIN}${NC} (SSL via Let's Encrypt)"
    fi
    echo -e "  App Dir     : ${GREEN}${APP_DIR}${NC}"
    echo -e "  App User    : ${GREEN}${APP_USER}${NC}"
    echo -e "  Database    : ${GREEN}${DB_NAME}${NC}"
    echo -e "  UFW         : ${GREEN}$([ "${SKIP_UFW}" = "true" ] && echo "DILEWATI (LXC)" || echo "Aktif")${NC}"
    if [ "${DEPLOY_ENV}" = "vps" ] && [ "${INSTALL_VPN_SERVERS:-}" != "false" ]; then
    echo -e "  VPN Servers : ${GREEN}WireGuard + L2TP (akan ditanya setelah install utama)${NC}"
    fi
    echo -e "${WHITE}=====================================================${NC}"
    echo ""
    if [ "${UNATTENDED}" = "true" ]; then
        echo "[unattended] Melanjutkan instalasi otomatis..."
    else
        read -t 30 -p "Mulai instalasi? [Y/n]: " CONFIRM_START </dev/tty || CONFIRM_START="y"
        echo ""
        if [[ "$CONFIRM_START" =~ ^[Nn]$ ]]; then
            print_info "Instalasi dibatalkan."
            exit 0
        fi
    fi
}

# ============================================================================
# VPN SERVER INSTALLATION (Public VPS only)
# ============================================================================

install_vpn_servers() {
    print_step "Step 10: VPN Server Setup (WireGuard + L2TP)"

    export VPN_WG_INSTALLED="false"
    export VPN_L2TP_INSTALLED="false"

    print_info "VPS publik terdeteksi. Setup VPN server memungkinkan NAS/router"
    print_info "(MikroTik, dll) terhubung langsung ke VPS tanpa CHR forwarder."
    echo ""
    echo -e "  ${WHITE}WireGuard${NC} — RouterOS 7.1+, UDP 51820, overhead minimal"
    echo -e "  ${WHITE}L2TP/IPsec${NC} — RouterOS 6.x+, kompatibel luas"
    echo ""

    # Tentukan default berdasarkan flag CLI
    local DEFAULT_WG="y"
    local DEFAULT_L2TP="y"

    if [ "${INSTALL_VPN_SERVERS:-}" = "false" ]; then
        # --no-vpn flag: skip semua
        print_info "[--no-vpn] Melewati setup VPN server."
        return 0
    fi

    # Tanya WireGuard
    local INSTALL_WG=""
    if [ "${UNATTENDED}" = "true" ]; then
        INSTALL_WG="${INSTALL_VPN_SERVERS:-n}"
        [ "${INSTALL_WG}" = "true" ] && INSTALL_WG="y"
        echo "[unattended] WireGuard install: ${INSTALL_WG}"
    else
        read -t 30 -p "Install WireGuard VPN Server? [Y/n]: " INSTALL_WG </dev/tty || INSTALL_WG="y"
    fi
    INSTALL_WG="${INSTALL_WG:-y}"
    echo ""

    if [[ "${INSTALL_WG}" =~ ^[Yy]$ ]]; then
        print_info "Menjalankan install-wg-server.sh..."
        if bash "$SCRIPT_DIR/install-wg-server.sh"; then
            export VPN_WG_INSTALLED="true"
            print_success "WireGuard VPN Server terinstall (wg0, port 51820)"
        else
            print_warning "WireGuard install gagal — bisa diulang manual: bash vps-install/install-wg-server.sh"
        fi
    else
        print_info "WireGuard dilewati."
    fi
    echo ""

    # Tanya L2TP
    local INSTALL_L2TP=""
    if [ "${UNATTENDED}" = "true" ]; then
        INSTALL_L2TP="${INSTALL_VPN_SERVERS:-n}"
        [ "${INSTALL_L2TP}" = "true" ] && INSTALL_L2TP="y"
        echo "[unattended] L2TP install: ${INSTALL_L2TP}"
    else
        read -t 30 -p "Install L2TP/IPsec VPN Server (untuk RouterOS 6.x)? [Y/n]: " INSTALL_L2TP </dev/tty || INSTALL_L2TP="y"
    fi
    INSTALL_L2TP="${INSTALL_L2TP:-y}"
    echo ""

    if [[ "${INSTALL_L2TP}" =~ ^[Yy]$ ]]; then
        print_info "Menjalankan install-l2tp-server.sh..."
        if bash "$SCRIPT_DIR/install-l2tp-server.sh"; then
            export VPN_L2TP_INSTALLED="true"
            print_success "L2TP/IPsec VPN Server terinstall"
        else
            print_warning "L2TP install gagal — bisa diulang manual: bash vps-install/install-l2tp-server.sh"
        fi
    else
        print_info "L2TP dilewati."
    fi
    echo ""
}

# ============================================================================
# MAIN INSTALLATION PROCESS
# ============================================================================

run_installation() {
    print_step "Starting SALFANET RADIUS Installation"
    
    # Step 1: System Setup
    print_info "Running Step 1: System Setup..."
    source "$SCRIPT_DIR/install-system.sh"
    install_system || {
        print_error "System setup failed!"
        exit 1
    }
    
    # Step 2: Node.js Installation
    print_info "Running Step 2: Node.js Installation..."
    source "$SCRIPT_DIR/install-nodejs.sh"
    install_nodejs || {
        print_error "Node.js installation failed!"
        exit 1
    }
    
    # Step 3: MySQL Installation
    print_info "Running Step 3: MySQL Installation..."
    source "$SCRIPT_DIR/install-mysql.sh"
    install_mysql || {
        print_error "MySQL installation failed!"
        exit 1
    }
    
    # Step 4: Application Setup
    print_info "Running Step 4: Application Setup..."
    source "$SCRIPT_DIR/install-app.sh"
    install_app || {
        print_error "Application setup failed!"
        exit 1
    }

    # Step 4.5: Go Backend Build & Systemd Service
    print_info "Running Step 4.5: Go Backend (build + systemd service)..."
    source "$SCRIPT_DIR/install-go.sh"
    install_go_backend || {
        print_error "Go backend installation failed!"
        exit 1
    }

    # Step 5: FreeRADIUS Installation
    print_info "Running Step 5: FreeRADIUS Installation..."
    source "$SCRIPT_DIR/install-freeradius.sh"
    install_freeradius || {
        print_error "FreeRADIUS installation failed!"
        exit 1
    }
    
    # Step 6: Nginx Configuration
    print_info "Running Step 6: Nginx Configuration..."
    source "$SCRIPT_DIR/install-nginx.sh"
    install_nginx || {
        print_error "Nginx configuration failed!"
        exit 1
    }
    
    # Step 7: PM2 & Build
    print_info "Running Step 7: PM2 Installation & Application Build..."
    source "$SCRIPT_DIR/install-pm2.sh"
    install_pm2_and_build || {
        print_error "PM2 installation or build failed!"
        exit 1
    }

    # Step 8: Security (fail2ban + UFW + cleanup cron)
    print_info "Running Step 8: Security Setup (fail2ban, UFW, cleanup cron)..."
    source "$SCRIPT_DIR/install-security.sh"
    install_security || {
        print_warning "Security setup partial — lanjutkan manual: bash vps-install/install-security.sh"
    }

    # Step 9: Auth Self-Heal (legacy-safe; ensures active SUPER_ADMIN exists)
    print_info "Running Step 9: Auth Self-Heal..."
    if [ -f "$SCRIPT_DIR/fix-auth-after-update.sh" ]; then
        APP_DIR="$APP_DIR" bash "$SCRIPT_DIR/fix-auth-after-update.sh" || {
            print_warning "Auth self-heal warning (continuing...)"
        }
    fi

    export VPN_CLIENT_INSTALLED="false"

    # Step 10: VPN Servers (hanya untuk public VPS)
    if [ "${DEPLOY_ENV}" = "vps" ]; then
        install_vpn_servers
    fi

    print_success "All installation steps completed successfully!"
}

# ============================================================================
# FINALIZATION
# ============================================================================

create_installation_info() {
    print_step "Creating Installation Information"
    
    cat > "${INSTALL_INFO_FILE}" <<EOF
============================================
SALFANET RADIUS - Installation Information
============================================

Installation Date : $(date)
Environment       : ${DEPLOY_ENV_LABEL}
DEPLOY_ENV        : ${DEPLOY_ENV}
VPS/Server IP     : ${VPS_IP}
Domain            : ${VPS_DOMAIN:-"(tidak diset - akses via IP)"}
SSL               : $([ "${VPS_USE_SSL:-false}" = "true" ] && echo "Aktif - /etc/letsencrypt/live/${VPS_DOMAIN}/" || echo "Tidak aktif")
Timezone          : $(timedatectl show --property=Timezone --value 2>/dev/null || echo 'Asia/Jakarta')
Current Time      : $(date '+%Y-%m-%d %H:%M:%S %Z')

>> SYSTEM INFORMATION
--------------------
Operating System: $(lsb_release -d 2>/dev/null | cut -f2 || cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)
Node.js Version : $(node --version 2>/dev/null || echo 'N/A')
npm Version     : $(npm --version 2>/dev/null || echo 'N/A')
MySQL Version   : $(mysql --version 2>/dev/null | awk '{print $3}' || echo 'N/A')
FreeRADIUS      : $(freeradius -v 2>&1 | head -n1 || echo 'N/A')
PM2 Version     : $(pm2 --version 2>/dev/null || echo 'N/A')
Go Version      : $(go version 2>/dev/null | awk '{print $3}' || echo 'N/A')
Go API Service  : $(systemctl is-active salfanet-api 2>/dev/null || echo 'N/A')

[>] DATABASE CREDENTIALS
-----------------------
Database Name  : ${DB_NAME}
Database User  : ${DB_USER}
Database Pass  : ${DB_PASSWORD}
MySQL Root Pass: ${DB_ROOT_PASSWORD}

[>] APPLICATION CONFIGURATION
----------------------------
App Directory   : ${APP_DIR}
Environment File: ${APP_DIR}/.env
PM2 Config      : ${APP_DIR}/ecosystem.config.js
Deploy Script   : ${APP_DIR}/deploy.sh

[>] ACCESS INFORMATION
---------------------
Application URL : $([ -n "${VPS_DOMAIN:-}" ] && echo "https://${VPS_DOMAIN}" || echo "http://${VPS_IP}")
Admin Panel     : $([ -n "${VPS_DOMAIN:-}" ] && echo "https://${VPS_DOMAIN}/admin" || echo "http://${VPS_IP}/admin")
Default Login   : Lihat database seeders
Firewall (UFW)  : $([ "${SKIP_UFW}" = "true" ] && echo "DILEWATI - konfigurasi di Proxmox host" || echo "$(ufw status 2>/dev/null | head -1 | sed 's/^Status: //')")

[>] SSTP VPN CLIENT (koneksi ke MikroTik CHR)
--------------------------------------------
VPN Client Tools: $([ "${VPN_CLIENT_INSTALLED:-false}" = "true" ] && echo "Terinstall (/usr/local/bin/vpn-connect)" || echo "Tidak diinstall (opsional)")
Install VPN     : bash ${APP_DIR}/vps-install/install-vpn-client.sh
Konfigurasi VPN : Edit /etc/vpn/vpn.conf (VPN_SERVER, VPN_USER, VPN_PASS, VPN_SUBNET)
Koneksi VPN     : vpn-connect start
Status VPN      : vpn-connect status

[>] WIREGUARD VPN SERVER (VPS sebagai server, NAS connect langsung)
-------------------------------------------------------------------
Protokol        : WireGuard (UDP 51820) — RouterOS 7.1+
Subnet          : 10.200.0.0/24
Install WG      : bash ${APP_DIR}/vps-install/install-wg-server.sh
Install L2TP    : bash ${APP_DIR}/vps-install/install-l2tp-server.sh  (untuk RouterOS 6)
Status WG       : wg show wg0
Manage Peers    : Admin Panel → Network → VPN Server → klik tombol WireGuard
Info File       : /etc/wireguard/wg-server-info.json

[>] SERVICE STATUS
-----------------
MySQL     : $(systemctl is-active mysql 2>/dev/null || echo 'N/A')
FreeRADIUS: $(systemctl is-active freeradius 2>/dev/null || echo 'N/A')
Nginx     : $(systemctl is-active nginx 2>/dev/null || echo 'N/A')
PM2 App   : $(pm2 list 2>/dev/null | grep -q "salfanet-radius.*online" && echo "online" || echo "offline")

[>] NETWORK / FIREWALL
----------------------
Deploy Env: ${DEPLOY_ENV_LABEL}
$([ "${SKIP_UFW}" = "true" ] && echo "UFW      : DILEWATI - konfigurasi firewall di Proxmox host/datacenter" || \
    echo "UFW      : $(ufw status 2>/dev/null | head -1 | sed 's/^Status: //')")
Port 80   : HTTP
Port 443  : HTTPS
Port 1812 : RADIUS Authentication (UDP)
Port 1813 : RADIUS Accounting (UDP)
Port 3799 : RADIUS CoA (UDP)
$([ "${DEPLOY_ENV}" = "lxc" ] && echo "
NOTE Proxmox LXC:
  Tambahkan port forwarding di Proxmox host jika perlu akses dari luar LAN:
  iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to ${VPS_IP}:80
  Atau gunakan Proxmox built-in firewall rules." || true)

[>] USEFUL COMMANDS
------------------
# Application Management
pm2 status                          # Cek status app
pm2 logs salfanet-radius            # Lihat logs
pm2 restart salfanet-radius         # Restart app
${APP_DIR}/deploy.sh                # Deploy update

# Database
mysql -u ${DB_USER} -p${DB_PASSWORD} ${DB_NAME}          # Connect DB
mysqldump -u root -p${DB_ROOT_PASSWORD} ${DB_NAME} > backup.sql  # Backup DB

# FreeRADIUS
freeradius -X                       # Debug mode
systemctl restart freeradius        # Restart
tail -f /var/log/freeradius/radius.log  # Logs

[>] NEXT STEPS
-------------
1. Akses aplikasi : $([ -n "${VPS_DOMAIN:-}" ] && echo "https://${VPS_DOMAIN}" || echo "http://${VPS_IP}")
2. Login admin, ganti password
3. Tambahkan NAS/router di pengaturan RADIUS
$([ "${DEPLOY_ENV}" = "vps" ] && [ -z "${VPS_DOMAIN:-}" ] && echo "6. Setup SSL: certbot --nginx -d yourdomain.com" || \
  echo "6. [Opsional] Setup Cloudflare Tunnel untuk akses dari internet")

============================================
Installasi selesai! Environment: ${DEPLOY_ENV_LABEL}
============================================
EOF

    print_success "Info instalasi disimpan: ${INSTALL_INFO_FILE}"
}

show_final_summary() {
    echo ""
    echo -e "${GREEN}=====================================================${NC}"
    echo -e "${GREEN}  INSTALASI SELESAI!${NC}"
    echo -e "${GREEN}=====================================================${NC}"
    echo ""
    echo -e "  Environment : ${CYAN}${DEPLOY_ENV_LABEL}${NC}"
    echo -e "  IP Address  : ${WHITE}${VPS_IP}${NC}"
    if [ -n "${VPS_DOMAIN:-}" ]; then
    echo -e "  Domain      : ${WHITE}${VPS_DOMAIN}${NC}"
    fi
    echo ""
    echo -e "${CYAN}Akses Aplikasi:${NC}"
    if [ -n "${VPS_DOMAIN:-}" ]; then
    echo -e "  Web App : ${WHITE}https://${VPS_DOMAIN}${NC}"
    echo -e "  Admin   : ${WHITE}https://${VPS_DOMAIN}/admin${NC}"
    else
    echo -e "  Web App : ${WHITE}http://${VPS_IP}${NC}"
    echo -e "  Admin   : ${WHITE}http://${VPS_IP}/admin${NC}"
    fi


    if [ "${DEPLOY_ENV}" = "lxc" ]; then
        echo -e "${YELLOW}Info Proxmox LXC:${NC}"
        echo "  - Akses dari host/LAN: http://${VPS_IP}"
        echo "  - Port yang perlu dibuka di Proxmox firewall: 80, 443, 1812, 1813, 3799"
        echo "  - UFW tidak diaktifkan (gunakan Proxmox Datacenter Firewall)"
        echo ""
    elif [ "${DEPLOY_ENV}" = "vm" ] || [ "${DEPLOY_ENV}" = "bare" ]; then
        echo -e "${YELLOW}Info Local Server:${NC}"
        echo "  - Akses dari LAN: http://${VPS_IP}"
        echo "  - Untuk akses dari internet, setup port-forward atau Cloudflare Tunnel"
        echo ""
    fi
    echo -e "${CYAN}PM2 Status:${NC}"
    sudo su - ${APP_USER} -c 'pm2 list' 2>/dev/null || pm2 list 2>/dev/null || true
    echo ""
    echo -e "${CYAN}Kredensial Database:${NC}"
    echo "  DB Name : ${DB_NAME}"
    echo "  DB User : ${DB_USER}"
    echo "  DB Pass : ${DB_PASSWORD}"
    echo "  Root Pass: ${DB_ROOT_PASSWORD}"
    echo ""
    echo -e "${CYAN}File Info Instalasi:${NC}"
    echo "  ${INSTALL_INFO_FILE}"
    echo ""
    echo -e "${YELLOW}[!] KEAMANAN:${NC}"
    echo "  - Ganti password admin default setelah login pertama"
    if [ "${DEPLOY_ENV}" = "vps" ] && [ -z "${VPS_DOMAIN:-}" ]; then
        echo "  - Setup SSL: certbot --nginx -d yourdomain.com"
    fi
    echo ""
    if [ "${DEPLOY_ENV}" = "vps" ]; then
        echo -e "${CYAN}VPN Server:${NC}"
        if [ "${VPN_WG_INSTALLED:-false}" = "true" ]; then
        echo -e "  WireGuard   : ${GREEN}Terinstall${NC} (wg0, UDP 51820, subnet 10.200.0.0/24)"
        echo "  Status      : wg show wg0"
        echo "  Info        : cat /etc/wireguard/wg-server-info.json"
        else
        echo -e "  WireGuard   : ${YELLOW}Tidak diinstall${NC} — jalankan manual: bash vps-install/install-wg-server.sh"
        fi
        if [ "${VPN_L2TP_INSTALLED:-false}" = "true" ]; then
        echo -e "  L2TP/IPsec  : ${GREEN}Terinstall${NC} (xl2tpd, UDP 1701, subnet 10.201.0.0/24)"
        echo "  Info        : cat /etc/salfanet/l2tp/l2tp-server-info.json"
        else
        echo -e "  L2TP/IPsec  : ${YELLOW}Tidak diinstall${NC} — jalankan manual: bash vps-install/install-l2tp-server.sh"
        fi
        echo ""
    fi
    echo -e "${CYAN}Perintah Berguna:${NC}"
    echo "  Lihat logs : sudo su - ${APP_USER} -c 'pm2 logs salfanet-radius'"
    echo "  Restart app: sudo su - ${APP_USER} -c 'pm2 restart all'"
    echo "  PM2 status : sudo su - ${APP_USER} -c 'pm2 list'"
    echo "  Deploy update: ${APP_DIR}/deploy.sh"
    echo ""
    echo -e "${GREEN}=====================================================${NC}"
    echo ""
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================

main() {
    # Pre-checks
    check_root
    check_directory
    detect_os

    # Step 0: Pilih environment (auto-detect + konfirmasi user)
    select_environment

    # Initialize (user, IP)
    initialize_installer

    # Konfirmasi sebelum mulai
    confirm_installation

    # Run installation
    run_installation

    # Ensure firewall is active (except environments where UFW is skipped)
    if ! ensure_ufw_enabled; then
        print_warning "UFW tidak dapat diaktifkan otomatis. Cek manual: ufw status verbose"
    fi
    
    # Finalize
    create_installation_info
    show_final_summary

    # Log completion
    print_info "Installation log: $INSTALL_LOG"
}

# Run main function
main "$@"
