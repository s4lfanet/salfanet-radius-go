#!/usr/bin/env bash
# ============================================================================
# Salfanet RADIUS — Go Backend Installer Module
# ============================================================================
# When sourced by vps-installer.sh: provides install_go_backend() function
# When run standalone: full Go + systemd setup
# Usage:
#   sudo bash install-go.sh               # standalone full install
#   source install-go.sh; install_go_backend  # modular call from installer
# ============================================================================

# ─── Config (defaults — overridden by common.sh when sourced) ────────────────
GO_VERSION="${GO_VERSION:-1.23.5}"
GO_ARCH="linux-amd64"
GO_SERVICE_NAME="salfanet-api"
APP_PORT="${APP_PORT:-8080}"
WA_PORT="${WA_PORT:-3001}"

# Try to source common.sh for shared print/config functions (when sourced from installer)
_INSTALL_GO_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$_INSTALL_GO_SCRIPT_DIR/common.sh" ] && [ -z "${_COMMON_SH_LOADED:-}" ]; then
    source "$_INSTALL_GO_SCRIPT_DIR/common.sh"
fi

# Fallback print functions when run standalone (not via installer)
if ! declare -f print_step >/dev/null 2>&1; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
    print_step()    { echo -e "\n${YELLOW}▶ $1${NC}"; }
    print_success() { echo -e "${GREEN}[OK] $1${NC}"; }
    print_info()    { echo -e "${YELLOW}[INFO] $1${NC}"; }
    print_error()   { echo -e "${RED}[ERROR] $1${NC}" >&2; }
    print_warning() { echo -e "${YELLOW}[WARN] $1${NC}"; }
fi

# ─── Root check ──────────────────────────────────────────────────────────────
_check_go_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "Run as root: sudo bash $0"
        exit 1
    fi
}


# ============================================================================
# FUNCTIONS
# ============================================================================

# Install Go runtime (idempotent)
install_go_runtime() {
    print_step "Installing Go ${GO_VERSION} runtime"
    export PATH="/usr/local/go/bin:$PATH"

    local INSTALLED_VER
    INSTALLED_VER="$(go version 2>/dev/null | awk '{print $3}')" || true

    if [ "$INSTALLED_VER" = "go${GO_VERSION}" ]; then
        print_success "Go ${GO_VERSION} already installed, skipping download"
        return 0
    fi

    print_info "Downloading Go ${GO_VERSION}..."
    local GO_TAR="go${GO_VERSION}.${GO_ARCH}.tar.gz"
    wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}" || {
        print_error "Failed to download Go tarball"
        return 1
    }
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm -f "/tmp/${GO_TAR}"
    ln -sf /usr/local/go/bin/go   /usr/local/bin/go
    ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

    print_success "Go installed: $(go version)"
}

# Build the Go binary (requires app source in APP_DIR)
build_go_binary() {
    print_step "Building Go backend binary"
    local _APP_DIR="${APP_DIR:-/var/www/salfanet-radius}"
    export PATH="/usr/local/go/bin:$PATH"

    cd "$_APP_DIR" || {
        print_error "App directory not found: $_APP_DIR"
        return 1
    }

    mkdir -p "$_APP_DIR/bin" "$_APP_DIR/logs" "$_APP_DIR/uploads/logos" "$_APP_DIR/uploads/payment-proofs" "$_APP_DIR/uploads/customer-photos"

    print_info "Running go mod download..."
    go mod download 2>&1 | tail -5 || true

    print_info "Compiling server binary..."
    if go build -o bin/server ./cmd/server/ 2>/tmp/go-build.log; then
        chmod +x bin/server
        print_success "Go binary built: bin/server ($(du -sh "$_APP_DIR/bin/server" | cut -f1))"
    else
        print_error "Go build failed!"
        cat /tmp/go-build.log
        return 1
    fi
}

# Create and enable systemd service for the Go API binary
setup_go_systemd_service() {
    print_step "Setting up systemd service: ${GO_SERVICE_NAME}"
    local _APP_DIR="${APP_DIR:-/var/www/salfanet-radius}"
    local _APP_USER="${APP_USER:-salfanet}"

    cat > "/etc/systemd/system/${GO_SERVICE_NAME}.service" <<SYSTEMD
[Unit]
Description=Salfanet RADIUS Go API Server
After=network.target mysql.service
Wants=mysql.service

[Service]
Type=simple
User=${_APP_USER}
WorkingDirectory=${_APP_DIR}
EnvironmentFile=${_APP_DIR}/.env
ExecStart=${_APP_DIR}/bin/server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${GO_SERVICE_NAME}

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=${_APP_DIR}/logs ${_APP_DIR}/uploads ${_APP_DIR}/backups /etc/ppp /etc/salfanet /etc/wireguard
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
SYSTEMD

    # Create directories referenced in ReadWritePaths to prevent systemd NAMESPACE error
    mkdir -p /etc/salfanet /etc/wireguard ${_APP_DIR}/backups

    systemctl daemon-reload
    systemctl enable "${GO_SERVICE_NAME}"
    print_success "Systemd service created: ${GO_SERVICE_NAME}.service"
}

# Start or restart the Go API service
start_go_service() {
    local _APP_DIR="${APP_DIR:-/var/www/salfanet-radius}"

    if systemctl is-active --quiet "${GO_SERVICE_NAME}" 2>/dev/null; then
        systemctl restart "${GO_SERVICE_NAME}"
        print_success "Go API service restarted (${GO_SERVICE_NAME})"
    elif [ -f "/etc/systemd/system/${GO_SERVICE_NAME}.service" ]; then
        systemctl start "${GO_SERVICE_NAME}"
        print_success "Go API service started (${GO_SERVICE_NAME})"
    else
        # Fallback: run with nohup if no systemd unit exists yet
        fuser -k "${APP_PORT}/tcp" 2>/dev/null || true
        sleep 1
        mkdir -p "$_APP_DIR/logs"
        nohup "$_APP_DIR/bin/server" > "$_APP_DIR/logs/server.log" 2>&1 &
        sleep 2
        if curl -sf "http://localhost:${APP_PORT}/api/health" >/dev/null 2>&1; then
            print_success "Go API server running on port ${APP_PORT} (nohup)"
        else
            print_warning "Go server started but health check failed — check logs/server.log"
        fi
    fi
}

# ── Modular entry point (used by vps-installer.sh) ──────────────────────────
# Installs Go runtime, builds binary, sets up systemd service, starts it.
# Assumes APP_DIR already has the source code (run after copy_application_files).
install_go_backend() {
    install_go_runtime || return 1
    build_go_binary    || return 1
    setup_go_systemd_service
    start_go_service
    print_success "Go backend installed and running on port ${APP_PORT}"
}

# ============================================================================
# STANDALONE MODE — full install (run directly: sudo bash install-go.sh)
# ============================================================================

_standalone_install() {
    _check_go_root
    set -euo pipefail

    local _REPO_URL="${REPO_URL:-https://github.com/s4lfanet/salfanet-radius-go.git}"
    local _APP_DIR="${APP_DIR:-/var/www/salfanet-radius}"
    local _APP_USER="${APP_USER:-salfanet}"
    local _NODE_VERSION="${NODE_VERSION:-20}"

    print_step "=== Salfanet RADIUS Go — Standalone Install ==="
    print_info "App dir: $_APP_DIR | API port: ${APP_PORT} | WA port: ${WA_PORT}"

    # 1. System packages
    print_info "Updating system packages..."
    apt-get update -qq
    apt-get install -y -qq curl wget git unzip build-essential \
        nginx ufw mysql-client \
        freeradius freeradius-mysql \
        ca-certificates gnupg

    # 2. Go runtime
    install_go_runtime

    # 3. Node.js (for wa-service sidecar)
    if ! command -v node &>/dev/null; then
        print_info "Installing Node.js ${_NODE_VERSION}..."
        curl -fsSL "https://deb.nodesource.com/setup_${_NODE_VERSION}.x" | bash -
        apt-get install -y nodejs
    else
        print_success "Node.js $(node --version) already installed"
    fi

    # 4. PM2
    command -v pm2 &>/dev/null || npm install -g pm2 --quiet

    # 5. App user
    id "$_APP_USER" &>/dev/null || \
        useradd --system --no-create-home --shell /bin/false "$_APP_USER"

    # 6. App directory
    mkdir -p "$_APP_DIR/bin" "$_APP_DIR/logs" "$_APP_DIR/public" "$_APP_DIR/uploads/logos" "$_APP_DIR/uploads/payment-proofs" "$_APP_DIR/uploads/customer-photos"

    # 7. Clone / update repo
    if [ -d "$_APP_DIR/.git" ]; then
        print_info "Updating repo..."
        git -C "$_APP_DIR" pull --ff-only
    else
        print_info "Cloning from $_REPO_URL..."
        git clone "$_REPO_URL" "$_APP_DIR"
    fi

    # 8. Build Go binary
    export APP_DIR="$_APP_DIR"
    build_go_binary

    # 9. wa-service npm deps
    print_info "Installing wa-service dependencies..."
    npm --prefix "$_APP_DIR" install --omit=dev --quiet

    # 10. .env file
    if [ ! -f "$_APP_DIR/.env" ]; then
        print_info "Creating .env template..."
        cat > "$_APP_DIR/.env" <<'ENV'
# === Go API ===
APP_ENV=production
PORT=8080
APP_TIMEZONE=Asia/Jakarta

# === Database ===
# Format: mysql://user:password@tcp(host:port)/dbname?parseTime=true&loc=Asia%2FJakarta
DATABASE_URL=mysql://salfanet:CHANGE_ME@tcp(127.0.0.1:3306)/salfanet_radius?parseTime=true&loc=Asia%2FJakarta

# === JWT ===
JWT_SECRET=CHANGE_ME_TO_RANDOM_64_CHAR_STRING

# === CORS (comma-separated list of allowed origins) ===
CORS_ORIGINS=https://yourdomain.com,http://localhost:3000

# === WhatsApp sidecar ===
WA_SERVICE_URL=http://localhost:3001

# === Next.js Frontend (separate .env for Next.js in same repo) ===
# NODE_ENV=production
# NEXTAUTH_SECRET=CHANGE_ME
# NEXTAUTH_URL=https://yourdomain.com
# GO_API_URL=http://127.0.0.1:8080
ENV
        print_warning "IMPORTANT: Edit $_APP_DIR/.env with real values before starting!"
    fi

    # 11. Systemd service
    export APP_USER="$_APP_USER"
    setup_go_systemd_service

    # 12. Nginx (basic config with /api/ → Go, / → Next.js)
    local _DOMAIN="${DOMAIN:-_}"
    local _NGINX_CONF="/etc/nginx/sites-available/salfanet-radius"
    cat > "$_NGINX_CONF" <<NGINX
server {
    listen 80;
    server_name ${_DOMAIN};

    client_max_body_size 100M;
    access_log /var/log/nginx/salfanet-access.log;
    error_log  /var/log/nginx/salfanet-error.log;

    location /_next/static/ {
        alias ${_APP_DIR}/.next/static/;
        expires 365d;
        add_header Cache-Control "public, immutable";
    }

    # Go API backend
    location /api/ {
        proxy_pass         http://127.0.0.1:${APP_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host \$host;
        proxy_set_header   X-Real-IP \$remote_addr;
        proxy_set_header   X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
        proxy_read_timeout 300s;
        add_header Cache-Control 'no-store' always;
    }

    # Next.js frontend
    location / {
        proxy_pass         http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade \$http_upgrade;
        proxy_set_header   Connection "upgrade";
        proxy_set_header   Host \$host;
        proxy_set_header   X-Real-IP \$remote_addr;
        proxy_cache_bypass \$http_upgrade;
    }
}
NGINX
    ln -sf "$_NGINX_CONF" /etc/nginx/sites-enabled/salfanet-radius
    rm -f /etc/nginx/sites-enabled/default
    nginx -t && systemctl reload nginx
    print_success "Nginx configured: /api/ → Go:${APP_PORT}, / → Next.js:3000"

    # 13. UFW
    ufw allow OpenSSH ; ufw allow 'Nginx Full' ; ufw --force enable

    # 14. FreeRADIUS config
    if [ -d "$_APP_DIR/freeradius-config" ]; then
        cp -r "$_APP_DIR/freeradius-config/"* /etc/freeradius/3.0/ 2>/dev/null || true
        chown -R freerad:freerad /etc/freeradius/3.0/
        systemctl restart freeradius 2>/dev/null || print_warning "FreeRADIUS restart failed"
    fi

    # 15. File ownership + start service
    chown -R "$_APP_USER:$_APP_USER" "$_APP_DIR"
    start_go_service

    echo ""
    print_success "========================================================"
    print_success "  Salfanet RADIUS Go installed successfully!"
    print_success "========================================================"
    print_info "  Go API health: curl http://localhost:${APP_PORT}/api/health"
    print_info "  Service logs:  journalctl -u ${GO_SERVICE_NAME} -f"
    print_warning "  EDIT .env:     nano $_APP_DIR/.env"
    print_info "  Then restart:  systemctl restart ${GO_SERVICE_NAME}"
    print_success "========================================================"
}

# ─── Entry point ─────────────────────────────────────────────────────────────
# Run standalone install only when executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    _standalone_install "$@"
fi
