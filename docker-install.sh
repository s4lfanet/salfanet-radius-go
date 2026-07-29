#!/bin/bash
# ============================================================================
# SALFANET RADIUS — Docker Installer
# ============================================================================
# One-command Docker deployment for Salfanet RADIUS
#
# Usage:
#   bash docker-install.sh                    # Interactive
#   bash docker-install.sh --unattended        # Non-interactive (use defaults)
#   bash docker-install.sh --domain radius.example.com  # With domain
#
# Prerequisites:
#   - Docker Engine 24+
#   - Docker Compose v2+
#   ============================================================================

set -e
set -o pipefail

# ─── Colors ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; WHITE='\033[1;37m'; NC='\033[0m'

print_step()    { echo -e "\n${CYAN}=============================================${NC}"; echo -e "${CYAN}$1${NC}"; echo -e "${CYAN}=============================================${NC}\n"; }
print_success() { echo -e "${GREEN}[OK] $1${NC}"; }
print_info()    { echo -e "${YELLOW}[INFO] $1${NC}"; }
print_error()   { echo -e "${RED}[ERROR] $1${NC}" >&2; }
print_warning() { echo -e "${YELLOW}[WARN] $1${NC}"; }

# ─── Parse CLI args ──────────────────────────────────────────────────────────
UNATTENDED=false
DOMAIN=""
for arg in "$@"; do
    case "$arg" in
        --unattended) UNATTENDED=true ;;
        --domain)     shift; DOMAIN="$1" ;;
        --help|-h)
            echo "Usage: bash docker-install.sh [--unattended] [--domain DOMAIN]"
            exit 0
            ;;
    esac
    shift 2>/dev/null || true
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env.docker"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.full.yml"

# ─── Pre-flight checks ───────────────────────────────────────────────────────
check_prerequisites() {
    print_step "Checking prerequisites"

    if ! command -v docker &>/dev/null; then
        print_error "Docker is not installed!"
        print_info "Install: curl -fsSL https://get.docker.com | sh"
        exit 1
    fi
    print_success "Docker found: $(docker --version)"

    if ! docker compose version &>/dev/null; then
        print_error "Docker Compose v2 is not installed!"
        print_info "Install: sudo apt-get install docker-compose-plugin"
        exit 1
    fi
    print_success "Docker Compose: $(docker compose version)"

    # Check if user is in docker group or root
    if [ "$EUID" -ne 0 ] && ! docker ps &>/dev/null 2>&1; then
        print_error "Cannot run docker commands. Run as root or add user to docker group:"
        print_info "  sudo usermod -aG docker \$USER && newgrp docker"
        exit 1
    fi
}

# ─── Generate random secrets ─────────────────────────────────────────────────
gen_secret() {
    head -c 32 /dev/urandom | base64 | tr -d '\n/'
}

gen_hex() {
    openssl rand -hex 16
}

gen_password() {
    head -c 16 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 20
}

# ─── Detect server IP ────────────────────────────────────────────────────────
detect_ip() {
    local IP=""
    IP=$(curl -s --connect-timeout 5 https://api.ipify.org 2>/dev/null) || \
    IP=$(hostname -I 2>/dev/null | awk '{print $1}') || \
    IP="127.0.0.1"
    echo "$IP"
}

# ─── Create .env.docker ──────────────────────────────────────────────────────
create_env_file() {
    print_step "Creating .env.docker configuration"

    if [ -f "$ENV_FILE" ]; then
        print_warning ".env.docker already exists"
        if [ "$UNATTENDED" = "true" ]; then
            print_info "Using existing .env.docker"
            return 0
        fi
        read -p "Overwrite? [y/N]: " OVERWRITE </dev/tty || OVERWRITE="n"
        [[ "$OVERWRITE" =~ ^[Yy]$ ]] || { print_info "Keeping existing .env.docker"; return 0; }
    fi

    local SERVER_IP=$(detect_ip)
    local APP_URL="http://${SERVER_IP}"

    if [ -n "$DOMAIN" ]; then
        APP_URL="https://${DOMAIN}"
    elif [ "$UNATTENDED" = "false" ]; then
        read -p "Domain (empty = use IP ${SERVER_IP}): " DOMAIN_INPUT </dev/tty || DOMAIN_INPUT=""
        if [ -n "$DOMAIN_INPUT" ]; then
            DOMAIN="$DOMAIN_INPUT"
            APP_URL="https://${DOMAIN}"
        fi
    fi

    local DB_PASS=$(gen_password)
    local ROOT_PASS=$(gen_password)
    local JWT=$(gen_secret)
    local NEXTAUTH=$(gen_secret)
    local AGENT_JWT=$(gen_secret)
    local ENC_KEY=$(gen_hex)
    local RADIUS_SECRET=$(gen_password)

    cat > "$ENV_FILE" <<EOF
# SALFANET RADIUS — Docker Configuration
# Generated: $(date)

# Database
DB_NAME=salfanet_radius
DB_USER=salfanet_user
DB_PASSWORD=${DB_PASS}
DB_ROOT_PASSWORD=${ROOT_PASS}

# JWT & Auth
JWT_SECRET=${JWT}
NEXTAUTH_SECRET=${NEXTAUTH}
AGENT_JWT_SECRET=${AGENT_JWT}
ENCRYPTION_KEY=${ENC_KEY}

# Application
APP_BASE_URL=${APP_URL}
RADIUS_SERVER_IP=${SERVER_IP}
CORS_ORIGINS=http://${SERVER_IP}$([ -n "$DOMAIN" ] && echo ",https://${DOMAIN}" || true)

# RADIUS
RADIUS_SHARED_SECRET=${RADIUS_SECRET}

# VAPID (generate later with: npx web-push generate-vapid-keys --json)
VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=
VAPID_EMAIL=admin@${DOMAIN:-${SERVER_IP}}

# GenieACS (optional)
GENIEACS_URL=
GENIEACS_USERNAME=
GENIEACS_PASSWORD=

# Logging
LOG_LEVEL=error
EOF

    chmod 600 "$ENV_FILE"
    print_success ".env.docker created with generated secrets"
    print_info "Database password: ${DB_PASS}"
    print_info "Review: nano $ENV_FILE"
}

# ─── Build & Start ───────────────────────────────────────────────────────────
build_and_start() {
    print_step "Building and starting Docker containers"

    print_info "Pulling base images..."
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" pull 2>/dev/null || true

    print_info "Building custom images..."
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build

    print_info "Starting services..."
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

    print_success "All services started"
}

# ─── Database initialization ──────────────────────────────────────────────────
init_database() {
    print_step "Initializing database"

    # Wait for MySQL to be ready
    print_info "Waiting for MySQL to be ready..."
    local retries=30
    while [ $retries -gt 0 ]; do
        if docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T db \
            mysqladmin ping -h localhost -p"$(grep DB_ROOT_PASSWORD "$ENV_FILE" | cut -d= -f2)" &>/dev/null; then
            print_success "MySQL is ready"
            break
        fi
        sleep 2
        retries=$((retries - 1))
    done

    if [ $retries -eq 0 ]; then
        print_error "MySQL did not become ready in time"
        return 1
    fi

    # Run Prisma migrations via frontend container
    print_info "Running database schema setup..."
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T frontend \
        npx prisma db push --accept-data-loss 2>/dev/null || {
        print_warning "Prisma db push failed — may need to run manually"
    }

    print_info "Seeding database..."
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T frontend \
        npx tsx prisma/seeds/seed-all.ts 2>/dev/null || {
        print_warning "Database seeding had issues — check logs"
    }

    print_success "Database initialized"
}

# ─── Show status ──────────────────────────────────────────────────────────────
show_status() {
    print_step "Service Status"

    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps

    echo ""
    local SERVER_IP=$(grep RADIUS_SERVER_IP "$ENV_FILE" | cut -d= -f2)
    local DOMAIN=$(grep VAPID_EMAIL "$ENV_FILE" | cut -d@ -f2)

    echo -e "${CYAN}Access Information:${NC}"
    echo -e "  Web UI:    http://${SERVER_IP}"
    echo -e "  API:       http://${SERVER_IP}/api/health"
    echo ""
    echo -e "${CYAN}Management Commands:${NC}"
    echo "  Status:   docker compose --env-file .env.docker -f docker-compose.full.yml ps"
    echo "  Logs:     docker compose --env-file .env.docker -f docker-compose.full.yml logs -f"
    echo "  Restart:  docker compose --env-file .env.docker -f docker-compose.full.yml restart"
    echo "  Stop:     docker compose --env-file .env.docker -f docker-compose.full.yml down"
    echo "  Update:   git pull && docker compose --env-file .env.docker -f docker-compose.full.yml up -d --build"
    echo ""
    echo -e "${CYAN}RADIUS Ports:${NC}"
    echo "  Auth:     1812/udp"
    echo "  Acct:     1813/udp"
    echo "  CoA:      3799/udp"
    echo ""
    echo -e "${YELLOW}IMPORTANT: Configure your NAS/router with:${NC}"
    echo "  RADIUS Server IP: ${SERVER_IP}"
    echo "  Shared Secret:    (see .env.docker → RADIUS_SHARED_SECRET)"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Access the web UI and login as admin"
    echo "  2. Configure your NAS/router in RADIUS settings"
    echo "  3. Generate VAPID keys: docker compose exec frontend npx web-push generate-vapid-keys"
}

# ─── Main ────────────────────────────────────────────────────────────────────
main() {
    echo ""
    echo -e "${CYAN}=============================================${NC}"
    echo -e "${CYAN}  SALFANET RADIUS — Docker Installer${NC}"
    echo -e "${CYAN}=============================================${NC}"
    echo ""

    check_prerequisites
    create_env_file
    build_and_start
    init_database
    show_status

    print_success "Installation complete!"
}

main "$@"
