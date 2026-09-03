# Workflow Administrator: Pembuatan & Pengelolaan Project Baru

Panduan komprehensif bagi Administrator Server / DevOps dalam menginisialisasi environment server, membuat aplikasi terisolasi per-user, dan mengelola siklus hidup aplikasi menggunakan `project`.

---

## 📑 Daftar Isi
1. [Prasyarat Server](#1-prasyarat-server)
2. [Setup Awal Dependensi Server](#2-setup-awal-dependensi-server)
3. [Alur Pembuatan Project (`project create`)](#3-alur-pembuatan-project-project-create)
4. [Penjelasan 4 Pilihan Stack](#4-penjelasan-4-pilihan-stack)
5. [Template Handoff Kredensial ke Developer](#5-template-handoff-kredensial-ke-developer)
6. [Monitoring & Manajemen Aplikasi](#6-monitoring--manajemen-aplikasi)
7. [Penghapusan Aplikasi (`project delete`)](#7-penghapusan-aplikasi-project-delete)
8. [Troubleshooting Umum untuk Admin](#8-troubleshooting-umum-untuk-admin)

---

## 1. Prasyarat Server

Sebelum mulai, pastikan server Linux Anda memenuhi persyaratan berikut:
- **Hak Akses**: Memiliki akses user `root` atau `sudo`.
- **Distribusi Linux yang Didukung**:
  - Arch Linux / Manjaro / EndeavourOS (`pacman`)
  - Ubuntu / Debian (`apt`)
  - Fedora / RHEL / AlmaLinux / Rocky (`dnf`)
- **Systemd**: Menggunakan systemd dengan fitur `systemd-logind` aktif.
- **Database Server**: Server MySQL atau MariaDB sudah terpasang dan berjalan:
  ```bash
  sudo systemctl enable --now mariadb # atau mysql
  ```

---

## 2. Setup Awal Dependensi Server

Jika server baru selesai di-install, jalankan perintah setup otomatis untuk memastikan semua runtime bahasa, web server, shell, dan tools tersedia:

```bash
# Menjalankan seluruh tahapan setup secara otomatis
sudo project setup
```

Perintah di atas akan menginstal dan mengonfigurasi:
1. **PHP & Composer** (termasuk ekstensi PDO MySQL, cURL, mbstring, XML, GD, zip)
2. **Node.js & NPM**
3. **Web Server**: Caddy dan Nginx
4. **Database Client**: MariaDB/MySQL client
5. **Shell**: Fish Shell
6. **FastCGI PHP-FPM**: Deteksi path socket unix (`/run/php/...`) atau port TCP
7. **Symlink Global**: Memasang binary ke `/usr/local/bin/project`
8. **Shell Completion**: Autocompletion untuk Bash, Zsh, dan Fish

> **Tips Opsional**: Anda dapat mengecualikan komponen tertentu, misalnya jika sudah punya PHP:
> ```bash
> sudo project setup --except=php
> ```

---

## 3. Alur Pembuatan Project (`project create`)

Jalankan wizard interaktif pembuatan aplikasi dengan privilege root:

```bash
sudo project create
```

Admin akan dipandu melalui tahapan interaktif berikut:

```text
=== Wizard Pembuatan Aplikasi Terisolasi ===

1. Masukkan nama aplikasi (username) : myapp
2. Masukkan password user             : ********
   Konfirmasi password                : ********
3. Pilih Stack Aplikasi               : (1/2/3/4)
4. Link repositori git (Opsional)     : https://github.com/org/repo.git
5. Konfigurasi Port & Web Server      : Port, FastCGI, Web Server pilihan
6. Konfigurasi Database               : Nama DB, User DB, Password DB, Kredensial Root DB
```

### Yang Dikerjakan Sistem Secara Otomatis:
1. **Validasi Input**: Memastikan nama hanya alfanumerik huruf kecil, port valid (1-65535), dan user belum ada.
2. **Database Provisioning**: Membuat database `<db_name>` dan user MySQL `<db_user>` beserta privilege penuh via root credential.
3. **User Isolation**:
   - Membuat user Linux `<appName>` dengan home direktori di `/home/apps/<appName>`.
   - Mengatur shell default ke `fish` (atau `bash`).
   - Mengaktifkan systemd linger (`loginctl enable-linger <appName>`) agar service tetap berjalan meski user logout.
   - Menginjeksi auto-export `$XDG_RUNTIME_DIR` di shell profile user.
4. **Git Clone (Jika ada)**: Melakukan clone repositori langsung ke home direktori user.
5. **Web Server & Service Generator**:
   - Menghasilkan file konfigurasi web server (`Caddyfile` atau `nginx.conf`).
   - Membuat unit service systemd user di `/home/apps/<appName>/.config/systemd/user/<appName>.service`.
   - Membuat struktur placeholder files jika repo belum siap.
6. **Izin File (Security)**: Mengatur permission direktori home ke `0750` dengan kepemilikan `appName:appName`.
7. **Service Activation**: Menjalankan `daemon-reload`, `enable`, dan `start` pada user service systemd.
8. **Registry Persistence**: Menyimpan metadata aplikasi ke registry `/etc/project/apps.json`.

---

## 4. Penjelasan 4 Pilihan Stack

Saat memilih stack di menu opsi 3, berikut karakteristik dan parameter yang dibutuhkan:

| No | Stack | Deskripsi | Parameter yang Diminta |
|---|---|---|---|
| **1** | **`laravel`** | Laravel monolitik via PHP-FPM FastCGI | Port Web (FE), Mode FastCGI (Socket/Port), Web Server (Caddy/Nginx) |
| **2** | **`node-fullstack`** | Frontend statis Node.js + Backend API Node.js | Port Frontend, Port Backend API, Web Server (Caddy/Nginx) |
| **3** | **`node-api`** | Standalone Node.js murni (tanpa Caddy/Nginx) | Port Aplikasi tunggal |
| **4** | **`node-laravel`** ✨ | Node.js FE (Static) + Laravel BE via FastCGI Reverse Proxy | Port Web Frontend, Mode FastCGI PHP-FPM, Web Server (Caddy/Nginx) |

### Detail Khusus Stack 4: `node-laravel`
Pada stack ini:
- Port publik hanya **1 port** (Port FE).
- Request biasa (`/*`) dilayani langsung oleh web server menuju folder `fe/dist/`.
- Request API (`/api/*`) dialihkan oleh web server langsung ke **PHP-FPM FastCGI** mengarah ke entry point `be/public/index.php`.
- **Tidak membutuhkan `artisan serve`** dan tidak memakan port HTTP tambahan untuk backend.

---

## 5. Template Handoff Kredensial ke Developer

Setelah aplikasi berhasil dibuat, Admin akan melihat ringkasan di terminal. Salin dan kirimkan informasi tersebut ke Developer menggunakan template berikut:

```markdown
Halo [Nama Developer],

Environment aplikasi kamu di server telah berhasil disiapkan. Berikut kredensial dan detail aksesnya:

---
### 📋 Detail Akun & Akses Server
- **Server IP / Host** : 103.xx.xx.xx (atau domain-kamu.com)
- **SSH Username**     : myapp
- **SSH Password**     : [Password yang ditentukan saat create]
- **Home Directory**   : /home/apps/myapp
- **Port Aplikasi**    : 8080 (http://103.xx.xx.xx:8080)
- **Stack**            : Node.js FE + Laravel BE (atau sesuai pilihan)
- **Web Server**       : Caddy / Nginx

### 🗄️ Kredensial Database (MySQL/MariaDB)
- **DB Host**          : 127.0.0.1
- **DB Port**          : 3306
- **DB Name**          : myapp
- **DB User**          : myapp
- **DB Password**      : [Password DB yang ditentukan saat create]

### ⚙️ Service Systemd
- **Service Name**     : myapp.service (User-level)
- **Cek Status**       : systemctl --user status myapp
- **Restart Service**  : systemctl --user restart myapp
- **Lihat Log**        : journalctl --user -u myapp -f

Silakan baca panduan deployment developer di dokumentasi: `docs/workflow-developer.md`.
---
```

---

## 6. Monitoring & Manajemen Aplikasi

Admin dapat memantau dan mengontrol seluruh aplikasi tanpa harus masuk ke masing-masing user:

### 1. Melihat Daftar Semua Aplikasi & Statusnya
```bash
sudo project list
```
Tampilan output:
```text
========================================================================================================
                                       DAFTAR APLIKASI TERISOLASI                                       
========================================================================================================
NAMA APLIKASI   STACK          SERVER    PORT(FE/BE)       DATABASE (DB/USER)      STATUS SERVICE 
--------------------------------------------------------------------------------------------------------

========================================================================================================
```

### 2. Mengontrol Service Aplikasi dari Root
Admin dapat me-restart, mematikan, atau menyalakan service user:
```bash
sudo project manage <nama-app> restart
sudo project manage <nama-app> stop
sudo project manage <nama-app> start
sudo project manage <nama-app> status
```

### 3. Menginspeksi Log Aplikasi
```bash
# Menampilkan 50 baris log terakhir
sudo project logs <nama-app> -n 50
```

---

## 7. Penghapusan Aplikasi (`project delete`)

Jika aplikasi sudah tidak digunakan, admin dapat menghapus seluruh komponennya hingga bersih:

```bash
sudo project delete <nama-app>
```

Proses ini akan mengeksekusi:
1. Menghentikan (`stop`) dan menonaktifkan (`disable`) service systemd user.
2. Mematikan fitur linger user (`loginctl disable-linger <user>`).
3. Mematikan seluruh proses yang masih berjalan milik user (`pkill -u <user>`).
4. Menghapus user Linux beserta seluruh folder home `/home/apps/<nama-app>`.
5. Menghapus database dan user MySQL terkait secara otomatis.
6. Menghapus entri aplikasi dari registry `/etc/project/apps.json`.

---

## 8. Troubleshooting Umum untuk Admin

### Q1: Status aplikasi terbaca `INACTIVE` setelah dibuat
**Penyebab:**
- Binary Web Server (`caddy` atau `nginx`) belum terinstall di server, sehingga service crash saat dieksekusi (`exit code 203/EXEC`).
- PHP-FPM service belum aktif.

**Solusi:**
```bash
# Pastikan web server dan PHP-FPM terpasang
sudo project setup --web
sudo systemctl enable --now php8.3-fpm # sesuaikan versi PHP

# Restart service aplikasi
sudo project manage <nama-app> restart
```

### Q2: Port bentrok (*Address already in use*)
**Penyebab:** Port yang dimasukkan saat `create` sudah dipakai proses lain.

**Solusi:**
```bash
# Cek proses yang mendengarkan port tersebut
sudo ss -tulpn | grep :<port>

# Jika diperlukan, sesuaikan konfigurasi Caddyfile / nginx.conf di /home/apps/<nama-app>/
# Lalu reload service:
sudo project manage <nama-app> restart
```
