#!/usr/bin/env bash

# ==============================================================================
# project.sh - Script Otomasi Isolasi User & Manajemen Aplikasi Multi-Tenant
# ==============================================================================

set -eo pipefail

REGISTRY_DIR="/etc/project-manager"
REGISTRY_FILE="${REGISTRY_DIR}/apps.json"
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
    if [ ! -d "$APPS_BASE_DIR" ] && [ "$EUID" -eq 0 ]; then
        mkdir -p "$APPS_BASE_DIR"
        chmod 755 "$APPS_BASE_DIR"
    fi
}

# ------------------------------------------------------------------------------
# Registry Management (JSON using node or fallback)
# ------------------------------------------------------------------------------
save_app_metadata() {
    local json_payload="$1"
    init_registry
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
        // Hapus jika sudah ada nama yang sama
        data = data.filter(a => a.name !== newApp.name);
        data.push(newApp);
        fs.writeFileSync(file, JSON.stringify(data, null, 2));
    "
}

remove_app_metadata() {
    local app_name="$1"
    init_registry
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
}

get_app_metadata() {
    local app_name="$1"
    init_registry
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
    echo -e "  ${GREEN}create${NC}                 : Membuat user terisolasi, direktori home, database,"
    echo -e "                           dan konfigurasi web server/service aplikasi"
    echo -e "  ${GREEN}delete <nama>${NC}          : Menghapus user, folder home, database, dan service"
    echo -e "  ${GREEN}list${NC}                   : Menampilkan daftar aplikasi berserta info database & status"
    echo -e "  ${GREEN}logs <nama>${NC}            : Menampilkan log systemd service aplikasi"
    echo -e "  ${GREEN}manage <nama> <opsi>${NC}   : Mengelola service aplikasi"
    echo -e "                           Pilihan opsi: ${YELLOW}restart${NC}, ${YELLOW}stop${NC}, ${YELLOW}start${NC}, ${YELLOW}status${NC}\n"
    echo -e "${BOLD}STACK YANG DIDUKUNG:${NC}"
    echo -e "  1. Laravel (PHP-FPM)"
    echo -e "  2. Node.js (Fullstack / Static FE + API BE)"
    echo -e "  3. Node.js (Standalone API Only - Direct Node Runtime)\n"
    echo -e "${BOLD}WEB SERVER YANG DIDUKUNG (Stack 1 & 2):${NC}"
    echo -e "  1. Caddy"
    echo -e "  2. Nginx (User-space instance)\n"
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

    if [[ "$stack_choice" == "1" ]]; then
        # Laravel (PHP-FPM) - Hanya meminta 1 Port Web/HTTP
        while true; do
            read -rp "Masukkan Port Web / HTTP Aplikasi (misal: 8080): " port_fe
            if [[ "$port_fe" =~ ^[0-9]+$ ]] && [ "$port_fe" -ge 1 ] && [ "$port_fe" -le 65535 ]; then
                port_single="$port_fe"
                port_be="9000" # Default PHP-FPM local port
                break
            else
                log_error "Port harus berupa angka antara 1 dan 65535."
            fi
        done

        echo -e "\nPilih Web Server:"
        echo -e "  1. Caddy (Ringkas & zero-temp folder)"
        echo -e "  2. Nginx (User-space instance)"
        while true; do
            read -rp "Pilihan Web Server (1/2): " webserver_choice
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
            read -rp "Pilihan Web Server (1/2): " webserver_choice
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
        log_error "Proses pembuatan aplikasi dibatalkan."
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
            log_warn "Petunjuk: Password database untuk user '${db_user}' tidak memenuhi policy keamanan MySQL server (panjang, huruf besar/kecil, angka, simbol)."
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
            # Salin semua file dan hidden files (.env, .git, dll) ke home_dir
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
    if [[ "$stack_choice" == "1" || "$stack_choice" == "2" ]]; then
        if [[ "$webserver_name" == "caddy" ]]; then
            log_info "Membuat konfigurasi Caddyfile..."
            if [[ "$stack_choice" == "1" ]]; then
                # Laravel Caddyfile
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
    php_fastcgi 127.0.0.1:${port_be}
    file_server
    encode gzip
    try_files {path} {path}/ /index.php?{query}
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
            cat <<'EOF' > "${systemd_user_dir}/${app_name}.service"
[Unit]
Description=Caddy Web Server - %u
After=network.target

[Service]
Type=simple
WorkingDirectory=%h
ExecStart=/usr/bin/caddy run --config %h/Caddyfile
ExecReload=/usr/bin/caddy reload --config %h/Caddyfile
Restart=always

[Install]
WantedBy=default.target
EOF

        elif [[ "$webserver_name" == "nginx" ]]; then
            log_info "Membuat direktori temporary Nginx dan file nginx.conf..."
            mkdir -p "${home_dir}/tmp/"{client_body,proxy,fastcgi,uwsgi,scgi}

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
    include /etc/nginx/mime.types;
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
            fastcgi_pass 127.0.0.1:${port_be};
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
            include /etc/nginx/fastcgi_params;
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
    include /etc/nginx/mime.types;
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
            cat <<'EOF' > "${systemd_user_dir}/${app_name}.service"
[Unit]
Description=Nginx User Web Server - %u
After=network.target

[Service]
Type=simple
WorkingDirectory=%h
ExecStartPre=/usr/bin/mkdir -p %h/tmp/client_body %h/tmp/proxy %h/tmp/fastcgi
ExecStart=/usr/sbin/nginx -p %h -c %h/nginx.conf -g "daemon off;"
ExecReload=/usr/sbin/nginx -p %h -c %h/nginx.conf -s reload
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
    metadata_json=$(node -e "
        console.log(JSON.stringify({
            name: '${app_name}',
            user: '${app_name}',
            home: '${home_dir}',
            stack: '${stack_type}',
            stack_name: '${stack_name}',
            webserver: '${webserver_name}',
            port_fe: '${port_fe:-N/A}',
            port_be: '${port_be:-N/A}',
            port: '${port_single:-N/A}',
            db_name: '${db_name}',
            db_user: '${db_user}',
            git_repo: '${git_repo:-None}',
            created_at: new Date().toISOString()
        }));
    ")
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
        db_name=$(node -e "try { const d = ${meta}; console.log(d.db_name || ''); } catch(e){}")
        db_user=$(node -e "try { const d = ${meta}; console.log(d.db_user || ''); } catch(e){}")
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
            pad('PORT(FE/BE)', 16) +
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
                pad(ports, 16) +
                pad(dbInfo, 24) +
                statusFormatted
            );
        });
        console.log('========================================================================================================\n');
    "
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
