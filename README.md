# Server Project Management (Go Rewrite)

Aplikasi otomasi manajemen multi-tenant terisolasi pada server Linux (per-user systemd, per-user web server instance / reverse proxy, dan isolasi izin file / database MySQL).

Sistem ini telah ditulis ulang (*rewritten*) ke dalam bahasa **Go** dengan mengadopsi **Clean / Layered Architecture**, didukung oleh **Cobra** untuk antarmuka CLI & dynamic shell autocompletion (Bash, Zsh, Fish), serta endpoint **HTTP REST API** untuk integrasi eksternal.

Mendukung **Multi-OS**:
- **Arch Linux / Manjaro / EndeavourOS** (via `pacman`)
- **Ubuntu / Debian** (via `apt`)
- **Fedora / RHEL / AlmaLinux / Rocky** (via `dnf`)

---

## 🚀 Kompilasi & Instalasi

### 1. Build Binary
```bash
# Kompilasi CLI (bin/project) dan API (bin/project-api)
make build

# Atau hanya CLI
make build-cli
```

### 2. Install ke Sistem
```bash
make install
```
Binary akan dipasang ke `/usr/local/bin/project` dan dapat dipanggil langsung: `sudo project <command>`.

---

## 📌 Daftar Perintah CLI

- **`help`** : Menampilkan panduan dan daftar opsi command.
- **`setup [flags]`** : Setup dependensi server (PHP, Composer, Node.js, NPM, Caddy, Nginx, MariaDB Client, Fish Shell, Symlink global `/usr/local/bin/project`, Shell Autocompletion) serta konfigurasi mode FastCGI PHP-FPM (Unix Socket / TCP Port).
  - Tanpa opsi: Menjalankan semua tahap secara otomatis.
  - Flag tahap: `--php`, `--node`, `--web`, `--db`, `--fastcgi`, `--symlink`, `--completion`, `--all`.
  - Pengecualian: `-e` atau `--except=<tahap1,tahap2>` (contoh: `--except=db` atau `-e php,fastcgi`).
  - Interaktif: `--interactive` atau `-i`.
- **`create`** : Membuat user sistem terisolasi, home folder `/home/apps/<nama-aplikasi>`, database & user MySQL, serta service systemd user sesuai stack dan pilihan web server (Caddy / Nginx).
- **`delete <nama>`** : Menghapus user sistem, direktori home aplikasi, database & user MySQL, dan service systemd user.
- **`list`** : Menampilkan daftar aplikasi yang terdaftar beserta status service (ACTIVE/INACTIVE), web server, port, dan database terkait.
- **`logs <nama>`** : Menampilkan log journalctl service systemd user aplikasi (`-n` untuk jumlah baris).
- **`manage <nama> <aksi>`** : Mengelola service aplikasi (`restart`, `stop`, `start`, `status`).
- **`completion <shell>`** : Menghasilkan script shell autocompletion (`bash`, `zsh`, `fish`, `powershell`).
- **`serve [--port=:8080]`** : Menjalankan REST HTTP API Server.

---

## 🏗️ Stack Aplikasi yang Didukung

Saat membuat aplikasi baru (`project create`), tersedia 4 pilihan stack:

### 1. `laravel` — Laravel (PHP-FPM)
Laravel monolitik yang dilayani via **FastCGI** (PHP-FPM) menggunakan Caddy atau Nginx.

```
[portFE: Caddy/Nginx]
    └── /* ──► PHP-FPM FastCGI ──► public/index.php (Laravel)
```

**Direktori aplikasi:**
```
/home/apps/<nama>/
└── public/
    └── index.php   ← Laravel entry point
```

---

### 2. `node-fullstack` — Node.js Fullstack (Static FE + API BE)
Frontend statis Node.js (React/Vue/SPA) + Backend Node.js API, keduanya di-serve oleh satu web server melalui **reverse proxy HTTP**.

```
[portFE: Caddy/Nginx]
    ├── /*      ──► Static files (dist/)
    └── /api/*  ──► reverse_proxy 127.0.0.1:portBE (Node.js API)
```

**Direktori aplikasi:**
```
/home/apps/<nama>/
├── dist/       ← Build output FE (React/Vue/dll)
└── ...         ← Node.js API (portBE)
```

---

### 3. `node-api` — Node.js Standalone API
Node.js berjalan langsung sebagai server (tanpa web server wrapper). Cocok untuk microservice atau backend API murni.

```
[portSingle: Node.js langsung via run.sh]
```

**Direktori aplikasi:**
```
/home/apps/<nama>/
├── run.sh           ← Script entrypoint (auto-generated)
└── package.json     ← Atau backend/, api/, server/
```

---

### 4. `node-laravel` — Node.js FE + Laravel BE (FastCGI Reverse Proxy)
Frontend Node.js (React/Vue/SPA) + Backend **Laravel** dilayani oleh **satu web server** menggunakan teknik **reverse proxy FastCGI** — *tanpa `artisan serve`, tanpa port tambahan untuk Laravel*. Laravel diakses langsung via PHP-FPM.

```
[portFE: Caddy/Nginx]
    ├── /*      ──► Static files (fe/dist/)          ← Node.js FE
    └── /api/*  ──► PHP-FPM FastCGI (socket/TCP)    ← Laravel BE
                        └── be/public/index.php
```

**Konfigurasi Caddy (`Caddyfile`):**
```caddyfile
{
    admin off
}

:8080 {
    handle /api/* {
        uri strip_prefix /api
        root * /home/apps/<nama>/be/public
        php_fastcgi unix//run/php/php8.3-fpm.sock
        file_server
    }

    handle {
        root * /home/apps/<nama>/fe/dist
        file_server
        try_files {path} /index.html
    }
}
```

**Konfigurasi Nginx (`nginx.conf`):**
```nginx
server {
    listen 8080;

    location /api/ {
        alias /home/apps/<nama>/be/public/;
        try_files $uri /api/index.php?$query_string;

        location ~ \.php$ {
            fastcgi_pass unix:/run/php/php8.3-fpm.sock;
            fastcgi_param SCRIPT_FILENAME /home/apps/<nama>/be/public/index.php;
            fastcgi_param REQUEST_URI $request_uri;
            include /etc/nginx/fastcgi_params;
        }
    }

    location / {
        root /home/apps/<nama>/fe/dist;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
}
```

**Direktori aplikasi:**
```
/home/apps/<nama>/
├── fe/              ← Repo / hasil build frontend (Node.js)
│   └── dist/
│       └── index.html
└── be/              ← Repo Laravel backend
    └── public/
        └── index.php  ← Laravel entry point
```

> **Catatan penting**: PHP-FPM harus sudah berjalan di server (`sudo systemctl start php8.x-fpm`). Jalankan `sudo project setup --php` jika belum terkonfigurasi.

---

## ⚡ Autocompletion Pintar (Cobra Dynamic Completion)

Auto-completion didukung penuh untuk **Bash, Zsh, dan Fish**:
- Ketik `sudo project ` lalu tekan `<TAB>` -> Menampilkan daftar sub-command.
- Ketik `sudo project delete ` lalu tekan `<TAB>` -> Otomatis melengkapi daftar nama aplikasi dari sistem.
- Ketik `sudo project manage ` lalu tekan `<TAB>` -> Otomatis melengkapi nama aplikasi aktif.
- Ketik `sudo project manage myapp ` lalu tekan `<TAB>` -> Otomatis melengkapi aksi: `restart`, `stop`, `start`, `status`.
- Ketik `sudo project setup --except=` lalu tekan `<TAB>` -> Otomatis melengkapi nama tahapan setup.

---

## 🌐 HTTP REST API Endpoints

Ketika dijalankan via `sudo project serve` atau `bin/project-api`:

| Method | Endpoint | Deskripsi |
| --- | --- | --- |
| `GET` | `/health` | Health check endpoint |
| `GET` | `/api/v1/apps` | Mendapatkan daftar semua aplikasi dan status service |
| `POST` | `/api/v1/apps` | Membuat aplikasi baru terisolasi |
| `GET` | `/api/v1/apps/:name` | Mendapatkan detail metadata aplikasi |
| `DELETE` | `/api/v1/apps/:name` | Menghapus aplikasi, user, folder, DB & service |
| `GET` | `/api/v1/apps/:name/logs` | Mengambil log journalctl service |
| `POST` | `/api/v1/apps/:name/manage` | Mengontrol service (`restart`, `stop`, `start`, `status`) |
| `GET` | `/api/v1/setup/status` | Mendapatkan status dependensi & konfigurasi server |
| `POST` | `/api/v1/setup` | Menjalankan tahapan setup server |
| `POST` | `/api/v1/setup/fastcgi` | Mengonfigurasi koneksi FastCGI PHP-FPM |

**Contoh payload `POST /api/v1/apps` untuk stack `node-laravel`:**
```json
{
  "name": "myapp",
  "user_password": "secret",
  "stack": "node-laravel",
  "webserver": "caddy",
  "port_fe": "8080",
  "php_fpm_mode": "socket",
  "php_sock_path": "/run/php/php8.3-fpm.sock",
  "db_name": "myapp",
  "db_user": "myapp",
  "db_password": "dbsecret",
  "db_root_user": "root",
  "db_root_password": "rootsecret"
}
```

---

## 🏗️ Struktur Arsitektur (Clean Architecture)

```text
.
├── cmd/
│   ├── api/
│   │   └── main.go                 # Entrypoint HTTP REST API Server
│   └── project/
│       └── main.go                 # Entrypoint CLI Tool (/usr/local/bin/project)
├── internal/
│   ├── config/
│   │   └── config.go               # Registry paths & environment variables
│   ├── domain/                     # Core Business Entities & Interfaces
│   │   ├── app.go                  # StackType: laravel, node-fullstack, node-api, node-laravel
│   │   ├── config.go
│   │   ├── errors.go
│   │   ├── platform.go
│   │   ├── repository.go
│   │   └── usecase.go
│   ├── repository/                 # Data Persistence Layer
│   │   ├── json_app_repository.go
│   │   └── json_config_repository.go
│   ├── platform/                   # Linux OS, DB, Webserver & Installer Adapters
│   │   ├── system/                 # User management & systemd user services
│   │   ├── database/               # MySQL / MariaDB operations
│   │   ├── webserver/              # Caddy, Nginx, Node.js generator (incl. node-laravel)
│   │   ├── installer/              # Multi-OS package installer & FastCGI
│   │   ├── completion/             # Shell completion installer
│   │   ├── symlink/                # Global symlink manager
│   │   └── git/                    # Git repository cloner
│   ├── usecase/                    # Business Logic Layer
│   │   ├── app_usecase.go
│   │   ├── setup_usecase.go
│   │   └── service_usecase.go
│   └── delivery/                   # Transport Layer
│       ├── cli/                    # Cobra CLI Subcommands & Prompts
│       └── http/                   # Chi REST Handlers, Middlewares & Router
├── pkg/                            # Reusable Shared Utilities
│   ├── logger/
│   ├── response/
│   └── terminal/
├── Makefile
├── go.mod
└── README.md
```
