#!/usr/bin/env bash

# ==============================================================================
# project.sh - Script Otomasi Isolasi User & Manajemen Aplikasi Multi-Tenant
# Mendukung Multi-OS (Arch Linux, Ubuntu, Debian, Fedora, RHEL)
# ==============================================================================

set -eo pipefail

REGISTRY_DIR="/etc/project-manager"
REGISTRY_FILE="${REGISTRY_DIR}/apps.json"
CONFIG_FILE="${REGISTRY_DIR}/config.json"
APPS_BASE_DIR="/home/apps"

# ------------------------------------------------------------------------------
# Warna & Format Output
# ------------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ------------------------------------------------------------------------------
# Helper Functions
# ------------------------------------------------------------------------------
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Script ini memerlukan hak akses root. Jalankan dengan: sudo $0 $@"
        exit 1
    fi
}

cleanup_aborted_app() {
    local target_app="$1"
    if [ -n "$target_app" ]; then
        log_warn "Membersihkan resource yang sempat terbuat untuk '$target_app'..."
        run_systemctl_user "$target_app" "stop" "${target_app}.service" 2>/dev/null || true
        run_systemctl_user "$target_app" "disable" "${target_app}.service" 2>/dev/null || true
        loginctl disable-linger "$target_app" 2>/dev/null || true
        pkill -u "$target_app" 2>/dev/null || true
        if id "$target_app" &>/dev/null; then
            userdel -r "$target_app" 2>/dev/null || userdel -f "$target_app" 2>/dev/null || true
        fi
        if [ -d "${APPS_BASE_DIR}/${target_app}" ]; then
            rm -rf "${APPS_BASE_DIR}/${target_app}"
        fi
    fi
}

init_registry() {
    if [ ! -d "$REGISTRY_DIR" ] && [ "$EUID" -eq 0 ]; then
        mkdir -p "$REGISTRY_DIR"
    fi
    if [ ! -f "$REGISTRY_FILE" ] && [ "$EUID" -eq 0 ]; then
        echo "[]" > "$REGISTRY_FILE"
    fi
    if [ ! -f "$CONFIG_FILE" ] && [ "$EUID" -eq 0 ]; then
        echo "{}" > "$CONFIG_FILE"
    fi
    if [ ! -d "$APPS_BASE_DIR" ] && [ "$EUID" -eq 0 ]; then
        mkdir -p "$APPS_BASE_DIR"
        chmod 755 "$APPS_BASE_DIR"
    fi
}

# ------------------------------------------------------------------------------
# Deteksi Sistem Operasi & Helper Paket
# ------------------------------------------------------------------------------
detect_os() {
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        local os_id="${ID:-}"
        local os_like="${ID_LIKE:-}"
        if [[ "$os_id" =~ (arch|manjaro|endeavouros|artix|garuda) ]] || [[ "$os_like" =~ arch ]]; then
            echo "arch"
        elif [[ "$os_id" =~ (ubuntu|debian|pop|linuxmint|kali|raspbian|elementary) ]] || [[ "$os_like" =~ (debian|ubuntu) ]]; then
            echo "debian"
        elif [[ "$os_id" =~ (fedora|rhel|centos|rocky|almalinux) ]] || [[ "$os_like" =~ (fedora|rhel) ]]; then
            echo "fedora"
        else
            echo "$os_id"
        fi
    else
        echo "unknown"
    fi
}

detect_php_fpm_socket() {
    local possible_socks=(
        "/run/php/php8.4-fpm.sock"
        "/run/php/php8.3-fpm.sock"
        "/run/php/php8.2-fpm.sock"
        "/run/php/php8.1-fpm.sock"
        "/run/php/php-fpm.sock"
        "/run/php-fpm/php-fpm.sock"
        "/run/php-fpm/www.sock"
        "/var/run/php/php8.3-fpm.sock"
        "/var/run/php/php8.2-fpm.sock"
        "/var/run/php/php-fpm.sock"
    )
    for sock in "${possible_socks[@]}"; do
        if [ -S "$sock" ] || [ -e "$sock" ]; then
            echo "$sock"
            return 0
        fi
    done

    local find_sock
    find_sock=$(find /run/php /run/php-fpm /var/run/php -type s 2>/dev/null | head -n 1 || true)
    if [ -n "$find_sock" ]; then
        echo "$find_sock"
        return 0
    fi

    local os
    os="$(detect_os)"
    if [ "$os" = "arch" ]; then
        echo "/run/php-fpm/php-fpm.sock"
    elif [ "$os" = "debian" ]; then
        local php_v
        php_v=$(php -r 'echo PHP_MAJOR_VERSION.".".PHP_MINOR_VERSION;' 2>/dev/null || echo "8.3")
        echo "/run/php/php${php_v}-fpm.sock"
    else
        echo "/run/php/php8.3-fpm.sock"
    fi
}

detect_php_fpm_service() {
    local candidates=(
        "php8.4-fpm"
        "php8.3-fpm"
        "php8.2-fpm"
        "php8.1-fpm"
        "php-fpm"
        "php7.4-fpm"
    )
    for svc in "${candidates[@]}"; do
        if systemctl list-unit-files "${svc}.service" &>/dev/null && systemctl list-unit-files "${svc}.service" | grep -q "${svc}"; then
            echo "$svc"
            return 0
        fi
    done

    local os
    os="$(detect_os)"
    if [ "$os" = "arch" ]; then
        echo "php-fpm"
    else
        local php_v
        php_v=$(php -r 'echo PHP_MAJOR_VERSION.".".PHP_MINOR_VERSION;' 2>/dev/null || echo "8.3")
        echo "php${php_v}-fpm"
    fi
}

# ------------------------------------------------------------------------------
# Config & Registry Management (JSON with node/python3/jq fallback)
# ------------------------------------------------------------------------------
get_config_value() {
    local key="$1"
    local default_val="$2"
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "$default_val"
        return 0
    fi

    if command -v node &>/dev/null; then
        node -e "
            try {
                const c = require('${CONFIG_FILE}');
                console.log(c['${key}'] !== undefined ? c['${key}'] : '${default_val}');
            } catch(e){ console.log('${default_val}'); }
        " 2>/dev/null || echo "$default_val"
    elif command -v python3 &>/dev/null; then
        python3 -c "
import json
try:
    data = json.load(open('${CONFIG_FILE}'))
    print(data.get('${key}', '${default_val}'))
except:
    print('${default_val}')
" 2>/dev/null || echo "$default_val"
    elif command -v jq &>/dev/null; then
        local val
        val=$(jq -r ".${key} // \"${default_val}\"" "$CONFIG_FILE" 2>/dev/null || echo "$default_val")
        echo "${val:-$default_val}"
    else
        local val
        val=$(grep -o "\"${key}\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "$CONFIG_FILE" 2>/dev/null | head -n 1 | cut -d'"' -f4 || true)
        echo "${val:-$default_val}"
    fi
}

save_config_value() {
    local key="$1"
    local value="$2"
    init_registry
    if command -v node &>/dev/null; then
        node -e "
            const fs = require('fs');
            const file = '${CONFIG_FILE}';
            let data = {};
            try {
                if (fs.existsSync(file)) {
                    data = JSON.parse(fs.readFileSync(file, 'utf8'));
                }
            } catch(e){}
            data['${key}'] = '${value}';
            fs.writeFileSync(file, JSON.stringify(data, null, 2));
        " 2>/dev/null || true
    elif command -v python3 &>/dev/null; then
        python3 -c "
import json, os
file = '${CONFIG_FILE}'
data = {}
if os.path.exists(file):
    try: data = json.load(open(file))
    except: pass
data['${key}'] = '${value}'
with open(file, 'w') as f:
    json.dump(data, f, indent=2)
" 2>/dev/null || true
    fi
}

save_app_metadata() {
    local json_payload="$1"
    init_registry
    if command -v node &>/dev/null; then
        node -e "
            const fs = require('fs');
            const file = '${REGISTRY_FILE}';
            let data = [];
            try {
                if (fs.existsSync(file)) {
                    data = JSON.parse(fs.readFileSync(file, 'utf8'));
                }
            } catch (e) {
                data = [];
            }
            const newApp = ${json_payload};
            data = data.filter(a => a.name !== newApp.name);
            data.push(newApp);
            fs.writeFileSync(file, JSON.stringify(data, null, 2));
        "
    elif command -v python3 &>/dev/null; then
        python3 -c "
import json, os
file = '${REGISTRY_FILE}'
data = []
if os.path.exists(file):
    try: data = json.load(open(file))
    except: data = []
new_app = json.loads('''${json_payload}''')
data = [a for a in data if a.get('name') != new_app.get('name')]
data.append(new_app)
with open(file, 'w') as f:
    json.dump(data, f, indent=2)
"
    fi
}

remove_app_metadata() {
    local app_name="$1"
    init_registry
    if command -v node &>/dev/null; then
        node -e "
            const fs = require('fs');
            const file = '${REGISTRY_FILE}';
            let data = [];
            try {
                if (fs.existsSync(file)) {
                    data = JSON.parse(fs.readFileSync(file, 'utf8'));
                    data = data.filter(a => a.name !== '${app_name}');
                    fs.writeFileSync(file, JSON.stringify(data, null, 2));
                }
            } catch (e) {}
        "
    elif command -v python3 &>/dev/null; then
        python3 -c "
import json, os
file = '${REGISTRY_FILE}'
if os.path.exists(file):
    try:
        data = json.load(open(file))
        data = [a for a in data if a.get('name') != '${app_name}']
        with open(file, 'w') as f:
            json.dump(data, f, indent=2)
    except: pass
"
    fi
}

get_app_metadata() {
    local app_name="$1"
    init_registry
    if command -v node &>/dev/null; then
        node -e "
            const fs = require('fs');
            const file = '${REGISTRY_FILE}';
            try {
                if (fs.existsSync(file)) {
                    const data = JSON.parse(fs.readFileSync(file, 'utf8'));
                    const app = data.find(a => a.name === '${app_name}');
                    if (app) {
                        console.log(JSON.stringify(app));
                    }
                }
            } catch (e) {}
        "
    elif command -v python3 &>/dev/null; then
        python3 -c "
import json, os
file = '${REGISTRY_FILE}'
if os.path.exists(file):
    try:
        data = json.load(open(file))
        app = next((a for a in data if a.get('name') == '${app_name}'), None)
        if app:
            print(json.dumps(app))
    except: pass
"
    fi
}

# ------------------------------------------------------------------------------
# Systemd User Management Helpers
# ------------------------------------------------------------------------------
run_systemctl_user() {
    local username="$1"
    local action="$2"
    local service="$3"

    local user_uid
    user_uid=$(id -u "$username" 2>/dev/null || true)
    if [ -z "$user_uid" ]; then
        return 1
    fi

    # Coba via machinectl / systemctl -M
    if systemctl --user -M "${username}@" "$action" "$service" &>/dev/null; then
        return 0
    fi

    # Fallback via XDG_RUNTIME_DIR dan runuser
    local runtime_dir="/run/user/${user_uid}"
    if [ -d "$runtime_dir" ]; then
        runuser -u "$username" -- env XDG_RUNTIME_DIR="$runtime_dir" DBUS_SESSION_BUS_ADDRESS="unix:path=${runtime_dir}/bus" systemctl --user "$action" "$service" 2>/dev/null && return 0
    fi

    # Fallback runuser biasa
    runuser -u "$username" -- systemctl --user "$action" "$service" 2>/dev/null || return 1
}

# ------------------------------------------------------------------------------
# Command: HELP
# ------------------------------------------------------------------------------
cmd_help() {
    echo -e "${BOLD}${CYAN}=================================================================${NC}"
    echo -e "${BOLD}${CYAN}            PROJECT MANAGER - ISOLASI USER AUTOMATION           ${NC}"
    echo -e "${BOLD}${CYAN}=================================================================${NC}"
    echo -e "${BOLD}PENGGUNAAN:${NC}"
    echo -e "  sudo $0 [command] [argumen...]\n"
    echo -e "${BOLD}DAFTAR PERINTAH:${NC}"
    echo -e "  ${GREEN}help${NC}                   : Menampilkan daftar opsi command ini"
    echo -e "  ${GREEN}setup [opsi...]${NC}        : Setup dependensi server (PHP, Composer, Node.js, NPM,"
    echo -e "                           Caddy, Nginx, MariaDB, Fish) & konfigurasi PHP-FPM FastCGI."
    echo -e "                           Opsi: --php, --node, --web, --db, --fastcgi, --all (default)"
    echo -e "  ${GREEN}create${NC}                 : Membuat user terisolasi, direktori home, database,"
    echo -e "                           dan konfigurasi web server/service aplikasi"
    echo -e "  ${GREEN}delete <nama>${NC}          : Menghapus user, folder home, database, dan service"
    echo -e "  ${GREEN}list${NC}                   : Menampilkan daftar aplikasi berserta info database & status"
    echo -e "  ${GREEN}logs <nama>${NC}            : Menampilkan log systemd service aplikasi"
    echo -e "  ${GREEN}manage <nama> <opsi>${NC}   : Mengelola service aplikasi"
    echo -e "                           Pilihan opsi: ${YELLOW}restart${NC}, ${YELLOW}stop${NC}, ${YELLOW}start${NC}, ${YELLOW}status${NC}\n"
    echo -e "${BOLD}STACK YANG DIDUKUNG:${NC}"
    echo -e "  1. Laravel (PHP-FPM: Unix Socket / TCP Port)"
    echo -e "  2. Node.js (Fullstack / Static FE + API BE)"
    echo -e "  3. Node.js (Standalone API Only - Direct Node Runtime)\n"
    echo -e "${BOLD}WEB SERVER YANG DIDUKUNG (Stack 1 & 2):${NC}"
    echo -e "  1. Caddy (Ringkas & zero-temp folder)"
    echo -e "  2. Nginx (User-space instance)\n"
    echo -e "${BOLD}SISTEM OPERASI YANG DIDUKUNG:${NC}"
    echo -e "  - Arch Linux / Manjaro / EndeavourOS (pacman)"
    echo -e "  - Ubuntu / Debian (apt)"
    echo -e "  - Fedora / RHEL / AlmaLinux / Rocky (dnf)\n"
}

show_setup_help() {
    echo -e "${BOLD}PENGGUNAAN:${NC}"
    echo -e "  sudo $0 setup [opsi...]\n"
    echo -e "${BOLD}DESKRIPSI:${NC}"
    echo -e "  Jika tanpa opsi, semua tahap setup akan dijalankan secara otomatis."
    echo -e "  Jika diberikan opsi, hanya tahap yang dipilih yang akan dijalankan.\n"
    echo -e "${BOLD}DAFTAR OPSI TAHAP SETUP:${NC}"
    echo -e "  ${GREEN}--all${NC}, ${GREEN}all${NC}               : Jalankan semua tahap setup (default)"
    echo -e "  ${GREEN}--php${NC}, ${GREEN}php${NC}               : Install PHP, ekstensi umum, dan Composer"
    echo -e "  ${GREEN}--node${NC}, ${GREEN}node${NC}             : Install Node.js dan NPM"
    echo -e "  ${GREEN}--web${NC}, ${GREEN}web${NC}               : Install Web Server (Caddy & Nginx)"
    echo -e "  ${GREEN}--db${NC}, ${GREEN}db${NC}                 : Install Fish Shell dan MariaDB/MySQL Client"
    echo -e "  ${GREEN}--fastcgi${NC}, ${GREEN}fastcgi${NC}       : Konfigurasi koneksi PHP-FPM FastCGI (Socket / TCP)"
    echo -e "  ${GREEN}--interactive${NC}, ${GREEN}-i${NC}        : Jalankan setup melalui menu interaktif"
    echo -e "  ${GREEN}--help${NC}, ${GREEN}-h${NC}               : Menampilkan bantuan opsi setup ini\n"
    echo -e "${BOLD}CONTOH PENGGUNAAN:${NC}"
    echo -e "  sudo $0 setup                 # Menjalankan seluruh tahap"
    echo -e "  sudo $0 setup --php           # Hanya setup PHP & Composer"
    echo -e "  sudo $0 setup php node        # Setup PHP dan Node.js"
    echo -e "  sudo $0 setup --web --fastcgi # Setup Web Server dan FastCGI"
}

# ------------------------------------------------------------------------------
# Installer Functions for Setup
# ------------------------------------------------------------------------------
install_php_and_composer() {
    local os="$1"
    log_info "Menginstal PHP, ekstensi umum, dan Composer untuk OS: ${os}..."

    case "$os" in
        arch)
            pacman -Sy --noconfirm --needed php php-fpm php-gd php-intl php-sodium php-sqlite composer curl git unzip

            # Aktifkan ekstensi umum pada /etc/php/php.ini jika ada tanda komentar ';'
            if [ -f /etc/php/php.ini ]; then
                log_info "Mengaktifkan ekstensi PHP umum pada /etc/php/php.ini..."
                sed -i 's/^;extension=curl/extension=curl/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=pdo_mysql/extension=pdo_mysql/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=mysqli/extension=mysqli/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=mbstring/extension=mbstring/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=openssl/extension=openssl/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=zip/extension=zip/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=fileinfo/extension=fileinfo/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=gd/extension=gd/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=intl/extension=intl/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=sodium/extension=sodium/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=iconv/extension=iconv/' /etc/php/php.ini 2>/dev/null || true
                sed -i 's/^;extension=bcmath/extension=bcmath/' /etc/php/php.ini 2>/dev/null || true
            fi
            ;;
        debian)
            apt-get update
            DEBIAN_FRONTEND=noninteractive apt-get install -y \
                php-cli php-fpm php-mysql php-mbstring php-xml php-curl php-zip php-intl php-bcmath php-gd php-sqlite3 \
                curl git unzip composer || {
                    log_warn "Composer via apt gagal/tidak tersedia. Mengunduh installer resmi Composer..."
                    curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer || true
                }
            ;;
        fedora)
            dnf install -y php php-fpm php-mysqlnd php-mbstring php-xml php-curl php-zip php-intl php-bcmath php-gd composer curl git unzip
            ;;
        *)
            log_error "OS '${os}' belum didukung secara otomatis untuk instalasi PHP."
            return 1
            ;;
    esac

    log_success "PHP dan Composer berhasil diinstal!"
    php -v | head -n 1
    composer --version 2>/dev/null || true
}

install_nodejs_and_npm() {
    local os="$1"
    log_info "Menginstal Node.js dan NPM untuk OS: ${os}..."

    case "$os" in
        arch)
            pacman -Sy --noconfirm --needed nodejs npm
            ;;
        debian)
            apt-get update
            DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs npm
            ;;
        fedora)
            dnf install -y nodejs npm
            ;;
        *)
            log_error "OS '${os}' belum didukung secara otomatis untuk instalasi Node.js."
            return 1
            ;;
    esac

    log_success "Node.js dan NPM berhasil diinstal!"
    node -v 2>/dev/null || true
    npm -v 2>/dev/null || true
}

install_web_servers() {
    local os="$1"
    log_info "Menginstal Web Server (Caddy & Nginx) untuk OS: ${os}..."

    case "$os" in
        arch)
            pacman -Sy --noconfirm --needed caddy nginx
            ;;
        debian)
            apt-get update
            DEBIAN_FRONTEND=noninteractive apt-get install -y nginx

            if ! command -v caddy &>/dev/null; then
                log_info "Mengonfigurasi repositori resmi Caddy untuk Debian/Ubuntu..."
                DEBIAN_FRONTEND=noninteractive apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl gnupg
                curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg --yes 2>/dev/null || true
                curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null || true
                apt-get update
                DEBIAN_FRONTEND=noninteractive apt-get install -y caddy || log_warn "Instalasi Caddy via apt gagal. Anda dapat menginstalnya secara manual."
            fi
            ;;
        fedora)
            dnf install -y caddy nginx
            ;;
        *)
            log_error "OS '${os}' belum didukung secara otomatis untuk instalasi Web Server."
            return 1
            ;;
    esac

    log_success "Web Server (Caddy & Nginx) berhasil diinstal!"
    caddy version 2>/dev/null || true
    nginx -v 2>&1 || true
}

install_shell_and_db_client() {
    local os="$1"
    log_info "Menginstal Fish Shell dan MariaDB/MySQL Client untuk OS: ${os}..."

    case "$os" in
        arch)
            pacman -Sy --noconfirm --needed fish mariadb-clients
            ;;
        debian)
            apt-get update
            DEBIAN_FRONTEND=noninteractive apt-get install -y fish default-mysql-client || DEBIAN_FRONTEND=noninteractive apt-get install -y fish mariadb-client || true
            ;;
        fedora)
            dnf install -y fish mariadb
            ;;
        *)
            log_error "OS '${os}' belum didukung secara otomatis untuk instalasi Shell & DB Client."
            return 1
            ;;
    esac

    log_success "Fish shell dan Database client berhasil diinstal!"
}

configure_php_fpm_connection() {
    local os="$1"
    echo -e "\n${BOLD}${CYAN}--- Konfigurasi FastCGI PHP-FPM ---${NC}"
    
    local detected_sock
    detected_sock="$(detect_php_fpm_socket)"
    local current_mode
    current_mode="$(get_config_value "php_mode" "socket")"
    local current_sock
    current_sock="$(get_config_value "php_sock_path" "$detected_sock")"
    local current_port
    current_port="$(get_config_value "php_port" "9000")"

    echo -e "Pilih mode koneksi PHP-FPM FastCGI default:"
    echo -e "  1. Unix Socket (Default/Rekomendasi, misal: ${current_sock})"
    echo -e "  2. TCP Port (misal: 127.0.0.1:${current_port})"
    
    local mode_choice=""
    while true; do
        read -rp "Pilihan mode (1/2) [default: 1]: " mode_choice
        mode_choice="${mode_choice:-1}"
        case "$mode_choice" in
            1)
                read -rp "Masukkan path Unix Socket PHP-FPM [default: ${current_sock}]: " chosen_sock
                chosen_sock="${chosen_sock:-$current_sock}"
                save_config_value "php_mode" "socket"
                save_config_value "php_sock_path" "$chosen_sock"
                log_success "Mode PHP-FPM disetel ke Unix Socket: ${chosen_sock}"
                break
                ;;
            2)
                read -rp "Masukkan port TCP PHP-FPM [default: ${current_port}]: " chosen_port
                chosen_port="${chosen_port:-$current_port}"
                save_config_value "php_mode" "port"
                save_config_value "php_port" "$chosen_port"
                log_success "Mode PHP-FPM disetel ke TCP Port: 127.0.0.1:${chosen_port}"
                break
                ;;
            *)
                log_error "Pilihan tidak valid. Masukkan 1 atau 2."
                ;;
        esac
    done

    # Service activation
    local php_svc
    php_svc="$(detect_php_fpm_service)"
    log_info "Mengaktifkan dan menjalankan service PHP-FPM '${php_svc}'..."
    if systemctl enable --now "${php_svc}" &>/dev/null; then
        log_success "Service '${php_svc}' berhasil diaktifkan dan dijalankan!"
    else
        log_warn "Gagal mengaktifkan service '${php_svc}'. Pastikan PHP-FPM sudah terpasang."
    fi

    save_config_value "php_service" "$php_svc"
    save_config_value "os" "$os"
    save_config_value "updated_at" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}

# ------------------------------------------------------------------------------
# Command: SETUP
# ------------------------------------------------------------------------------
cmd_setup() {
    local os
    os="$(detect_os)"

    local run_php=false
    local run_node=false
    local run_web=false
    local run_shell_db=false
    local run_fastcgi=false
    local run_all=false
    local interactive=false

    if [ "$#" -eq 0 ]; then
        run_all=true
    else
        for arg in "$@"; do
            case "$arg" in
                --help|-h|help)
                    show_setup_help
                    return 0
                    ;;
                --all|all)
                    run_all=true
                    ;;
                --php|php|--composer|composer)
                    run_php=true
                    ;;
                --node|node|--nodejs|nodejs|--npm|npm)
                    run_node=true
                    ;;
                --web|web|--webserver|webserver|--webservers|webservers|--caddy|caddy|--nginx|nginx)
                    run_web=true
                    ;;
                --db|db|--shell|shell|--shell-db|shell-db|--fish|fish|--mariadb|mariadb|--mysql|mysql)
                    run_shell_db=true
                    ;;
                --fastcgi|fastcgi|--fpm|fpm|--php-fpm|php-fpm)
                    run_fastcgi=true
                    ;;
                --interactive|-i|interactive)
                    interactive=true
                    ;;
                *)
                    log_error "Opsi setup '$arg' tidak dikenal."
                    echo ""
                    show_setup_help
                    exit 1
                    ;;
            esac
        done
    fi

    if [ "$interactive" = true ]; then
        check_root
        init_registry

        echo -e "${BOLD}${CYAN}=================================================================${NC}"
        echo -e "${BOLD}${CYAN}       SETUP DEPENDENSI & FASTCGI PROJECT MANAGER MULTI-OS      ${NC}"
        echo -e "${BOLD}${CYAN}=================================================================${NC}"
        echo -e "Distro / OS Terdeteksi : ${BOLD}${GREEN}${os}${NC} ($(uname -s -r -m))"
        echo -e "Konfigurasi Disimpan Di: ${BOLD}${CONFIG_FILE}${NC}\n"

        echo -e "Pilih menu setup yang ingin dijalankan:"
        echo -e "  1. ${GREEN}Setup Lengkap (Semua Tahap)${NC}"
        echo -e "     (Install PHP + Composer + Node.js + NPM + Caddy + Nginx + Fish + DB Client + Konfigurasi FastCGI)"
        echo -e "  2. Install PHP, Ekstensi & Composer"
        echo -e "  3. Install Node.js & NPM"
        echo -e "  4. Install Web Server (Caddy & Nginx)"
        echo -e "  5. Install Fish Shell & Database Client"
        echo -e "  6. Konfigurasi PHP-FPM FastCGI (Unix Socket vs TCP Port)"
        echo -e "  7. Keluar\n"

        local setup_choice=""
        while true; do
            read -rp "Pilihan menu (1-7) [default: 1]: " setup_choice
            setup_choice="${setup_choice:-1}"
            case "$setup_choice" in
                1)
                    install_php_and_composer "$os"
                    install_nodejs_and_npm "$os"
                    install_web_servers "$os"
                    install_shell_and_db_client "$os"
                    configure_php_fpm_connection "$os"
                    break
                    ;;
                2)
                    install_php_and_composer "$os"
                    break
                    ;;
                3)
                    install_nodejs_and_npm "$os"
                    break
                    ;;
                4)
                    install_web_servers "$os"
                    break
                    ;;
                5)
                    install_shell_and_db_client "$os"
                    break
                    ;;
                6)
                    configure_php_fpm_connection "$os"
                    break
                    ;;
                7)
                    log_info "Keluar dari menu setup."
                    exit 0
                    ;;
                *)
                    log_error "Pilihan tidak valid. Masukkan angka antara 1 sampai 7."
                    ;;
            esac
        done

        echo -e "\n${BOLD}${GREEN}=================================================================${NC}"
        echo -e "${BOLD}${GREEN}                   SETUP SELESAI DILAKUKAN                       ${NC}"
        echo -e "${BOLD}${GREEN}=================================================================${NC}"
        echo -e "Anda sekarang dapat menjalankan: ${BOLD}sudo $0 create${NC} untuk membuat aplikasi baru."
        return 0
    fi

    if [ "$run_all" = true ]; then
        run_php=true
        run_node=true
        run_web=true
        run_shell_db=true
        run_fastcgi=true
    fi

    check_root
    init_registry

    echo -e "${BOLD}${CYAN}=================================================================${NC}"
    echo -e "${BOLD}${CYAN}       SETUP DEPENDENSI & FASTCGI PROJECT MANAGER MULTI-OS      ${NC}"
    echo -e "${BOLD}${CYAN}=================================================================${NC}"
    echo -e "Distro / OS Terdeteksi : ${BOLD}${GREEN}${os}${NC} ($(uname -s -r -m))"
    echo -e "Konfigurasi Disimpan Di: ${BOLD}${CONFIG_FILE}${NC}\n"

    # Jalankan tahap-tahap yang dipilih
    if [ "$run_php" = true ]; then
        install_php_and_composer "$os"
    fi

    if [ "$run_node" = true ]; then
        install_nodejs_and_npm "$os"
    fi

    if [ "$run_web" = true ]; then
        install_web_servers "$os"
    fi

    if [ "$run_shell_db" = true ]; then
        install_shell_and_db_client "$os"
    fi

    if [ "$run_fastcgi" = true ]; then
        configure_php_fpm_connection "$os"
    fi

    echo -e "\n${BOLD}${GREEN}=================================================================${NC}"
    echo -e "${BOLD}${GREEN}                   SETUP SELESAI DILAKUKAN                       ${NC}"
    echo -e "${BOLD}${GREEN}=================================================================${NC}"
    echo -e "Anda sekarang dapat menjalankan: ${BOLD}sudo $0 create${NC} untuk membuat aplikasi baru."
}

# ------------------------------------------------------------------------------
# Command: CREATE
# ------------------------------------------------------------------------------
cmd_create() {
    check_root
    init_registry

    echo -e "${BOLD}${CYAN}=== Wizard Pembuatan Aplikasi Terisolasi ===${NC}\n"

    # 1. Input Nama Aplikasi
    local app_name=""
    while true; do
        read -rp "Masukkan nama aplikasi (username): " app_name
        app_name="$(echo "$app_name" | tr -d '[:space:]')"
        if [[ -z "$app_name" ]]; then
            log_error "Nama aplikasi tidak boleh kosong."
        elif [[ ! "$app_name" =~ ^[a-z0-9_-]+$ ]]; then
            log_error "Nama aplikasi hanya boleh mengandung huruf kecil, angka, garis bawah (_), dan tanda hubung (-)."
        elif id "$app_name" &>/dev/null; then
            log_error "User '$app_name' sudah ada pada sistem. Silakan pilih nama lain."
        elif [ -d "${APPS_BASE_DIR}/${app_name}" ]; then
            log_error "Direktori ${APPS_BASE_DIR}/${app_name} sudah ada. Silakan gunakan nama lain."
        else
            break
        fi
    done

    # 2. Input Password User
    local user_password=""
    local user_password_confirm=""
    while true; do
        read -rsp "Masukkan password untuk user '$app_name': " user_password
        echo
        if [[ -z "$user_password" ]]; then
            log_error "Password tidak boleh kosong."
            continue
        fi
        read -rsp "Konfirmasi password: " user_password_confirm
        echo
        if [[ "$user_password" != "$user_password_confirm" ]]; then
            log_error "Konfirmasi password tidak cocok. Silakan coba lagi."
        else
            break
        fi
    done

    # 3. Pilihan Stack
    echo -e "\nPilih Stack Aplikasi:"
    echo -e "  1. Laravel (PHP-FPM)"
    echo -e "  2. Node.js (Fullstack / Static FE + API BE)"
    echo -e "  3. Node.js (Standalone API Only - Direct Node Runtime)"
    local stack_choice=""
    local stack_name=""
    local stack_type=""
    while true; do
        read -rp "Pilihan (1/2/3): " stack_choice
        case "$stack_choice" in
            1) stack_name="Laravel (PHP-FPM)"; stack_type="laravel"; break ;;
            2) stack_name="Node.js (Fullstack / Static FE + API BE)"; stack_type="node-fullstack"; break ;;
            3) stack_name="Node.js (Standalone API Only - Direct Node Runtime)"; stack_type="node-api"; break ;;
            *) log_error "Pilihan tidak valid. Masukkan 1, 2, atau 3." ;;
        esac
    done

    # 4. Link Git Repository (Opsional)
    local git_repo=""
    read -rp "Link repositori git (Opsional, tekan Enter untuk lewati): " git_repo

    # 5. Port & Web Server Configuration
    local port_fe=""
    local port_be=""
    local port_single=""
    local webserver_choice=""
    local webserver_name="None"
    local php_fpm_mode="socket"
    local php_sock_path=""
    local php_tcp_port="9000"
    local caddy_fastcgi_target=""
    local nginx_fastcgi_target=""

    if [[ "$stack_choice" == "1" ]]; then
        # Laravel (PHP-FPM) - Port Web/HTTP
        while true; do
            read -rp "Masukkan Port Web / HTTP Aplikasi (misal: 8000): " port_fe
            if [[ "$port_fe" =~ ^[0-9]+$ ]] && [ "$port_fe" -ge 1 ] && [ "$port_fe" -le 65535 ]; then
                port_single="$port_fe"
                break
            else
                log_error "Port harus berupa angka antara 1 dan 65535."
            fi
        done

        # Konfigurasi Koneksi FastCGI PHP-FPM
        local default_fpm_mode
        default_fpm_mode="$(get_config_value "php_mode" "socket")"
        local default_sock
        default_sock="$(get_config_value "php_sock_path" "")"
        if [ -z "$default_sock" ]; then
            default_sock="$(detect_php_fpm_socket)"
        fi
        local default_port
        default_port="$(get_config_value "php_port" "9000")"

        echo -e "\nPilih Mode Koneksi PHP-FPM FastCGI:"
        echo -e "  1. Unix Socket (Default: ${default_sock})"
        echo -e "  2. TCP Port (Default: 127.0.0.1:${default_port})"

        local default_fpm_choice="1"
        if [ "$default_fpm_mode" = "port" ]; then
            default_fpm_choice="2"
        fi

        local fpm_choice=""
        read -rp "Pilihan Mode FastCGI (1/2) [default: ${default_fpm_choice}]: " fpm_choice
        fpm_choice="${fpm_choice:-$default_fpm_choice}"

        if [[ "$fpm_choice" == "2" ]]; then
            php_fpm_mode="port"
            read -rp "Masukkan Port TCP PHP-FPM [default: ${default_port}]: " php_tcp_port
            php_tcp_port="${php_tcp_port:-$default_port}"
            port_be="$php_tcp_port"
            caddy_fastcgi_target="127.0.0.1:${php_tcp_port}"
            nginx_fastcgi_target="127.0.0.1:${php_tcp_port}"
        else
            php_fpm_mode="socket"
            read -rp "Masukkan Path Unix Socket PHP-FPM [default: ${default_sock}]: " php_sock_path
            php_sock_path="${php_sock_path:-$default_sock}"
            port_be="socket (${php_sock_path})"

            local clean_sock="${php_sock_path#unix:}"
            clean_sock="${clean_sock#unix/}"
            if [[ "$clean_sock" =~ ^/ ]]; then
                caddy_fastcgi_target="unix/${clean_sock}"
            else
                caddy_fastcgi_target="unix//${clean_sock}"
            fi
            nginx_fastcgi_target="unix:${clean_sock}"
        fi

        echo -e "\nPilih Web Server:"
        echo -e "  1. Caddy (Ringkas & zero-temp folder)"
        echo -e "  2. Nginx (User-space instance)"
        while true; do
            read -rp "Pilihan Web Server (1/2) [default: 1]: " webserver_choice
            webserver_choice="${webserver_choice:-1}"
            case "$webserver_choice" in
                1) webserver_name="caddy"; break ;;
                2) webserver_name="nginx"; break ;;
                *) log_error "Pilihan tidak valid. Masukkan 1 atau 2." ;;
            esac
        done

    elif [[ "$stack_choice" == "2" ]]; then
        # Node.js (Fullstack / Static FE + API BE) - Meminta Port FE dan Port BE
        while true; do
            read -rp "Masukkan Port Frontend (misal: 8080): " port_fe
            if [[ "$port_fe" =~ ^[0-9]+$ ]] && [ "$port_fe" -ge 1 ] && [ "$port_fe" -le 65535 ]; then
                break
            else
                log_error "Port Frontend harus berupa angka antara 1 dan 65535."
            fi
        done

        while true; do
            read -rp "Masukkan Port Backend API (misal: 3001): " port_be
            if [[ "$port_be" =~ ^[0-9]+$ ]] && [ "$port_be" -ge 1 ] && [ "$port_be" -le 65535 ]; then
                if [[ "$port_be" == "$port_fe" ]]; then
                    log_error "Port Backend API tidak boleh sama dengan Port Frontend."
                else
                    break
                fi
            else
                log_error "Port Backend API harus berupa angka antara 1 dan 65535."
            fi
        done

        echo -e "\nPilih Web Server:"
        echo -e "  1. Caddy (Ringkas & zero-temp folder)"
        echo -e "  2. Nginx (User-space instance)"
        while true; do
            read -rp "Pilihan Web Server (1/2) [default: 1]: " webserver_choice
            webserver_choice="${webserver_choice:-1}"
            case "$webserver_choice" in
                1) webserver_name="caddy"; break ;;
                2) webserver_name="nginx"; break ;;
                *) log_error "Pilihan tidak valid. Masukkan 1 atau 2." ;;
            esac
        done

    elif [[ "$stack_choice" == "3" ]]; then
        # Node.js (Standalone API Only - Direct Node Runtime) - Meminta 1 Port Runtime
        while true; do
            read -rp "Masukkan Port Aplikasi (misal: 3000): " port_single
            if [[ "$port_single" =~ ^[0-9]+$ ]] && [ "$port_single" -ge 1 ] && [ "$port_single" -le 65535 ]; then
                break
            else
                log_error "Port harus berupa angka antara 1 dan 65535."
            fi
        done
    fi

    # 6. Database Inputs
    echo -e "\n${BOLD}${CYAN}--- Konfigurasi Database ---${NC}"
    local db_name=""
    read -rp "Masukkan nama database [default: ${app_name}]: " db_name
    db_name="${db_name:-$app_name}"

    local db_user=""
    read -rp "Masukkan username database [default: ${app_name}]: " db_user
    db_user="${db_user:-$app_name}"

    local db_password=""
    while true; do
        read -rsp "Masukkan password database untuk user '$db_user': " db_password
        echo
        if [[ -z "$db_password" ]]; then
            log_error "Password database tidak boleh kosong."
        else
            break
        fi
    done

    echo -e "\n${YELLOW}Kredensial Root Database (untuk membuat database & user):${NC}"
    local db_root_user=""
    read -rp "Username Root Database [default: root]: " db_root_user
    db_root_user="${db_root_user:-root}"

    local db_root_pass=""
    read -rsp "Password Root Database: " db_root_pass
    echo

    # 7. Eksekusi Pembuatan Database Terlebih Dahulu (Abort jika gagal)
    local mysql_auth=(-u "$db_root_user")
    if [ -n "$db_root_pass" ]; then
        mysql_auth+=("-p${db_root_pass}")
    fi

    local mysql_cmd=""
    if command -v mariadb &>/dev/null; then
        mysql_cmd="mariadb"
    elif command -v mysql &>/dev/null; then
        mysql_cmd="mysql"
    else
        log_error "Binary mysql / mariadb client tidak ditemukan di PATH sistem. Database tidak dapat dibuat."
        log_error "Silakan jalankan 'sudo $0 setup' terlebih dahulu untuk menginstal dependensi."
        exit 1
    fi

    log_info "Memverifikasi koneksi root database..."
    local root_test_err
    if ! root_test_err=$("${mysql_cmd}" "${mysql_auth[@]}" -e "SELECT 1;" 2>&1); then
        log_error "Gagal terhubung ke database sebagai root ('$db_root_user')."
        echo -e "${RED}[Detail MySQL Error]:${NC} $root_test_err"
        log_error "Proses pembuatan aplikasi dibatalkan."
        exit 1
    fi

    log_info "Membuat database '${db_name}' dan user database '${db_user}'..."
    local sql_query="
        CREATE DATABASE IF NOT EXISTS \`${db_name}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
        CREATE USER IF NOT EXISTS '${db_user}'@'localhost' IDENTIFIED BY '${db_password}';
        CREATE USER IF NOT EXISTS '${db_user}'@'%' IDENTIFIED BY '${db_password}';
        GRANT ALL PRIVILEGES ON \`${db_name}\`.* TO '${db_user}'@'localhost';
        GRANT ALL PRIVILEGES ON \`${db_name}\`.* TO '${db_user}'@'%';
        FLUSH PRIVILEGES;
    "

    local db_exec_err
    if ! db_exec_err=$("${mysql_cmd}" "${mysql_auth[@]}" -e "$sql_query" 2>&1); then
        log_error "Gagal membuat database '${db_name}' atau user '${db_user}'."
        echo -e "${RED}[Detail MySQL Error]:${NC} $db_exec_err"
        if echo "$db_exec_err" | grep -qiE "password.*policy|policy.*requirements|validate_password|password validation"; then
            log_warn "Petunjuk: Password database untuk user '${db_user}' tidak memenuhi policy keamanan MySQL server."
        fi
        log_error "Proses pembuatan aplikasi dibatalkan (abort)."
        exit 1
    fi
    log_success "Database '${db_name}' dan hak akses user '${db_user}' berhasil dibuat!"

    # 8. Memeriksa ketersediaan Shell Fish & Membuat User Sistem
    local shell_path
    shell_path="$(which fish 2>/dev/null || echo "/usr/bin/fish")"
    if [ ! -x "$shell_path" ]; then
        log_warn "Fish shell tidak ditemukan di $shell_path. Menggunakan /bin/bash sebagai cadangan."
        shell_path="/bin/bash"
    fi

    log_info "Membuat user sistem '${app_name}'..."
    useradd -m -d "${APPS_BASE_DIR}/${app_name}" -s "$shell_path" "$app_name"
    echo "${app_name}:${user_password}" | chpasswd
    log_success "User '${app_name}' berhasil dibuat dengan shell '${shell_path}'."

    log_info "Mengaktifkan linger untuk user '${app_name}'..."
    loginctl enable-linger "$app_name" || log_warn "Gagal mengaktifkan linger secara otomatis via loginctl."

    local home_dir="${APPS_BASE_DIR}/${app_name}"
    local systemd_user_dir="${home_dir}/.config/systemd/user"
    mkdir -p "$systemd_user_dir"

    # 9. Clone Git Repo jika disediakan langsung ke home folder
    if [[ -n "$git_repo" ]]; then
        log_info "Melakukan git clone dari repositori '${git_repo}' ke '${home_dir}'..."
        local tmp_clone_dir="/tmp/project_git_${app_name}_$$"
        rm -rf "$tmp_clone_dir"
        mkdir -p "$tmp_clone_dir"

        if git clone "$git_repo" "$tmp_clone_dir"; then
            log_success "Repositori berhasil di-clone."
            cp -a "$tmp_clone_dir"/. "${home_dir}/"
            rm -rf "$tmp_clone_dir"
        else
            rm -rf "$tmp_clone_dir"
            log_error "Gagal meng-clone repositori git dari '${git_repo}'."
            read -rp "Apakah Anda ingin membatalkan pembuatan aplikasi? (Y/n): " abort_clone
            if [[ "$abort_clone" != "n" && "$abort_clone" != "N" ]]; then
                "${mysql_cmd}" "${mysql_auth[@]}" -e "DROP DATABASE IF EXISTS \`${db_name}\`; DROP USER IF EXISTS '${db_user}'@'localhost'; DROP USER IF EXISTS '${db_user}'@'%'; FLUSH PRIVILEGES;" 2>/dev/null || true
                cleanup_aborted_app "$app_name"
                log_error "Pembuatan aplikasi dibatalkan."
                exit 1
            fi
        fi
    fi

    # 10. Setup Konfigurasi Web Server / Fallback Sesuai Stack
    local caddy_bin
    caddy_bin="$(command -v caddy 2>/dev/null || which caddy 2>/dev/null || echo "/usr/bin/caddy")"
    local nginx_bin
    nginx_bin="$(command -v nginx 2>/dev/null || which nginx 2>/dev/null || echo "/usr/sbin/nginx")"

    if [[ "$stack_choice" == "1" || "$stack_choice" == "2" ]]; then
        if [[ "$webserver_name" == "caddy" ]]; then
            log_info "Membuat konfigurasi Caddyfile..."
            if [[ "$stack_choice" == "1" ]]; then
                # Laravel Caddyfile (Format Bersih & Standar)
                if [ ! -d "${home_dir}/public" ]; then
                    mkdir -p "${home_dir}/public"
                fi
                if [ ! -f "${home_dir}/public/index.php" ] && [ ! -f "${home_dir}/public/index.html" ]; then
                    cat <<EOF > "${home_dir}/public/index.php"
<?php
echo "<h1>Aplikasi Laravel (${app_name}) Siap</h1><p>Menunggu deployment.</p>";
EOF
                fi
                cat <<EOF > "${home_dir}/Caddyfile"
:${port_fe} {
    root * ${home_dir}/public
    php_fastcgi ${caddy_fastcgi_target}
    file_server
    encode gzip
}
EOF
            else
                # Node.js Fullstack Caddyfile
                if [ ! -d "${home_dir}/dist" ] && [ ! -d "${home_dir}/public" ]; then
                    mkdir -p "${home_dir}/dist"
                fi
                local fe_root="${home_dir}/dist"
                if [ ! -d "${home_dir}/dist" ] && [ -d "${home_dir}/public" ]; then
                    fe_root="${home_dir}/public"
                fi
                if [ ! -f "${fe_root}/index.html" ]; then
                    cat <<EOF > "${fe_root}/index.html"
<!DOCTYPE html>
<html>
<head><title>${app_name}</title></head>
<body>
<h1>Aplikasi Node.js Fullstack (${app_name}) Siap</h1>
<p>Frontend Static + Reverse Proxy API ke port ${port_be}.</p>
</body>
</html>
EOF
                fi
                cat <<EOF > "${home_dir}/Caddyfile"
:${port_fe} {
    root * ${fe_root}
    file_server
    handle /api/* {
        reverse_proxy 127.0.0.1:${port_be}
    }
    try_files {path} /index.html
}
EOF
            fi

            # Unit file systemd untuk Caddy
            cat <<EOF > "${systemd_user_dir}/${app_name}.service"
[Unit]
Description=Caddy Web Server - %u
After=network.target

[Service]
Type=simple
WorkingDirectory=%h
ExecStart=${caddy_bin} run --config %h/Caddyfile
ExecReload=${caddy_bin} reload --config %h/Caddyfile
Restart=always

[Install]
WantedBy=default.target
EOF

        elif [[ "$webserver_name" == "nginx" ]]; then
            log_info "Membuat direktori temporary Nginx dan file nginx.conf..."
            mkdir -p "${home_dir}/tmp/"{client_body,proxy,fastcgi,uwsgi,scgi}

            local mime_types_path="/etc/nginx/mime.types"
            if [ ! -f "$mime_types_path" ]; then
                mime_types_path="$(find /etc/nginx -name "mime.types" 2>/dev/null | head -n 1 || echo "/etc/nginx/mime.types")"
            fi

            local fastcgi_params_path="/etc/nginx/fastcgi_params"
            if [ ! -f "$fastcgi_params_path" ]; then
                fastcgi_params_path="/etc/nginx/fastcgi.conf"
            fi

            if [[ "$stack_choice" == "1" ]]; then
                # Laravel Nginx
                if [ ! -d "${home_dir}/public" ]; then
                    mkdir -p "${home_dir}/public"
                fi
                if [ ! -f "${home_dir}/public/index.php" ] && [ ! -f "${home_dir}/public/index.html" ]; then
                    cat <<EOF > "${home_dir}/public/index.php"
<?php
echo "<h1>Aplikasi Laravel (${app_name}) Siap</h1><p>Menunggu deployment.</p>";
EOF
                fi
                cat <<EOF > "${home_dir}/nginx.conf"
worker_processes 1;
pid ${home_dir}/tmp/nginx.pid;
error_log ${home_dir}/tmp/error.log;

events {
    worker_connections 1024;
}

http {
    include ${mime_types_path};
    default_type application/octet-stream;
    access_log ${home_dir}/tmp/access.log;

    client_body_temp_path ${home_dir}/tmp/client_body;
    proxy_temp_path ${home_dir}/tmp/proxy;
    fastcgi_temp_path ${home_dir}/tmp/fastcgi;
    uwsgi_temp_path ${home_dir}/tmp/uwsgi;
    scgi_temp_path ${home_dir}/tmp/scgi;

    server {
        listen ${port_fe};
        server_name _;
        root ${home_dir}/public;
        index index.php index.html;

        location / {
            try_files \$uri \$uri/ /index.php?\$query_string;
        }

        location ~ \.php$ {
            fastcgi_pass ${nginx_fastcgi_target};
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
            include ${fastcgi_params_path};
        }
    }
}
EOF
            else
                # Node.js Fullstack Nginx
                if [ ! -d "${home_dir}/dist" ] && [ ! -d "${home_dir}/public" ]; then
                    mkdir -p "${home_dir}/dist"
                fi
                local fe_root="${home_dir}/dist"
                if [ ! -d "${home_dir}/dist" ] && [ -d "${home_dir}/public" ]; then
                    fe_root="${home_dir}/public"
                fi
                if [ ! -f "${fe_root}/index.html" ]; then
                    cat <<EOF > "${fe_root}/index.html"
<!DOCTYPE html>
<html>
<head><title>${app_name}</title></head>
<body>
<h1>Aplikasi Node.js Fullstack (${app_name}) Siap</h1>
<p>Frontend Static + Reverse Proxy API ke port ${port_be}.</p>
</body>
</html>
EOF
                fi
                cat <<EOF > "${home_dir}/nginx.conf"
worker_processes 1;
pid ${home_dir}/tmp/nginx.pid;
error_log ${home_dir}/tmp/error.log;

events {
    worker_connections 1024;
}

http {
    include ${mime_types_path};
    default_type application/octet-stream;
    access_log ${home_dir}/tmp/access.log;

    client_body_temp_path ${home_dir}/tmp/client_body;
    proxy_temp_path ${home_dir}/tmp/proxy;
    fastcgi_temp_path ${home_dir}/tmp/fastcgi;
    uwsgi_temp_path ${home_dir}/tmp/uwsgi;
    scgi_temp_path ${home_dir}/tmp/scgi;

    server {
        listen ${port_fe};
        server_name _;
        root ${fe_root};
        index index.html;

        location /api/ {
            proxy_pass http://127.0.0.1:${port_be}/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host \$host;
            proxy_cache_bypass \$http_upgrade;
        }

        location / {
            try_files \$uri \$uri/ /index.html;
        }
    }
}
EOF
            fi

            # Unit file systemd untuk Nginx
            cat <<EOF > "${systemd_user_dir}/${app_name}.service"
[Unit]
Description=Nginx User Web Server - %u
After=network.target

[Service]
Type=simple
WorkingDirectory=%h
ExecStartPre=/usr/bin/mkdir -p %h/tmp/client_body %h/tmp/proxy %h/tmp/fastcgi
ExecStart=${nginx_bin} -p %h -c %h/nginx.conf -g "daemon off;"
ExecReload=${nginx_bin} -p %h -c %h/nginx.conf -s reload
Restart=always

[Install]
WantedBy=default.target
EOF
        fi

    elif [[ "$stack_choice" == "3" ]]; then
        log_info "Membuat service direct Node.js runtime dan run.sh..."

        # Unit file systemd untuk Stack 3
        cat <<EOF > "${systemd_user_dir}/${app_name}.service"
[Unit]
Description=NodeJS App (Direct Runtime) - %u
After=network.target

[Service]
Type=simple
WorkingDirectory=%h
ExecStart=%h/run.sh
Restart=always
RestartSec=5s

# Alokasi environment
Environment=PORT=${port_single}
Environment=NODE_ENV=production
Environment=PATH=/usr/local/bin:/usr/bin:/bin:%h/.nvm/versions/node/current/bin

# Membaca .env milik user jika tersedia
EnvironmentFile=-%h/.env

[Install]
WantedBy=default.target
EOF

        # Script run.sh
        cat <<EOF > "${home_dir}/run.sh"
#!/bin/bash
set -e

# 1. Fallback Server jika dependensi / repo belum siap
if [ ! -f "package.json" ] && [ ! -f "backend/package.json" ] && [ ! -f "api/package.json" ] && [ ! -f "server/package.json" ]; then
    echo "[INFO] Project files not found. Starting placeholder server on port \$PORT..."
    exec node -e "
        const http = require('http');
        http.createServer((req, res) => {
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end('<h1>Application Ready</h1><p>Awaiting deployment in user home directory.</p>');
        }).listen(process.env.PORT || ${port_single}, '0.0.0.0');
    "
fi

# 2. Deteksi lokasi working directory & jalankan entry point
if [ -d "backend" ] && [ -f "backend/package.json" ]; then
    cd backend
elif [ -d "api" ] && [ -f "api/package.json" ]; then
    cd api
elif [ -d "server" ] && [ -f "server/package.json" ]; then
    cd server
fi

# 3. Jalankan aplikasi (Developer bebas mengubah baris ini jika pakai Next.js, Bun, atau TS direct)
if [ -f "dist/main.js" ]; then
    exec node dist/main.js
elif [ -f "dist/index.js" ]; then
    exec node dist/index.js
else
    exec npm start
fi
EOF
        chmod +x "${home_dir}/run.sh"
    fi

    # 11. Perbaiki Kepemilikan & Izin File
    log_info "Mengatur perizinan file..."
    chown -R "${app_name}:${app_name}" "$home_dir"
    chmod 750 "$home_dir"

    # 12. Enable & Start Systemd Service
    log_info "Memulai user service systemd '${app_name}.service'..."
    run_systemctl_user "$app_name" "daemon-reload" || true
    if run_systemctl_user "$app_name" "enable" "${app_name}.service" && run_systemctl_user "$app_name" "start" "${app_name}.service"; then
        log_success "Service '${app_name}.service' berhasil diaktifkan dan dijalankan!"
    else
        log_warn "Service dibuat, namun belum dapat dimulai otomatis (mungkin memerlukan active user bus / binary web server)."
    fi

    # 13. Simpan Metadata ke Registry
    local metadata_json
    metadata_json="{\"name\":\"${app_name}\",\"user\":\"${app_name}\",\"home\":\"${home_dir}\",\"stack\":\"${stack_type}\",\"stack_name\":\"${stack_name}\",\"webserver\":\"${webserver_name}\",\"port_fe\":\"${port_fe:-N/A}\",\"port_be\":\"${port_be:-N/A}\",\"port\":\"${port_single:-N/A}\",\"php_mode\":\"${php_fpm_mode}\",\"db_name\":\"${db_name}\",\"db_user\":\"${db_user}\",\"git_repo\":\"${git_repo:-None}\",\"created_at\":\"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"}"
    save_app_metadata "$metadata_json"

    echo -e "\n${BOLD}${GREEN}=================================================================${NC}"
    echo -e "${BOLD}${GREEN}        APLIKASI '${app_name}' BERHASIL DIBUAT!        ${NC}"
    echo -e "${BOLD}${GREEN}=================================================================${NC}"
    echo -e "  - Username / App : ${BOLD}${app_name}${NC}"
    echo -e "  - Home Directory : ${home_dir}"
    echo -e "  - Stack          : ${stack_name}"
    if [[ "$stack_choice" == "1" ]]; then
        echo -e "  - Web Server     : ${webserver_name}"
        echo -e "  - Port Web/HTTP  : ${BOLD}${port_fe}${NC}"
        echo -e "  - FastCGI Target : ${BOLD}${caddy_fastcgi_target:-$nginx_fastcgi_target}${NC}"
    elif [[ "$stack_choice" == "2" ]]; then
        echo -e "  - Web Server     : ${webserver_name}"
        echo -e "  - Port Frontend  : ${BOLD}${port_fe}${NC}"
        echo -e "  - Port Backend   : ${BOLD}${port_be}${NC}"
    elif [[ "$stack_choice" == "3" ]]; then
        echo -e "  - Port Aplikasi  : ${BOLD}${port_single}${NC}"
    fi
    echo -e "  - Database Name  : ${db_name}"
    echo -e "  - Database User  : ${db_user}"
    echo -e "  - Service Name   : ${app_name}.service (systemd user)"
    echo -e "-----------------------------------------------------------------"
}

# ------------------------------------------------------------------------------
# Command: DELETE [nama]
# ------------------------------------------------------------------------------
cmd_delete() {
    check_root
    local app_name="$1"

    if [[ -z "$app_name" ]]; then
        log_error "Parameter nama aplikasi harus diisi."
        echo -e "Penggunaan: sudo $0 delete <nama-aplikasi>"
        exit 1
    fi

    # Cek metadata atau eksistensi user
    local meta
    meta=$(get_app_metadata "$app_name")
    local db_name=""
    local db_user=""

    if [[ -n "$meta" ]]; then
        if command -v node &>/dev/null; then
            db_name=$(node -e "try { const d = ${meta}; console.log(d.db_name || ''); } catch(e){}" 2>/dev/null || true)
            db_user=$(node -e "try { const d = ${meta}; console.log(d.db_user || ''); } catch(e){}" 2>/dev/null || true)
        elif command -v python3 &>/dev/null; then
            db_name=$(python3 -c "import json; d=json.loads('''${meta}'''); print(d.get('db_name',''))" 2>/dev/null || true)
            db_user=$(python3 -c "import json; d=json.loads('''${meta}'''); print(d.get('db_user',''))" 2>/dev/null || true)
        fi
    fi

    db_name="${db_name:-$app_name}"
    db_user="${db_user:-$app_name}"

    echo -e "${BOLD}${RED}PERINGATAN:${NC} Tindakan ini akan menghapus:"
    echo -e "  1. User sistem '${app_name}' dan semua prosesnya"
    echo -e "  2. Seluruh isi direktori home '${APPS_BASE_DIR}/${app_name}'"
    echo -e "  3. Service systemd user '${app_name}.service'"
    echo -e "  4. Database '${db_name}' dan user database '${db_user}' (jika ada)"
    echo

    read -rp "Apakah Anda yakin ingin melanjutkan? (y/N): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        log_info "Penghapusan dibatalkan."
        exit 0
    fi

    log_info "Menghentikan service dan proses user '${app_name}'..."
    run_systemctl_user "$app_name" "stop" "${app_name}.service" 2>/dev/null || true
    run_systemctl_user "$app_name" "disable" "${app_name}.service" 2>/dev/null || true
    loginctl disable-linger "$app_name" 2>/dev/null || true
    pkill -u "$app_name" 2>/dev/null || true

    # Hapus user dan home directory
    log_info "Menghapus user '${app_name}' dan direktori home..."
    if id "$app_name" &>/dev/null; then
        userdel -r "$app_name" 2>/dev/null || userdel -f "$app_name" 2>/dev/null || true
    fi

    if [ -d "${APPS_BASE_DIR}/${app_name}" ]; then
        rm -rf "${APPS_BASE_DIR}/${app_name}"
    fi

    # Hapus Database
    echo -e "\n${YELLOW}Kredensial Root Database untuk menghapus database '${db_name}':${NC}"
    local db_root_user=""
    read -rp "Username Root Database [default: root]: " db_root_user
    db_root_user="${db_root_user:-root}"

    local db_root_pass=""
    read -rsp "Password Root Database: " db_root_pass
    echo

    local mysql_auth=(-u "$db_root_user")
    if [ -n "$db_root_pass" ]; then
        mysql_auth+=("-p${db_root_pass}")
    fi

    local mysql_cmd=""
    if command -v mariadb &>/dev/null; then
        mysql_cmd="mariadb"
    elif command -v mysql &>/dev/null; then
        mysql_cmd="mysql"
    fi

    if [ -n "$mysql_cmd" ]; then
        local drop_query="
            DROP DATABASE IF EXISTS \`${db_name}\`;
            DROP USER IF EXISTS '${db_user}'@'localhost';
            DROP USER IF EXISTS '${db_user}'@'%';
            FLUSH PRIVILEGES;
        "
        if "${mysql_cmd}" "${mysql_auth[@]}" -e "$drop_query" 2>/dev/null || mysql -e "$drop_query" 2>/dev/null; then
            log_success "Database '${db_name}' dan user '${db_user}' berhasil dihapus."
        else
            log_warn "Gagal menghapus database otomatis. Pastikan password root DB benar."
        fi
    fi

    remove_app_metadata "$app_name"
    log_success "Aplikasi '${app_name}' berhasil dihapus secara menyeluruh."
}

# ------------------------------------------------------------------------------
# Command: LIST
# ------------------------------------------------------------------------------
cmd_list() {
    init_registry

    echo -e "\n${BOLD}${CYAN}========================================================================================================${NC}"
    echo -e "${BOLD}${CYAN}                                       DAFTAR APLIKASI TERISOLASI                                       ${NC}"
    echo -e "${BOLD}${CYAN}========================================================================================================${NC}"

    if command -v node &>/dev/null; then
        node -e "
            const fs = require('fs');
            const { execSync } = require('child_process');
            const file = '${REGISTRY_FILE}';
            let apps = [];
            try {
                apps = JSON.parse(fs.readFileSync(file, 'utf8'));
            } catch (e) {}

            if (apps.length === 0) {
                console.log('  (Belum ada aplikasi yang dibuat)');
                console.log('========================================================================================================\n');
                process.exit(0);
            }

            const pad = (str, len) => (str + ' '.repeat(len)).slice(0, len);

            console.log(
                pad('NAMA APLIKASI', 16) +
                pad('STACK', 15) +
                pad('SERVER', 10) +
                pad('PORT(FE/BE)', 18) +
                pad('DATABASE (DB/USER)', 24) +
                pad('STATUS SERVICE', 15)
            );
            console.log('-'.repeat(104));

            apps.forEach(app => {
                let status = 'INACTIVE';
                try {
                    const out = execSync(\`systemctl --user -M \${app.name}@ is-active \${app.name}.service 2>/dev/null || true\`).toString().trim();
                    if (out === 'active') {
                        status = 'ACTIVE';
                    }
                } catch (e) {}

                let ports = '-';
                if (app.stack === 'laravel') {
                    ports = app.port_fe !== 'N/A' ? app.port_fe : (app.port || '-');
                } else if (app.stack === 'node-fullstack') {
                    ports = \`\${app.port_fe}/\${app.port_be}\`;
                } else if (app.stack === 'node-api') {
                    ports = app.port !== 'N/A' ? app.port : '-';
                } else {
                    ports = app.port !== 'N/A' ? app.port : \`\${app.port_fe}/\${app.port_be}\`;
                }
                const dbInfo = \`\${app.db_name || '-'} / \${app.db_user || '-'}\`;
                const statusFormatted = status === 'ACTIVE' ? '\x1b[32mACTIVE\x1b[0m' : '\x1b[31mINACTIVE\x1b[0m';

                console.log(
                    pad(app.name, 16) +
                    pad(app.stack || '-', 15) +
                    pad(app.webserver || '-', 10) +
                    pad(ports, 18) +
                    pad(dbInfo, 24) +
                    statusFormatted
                );
            });
            console.log('========================================================================================================\n');
        "
    else
        cat "$REGISTRY_FILE"
    fi
}

# ------------------------------------------------------------------------------
# Command: LOGS [nama]
# ------------------------------------------------------------------------------
cmd_logs() {
    check_root
    local app_name="$1"

    if [[ -z "$app_name" ]]; then
        log_error "Parameter nama aplikasi harus diisi."
        echo -e "Penggunaan: sudo $0 logs <nama-aplikasi>"
        exit 1
    fi

    if ! id "$app_name" &>/dev/null; then
        log_error "User/Aplikasi '$app_name' tidak ditemukan."
        exit 1
    fi

    log_info "Menampilkan 100 baris log terakhir untuk service '${app_name}.service'..."
    echo -e "${BOLD}-----------------------------------------------------------------${NC}"
    if ! journalctl --user -M "${app_name}@" -u "${app_name}.service" -n 100 --no-pager 2>/dev/null; then
        local user_uid
        user_uid=$(id -u "$app_name")
        journalctl -u "user@${user_uid}.service" -n 100 --no-pager || log_warn "Tidak dapat membaca log journalctl."
    fi
}

# ------------------------------------------------------------------------------
# Command: MANAGE [nama] [opsi]
# ------------------------------------------------------------------------------
cmd_manage() {
    check_root
    local app_name="$1"
    local action="$2"

    if [[ -z "$app_name" || -z "$action" ]]; then
        log_error "Parameter nama aplikasi dan opsi harus diisi."
        echo -e "Penggunaan: sudo $0 manage <nama-aplikasi> <restart|stop|start|status>"
        exit 1
    fi

    if ! id "$app_name" &>/dev/null; then
        log_error "User/Aplikasi '$app_name' tidak ditemukan."
        exit 1
    fi

    case "$action" in
        restart)
            log_info "Merestart service '${app_name}.service'..."
            if run_systemctl_user "$app_name" "restart" "${app_name}.service"; then
                log_success "Service '${app_name}.service' berhasil di-restart."
            else
                log_error "Gagal merestart service '${app_name}.service'."
            fi
            ;;
        stop)
            log_info "Menghentikan service '${app_name}.service'..."
            if run_systemctl_user "$app_name" "stop" "${app_name}.service"; then
                log_success "Service '${app_name}.service' berhasil dihentikan."
            else
                log_error "Gagal menghentikan service '${app_name}.service'."
            fi
            ;;
        start)
            log_info "Menjalankan service '${app_name}.service'..."
            if run_systemctl_user "$app_name" "start" "${app_name}.service"; then
                log_success "Service '${app_name}.service' berhasil dijalankan."
            else
                log_error "Gagal menjalankan service '${app_name}.service'."
            fi
            ;;
        status)
            log_info "Status service '${app_name}.service':"
            run_systemctl_user "$app_name" "status" "${app_name}.service" || true
            ;;
        *)
            log_error "Opsi tidak valid: '$action'."
            echo -e "Opsi yang tersedia: ${YELLOW}restart${NC}, ${YELLOW}stop${NC}, ${YELLOW}start${NC}, ${YELLOW}status${NC}"
            exit 1
            ;;
    esac
}

# ------------------------------------------------------------------------------
# Entrypoint Router
# ------------------------------------------------------------------------------
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        help|--help|-h)
            cmd_help
            ;;
        setup)
            cmd_setup "$@"
            ;;
        create)
            cmd_create "$@"
            ;;
        delete)
            cmd_delete "$@"
            ;;
        list)
            cmd_list "$@"
            ;;
        logs)
            cmd_logs "$@"
            ;;
        manage)
            cmd_manage "$@"
            ;;
        *)
            log_error "Command '$command' tidak dikenal."
            cmd_help
            exit 1
            ;;
    esac
}

main "$@"
