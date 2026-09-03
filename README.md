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
│   │   ├── app.go
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
│   │   ├── webserver/              # Caddy, Nginx, Node.js generator
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
