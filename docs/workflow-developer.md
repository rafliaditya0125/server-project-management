# Workflow Developer: Panduan Deployment & Operasional Aplikasi

Panduan lengkap bagi Developer setelah menerima kredensial dari Administrator Server, mencakup langkah login, penyiapan kode sumber (*source code*), konfigurasi database, pengelolaan service systemd, hingga aplikasi live dan dapat diakses publik.

---

## 📑 Daftar Isi
1. [Memahami Kredensial yang Diberikan](#1-memahami-kredensial-yang-diberikan)
2. [Akses Masuk ke Server (SSH / Terminal)](#2-akses-masuk-ke-server-ssh--terminal)
3. [Alur Deployment Berdasarkan Stack](#3-alur-deployment-berdasarkan-stack)
   - [Stack 1: Laravel (PHP-FPM)](#stack-1-laravel-php-fpm)
   - [Stack 2: Node.js Fullstack (Static FE + API BE)](#stack-2-nodejs-fullstack-static-fe--api-be)
   - [Stack 3: Node.js Standalone API](#stack-3-nodejs-standalone-api)
   - [Stack 4: Node.js FE + Laravel BE (FastCGI Reverse Proxy)](#stack-4-nodejs-fe--laravel-be-fastcgi-reverse-proxy)
4. [Kustomisasi Caddyfile & nginx.conf (Fleksibilitas Struktur Folder)](#4-kustomisasi-caddyfile--nginxconf-fleksibilitas-struktur-folder)
5. [Manajemen Service Systemd (User-Level)](#5-manajemen-service-systemd-user-level)
6. [Monitoring & Membaca Log Aplikasi](#6-monitoring--membaca-log-aplikasi)
7. [Verifikasi Akses Aplikasi](#7-verifikasi-akses-aplikasi)
8. [Troubleshooting & Solusi Masalah Umum](#8-troubleshooting--solusi-masalah-umum)

---

## 1. Memahami Kredensial yang Diberikan

Administrator server akan menyerahkan informasi akses yang mirip dengan format berikut:

- **Host / IP Server**: `103.xx.xx.xx`
- **Username SSH**: `myapp` *(nama aplikasi Anda)*
- **Password SSH**: `********`
- **Home Directory**: `/home/apps/myapp`
- **Port Aplikasi**: `8080` *(akses via `http://103.xx.xx.xx:8080`)*
- **Kredensial Database**:
  - Host: `127.0.0.1`
  - Port: `3306`
  - DB Name: `myapp`
  - DB User: `myapp`
  - DB Password: `********`
- **Nama Service**: `myapp.service`

> 💡 **Prinsip Lingkungan Terisolasi**:
> Anda memiliki kontrol **penuh 100%** atas direktori home Anda sendiri dan service systemd Anda. Anda **tidak memerlukan hak akses `sudo / root`** untuk menjalankan composer, npm, build frontend, migrasi database, me-restart web server, ataupun melihat log.

---

## 2. Akses Masuk ke Server (SSH / Terminal)

### A. Login Melalui SSH (Dari Komputer Lokal Developer)
Buka terminal lokal Anda dan lakukan koneksi:

```bash
ssh myapp@103.xx.xx.xx
```
Masukkan password yang diberikan. Begitu berhasil login, Anda akan berada di direktori `/home/apps/myapp`.

### B. Login via `su` (Jika Menggunakan Server Lokal / Shared Terminal)
Jika Anda login dari user lain pada server yang sama:
```bash
# Selalu gunakan tanda hubung '-' agar memuat login environment secara utuh
su - myapp
```

---

## 3. Alur Deployment Berdasarkan Stack

Pilihlah panduan deployment di bawah ini sesuai stack yang dikonfigurasikan untuk aplikasi Anda:

---

### Stack 1: Laravel (PHP-FPM)

Pada stack ini, web server (Caddy / Nginx) langsung melayani direktori `public/index.php` melalui FastCGI PHP-FPM.

#### 📁 Struktur Direktori yang Diharapkan:
```text
/home/apps/myapp/
├── app/
├── bootstrap/
├── config/
├── database/
├── public/
│   └── index.php       <-- Entry point utama
├── routes/
├── storage/
├── .env
├── composer.json
└── Caddyfile (atau nginx.conf)
```

#### 🚀 Langkah-langkah Deployment:
1. **Clone atau Upload Kode Anda**:
   ```bash
   cd /home/apps/myapp
   # Jika folder masih berisi placeholder bawaan, hapus atau ganti dengan repo Anda:
   git clone <repo-url> .
   ```

2. **Pasang Dependensi PHP**:
   ```bash
   composer install --no-dev --optimize-autoloader
   ```

3. **Konfigurasi Environment (`.env`)**:
   Salin dan sesuaikan konfigurasi database:
   ```bash
   cp .env.example .env
   php artisan key:generate
   ```
   Buka `.env` (misal dengan `nano .env`) dan sesuaikan blok database:
   ```env
   DB_CONNECTION=mysql
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_DATABASE=myapp
   DB_USERNAME=myapp
   DB_PASSWORD=password_database_anda
   ```

4. **Migrasi Database & Storage Link**:
   ```bash
   php artisan migrate --force
   php artisan storage:link
   ```

5. **Atur Izin Folder Storage & Cache**:
   ```bash
   chmod -R 775 storage bootstrap/cache
   ```

6. **Restart Web Server**:
   ```bash
   systemctl --user restart myapp
   ```

---

### Stack 2: Node.js Fullstack (Static FE + API BE)

Pada stack ini, web server melayani file static frontend di `dist/` dan meneruskan rute `/api/*` ke backend Node.js Anda yang berjalan di port internal (`portBE`).

#### 📁 Struktur Direktori yang Diharapkan:
```text
/home/apps/myapp/
├── dist/               <-- Hasil build frontend (index.html, assets/)
├── backend/            <-- Kode backend Node.js API
│   ├── package.json
│   ├── index.js
│   └── .env
└── Caddyfile (atau nginx.conf)
```

#### 🚀 Langkah-langkah Deployment:
1. **Build Frontend**:
   ```bash
   cd /home/apps/myapp/frontend # atau folder sumber FE Anda
   npm install
   npm run build
   # Pastikan file hasil build berada di /home/apps/myapp/dist/
   ```

2. **Jalankan Backend API**:
   Pastikan backend mendengarkan pada port internal (`portBE`) yang ditentukan admin:
   ```bash
   cd /home/apps/myapp/backend
   npm install
   # Sesuaikan port di .env backend: PORT=3001
   ```
   Anda dapat menjalankan backend menggunakan process manager milik user seperti PM2:
   ```bash
   pm2 start index.js --name "myapp-api"
   pm2 save
   ```

3. **Restart Web Server Frontend**:
   ```bash
   systemctl --user restart myapp
   ```

---

### Stack 3: Node.js Standalone API

Pada stack ini, aplikasi Anda dijalankan langsung oleh unit systemd user melalui script runner `run.sh` bawaan tanpa web server perantara.

#### 📁 Struktur Direktori yang Diharapkan:
```text
/home/apps/myapp/
├── package.json
├── dist/
│   └── main.js (atau index.js)
├── .env
└── run.sh              <-- Dibuat otomatis oleh sistem
```

#### 🚀 Langkah-langkah Deployment:
1. **Install Dependensi & Build**:
   ```bash
   cd /home/apps/myapp
   npm install
   npm run build
   ```

2. **Konfigurasi Environment**:
   File `run.sh` secara otomatis membaca variabel dari `.env` dan menyuntikkan variabel `PORT` yang telah dialokasikan oleh Admin.
   Pastikan kode server Anda membaca `process.env.PORT`:
   ```javascript
   const PORT = process.env.PORT || 3000;
   app.listen(PORT, '0.0.0.0', () => {
       console.log(`Server running on port ${PORT}`);
   });
   ```

3. **Restart Service Aplikasi**:
   ```bash
   systemctl --user restart myapp
   ```

---

### Stack 4: Node.js FE + Laravel BE (FastCGI Reverse Proxy)

Stack ini menggabungkan frontend modern berbasis Node.js (Vite, React, Vue, Svelte) dan backend **Laravel API** dalam satu kesatuan web server yang ringkas dan efisien tanpa `artisan serve`.

```text
Request Pengunjung
        │
        ▼
   [Port 8080: Caddy / Nginx]
        ├── /*       ──► Static SPA (fe/dist/index.html)
        └── /api/*   ──► PHP-FPM FastCGI (be/public/index.php)
```

#### 📁 Struktur Direktori yang Diharapkan:
```text
/home/apps/myapp/
├── fe/                 <-- Proyek Frontend Node.js
│   ├── src/
│   ├── package.json
│   ├── vite.config.js
│   └── dist/           <-- Target build frontend (index.html)
├── be/                 <-- Proyek Backend Laravel
│   ├── app/
│   ├── public/
│   │   └── index.php   <-- Entry point Laravel API
│   ├── .env
│   ├── composer.json
│   └── routes/api.php
└── Caddyfile (atau nginx.conf)
```

#### 🚀 Langkah-langkah Deployment:

#### Tahap 1: Setup Frontend (Node.js)
```bash
cd /home/apps/myapp/fe
npm install
npm run build
```
*Pastikan output build berada di folder `/home/apps/myapp/fe/dist` dan memuat file `index.html`.*

#### Tahap 2: Setup Backend (Laravel)
```bash
cd /home/apps/myapp/be
composer install --no-dev --optimize-autoloader

# Konfigurasi .env
cp .env.example .env
php artisan key:generate
```

Ubah konfigurasi database di `be/.env`:
```env
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=myapp
DB_USERNAME=myapp
DB_PASSWORD=password_db_kamu
```

Jalankan migrasi database dan perbaiki izin direktori:
```bash
php artisan migrate --force
chmod -R 775 storage bootstrap/cache
```

#### Tahap 3: Restart Web Server
```bash
systemctl --user restart myapp
```

> 🎯 **Cara Kerja Rute:**
> - Buka `http://103.xx.xx.xx:8080/` -> Menampilkan UI frontend SPA Anda.
> - Request ke `http://103.xx.xx.xx:8080/api/users` -> Otomatis diproses oleh `routes/api.php` di Laravel.

---

## 4. Kustomisasi Caddyfile & nginx.conf (Fleksibilitas Struktur Folder)

Setiap aplikasi memiliki file konfigurasi web server bawaan di `/home/apps/myapp/` (`Caddyfile` atau `nginx.conf`). File ini **100% milik Anda** (`myapp:myapp`), sehingga Anda bebas mengeditnya langsung jika arsitektur kode atau nama folder project Anda berbeda dari default generator.

### 💡 Kapan Anda Perlu Menyesuaikan Konfigurasi?
- **Folder Build Frontend Berbeda**: Misalnya output build frontend Anda ada di `build/`, `out/`, `client/dist/`, atau `frontend/dist/` (bukan `dist/` atau `fe/dist/`).
- **Folder Backend Berbeda**: Misalnya direktori public Laravel Anda berada di `backend/public/`, `server/public/`, atau langsung di root `public/`.
- **Rute API Khusus**: Jika rute backend Anda menggunakan prefix selain `/api/*` (misal `/v1/*`, `/backend/*`).
- **Kebutuhan Fitur Tambahan**: Mengaktifkan limit upload file besar (`client_max_body_size 50M`), WebSocket proxy, header CORS, gzip/zstd compression, dsb.

---

### Contoh 1: Menyesuaikan `Caddyfile` (Caddy)
Buka file dengan editor terminal Anda: `nano ~/Caddyfile`

```caddyfile
{
    admin off
}

:8080 {
    # 1. Jika direktori public Laravel Anda berada di lokasi lain:
    handle /api/* {
        uri strip_prefix /api
        # Ubah path root ke direktori public Laravel Anda:
        root * /home/apps/myapp/backend/public
        php_fastcgi unix//run/php/php8.3-fpm.sock
        file_server
    }

    # 2. Jika direktori build frontend Anda bukan 'fe/dist':
    handle {
        # Ubah path root ke lokasi index.html Anda:
        root * /home/apps/myapp/frontend/build
        file_server
        try_files {path} /index.html
    }
}
```

---

### Contoh 2: Menyesuaikan `nginx.conf` (Nginx)
Buka file dengan editor terminal Anda: `nano ~/nginx.conf`

```nginx
    # Di dalam blok 'server':
    server {
        listen 8080;
        server_name _;

        # Menambahkan batas upload jika aplikasi menerima file besar:
        client_max_body_size 50M;

        # 1. Menyesuaikan lokasi backend Laravel:
        location /api/ {
            alias /home/apps/myapp/backend/public/;
            try_files $uri /api/index.php?$query_string;

            location ~ \.php$ {
                fastcgi_pass unix:/run/php/php8.3-fpm.sock;
                # Pastikan path ini menunjuk ke file index.php Laravel yang sebenarnya:
                fastcgi_param SCRIPT_FILENAME /home/apps/myapp/backend/public/index.php;
                fastcgi_param REQUEST_URI $request_uri;
                include /etc/nginx/fastcgi_params;
            }
        }

        # 2. Menyesuaikan lokasi frontend:
        location / {
            # Ubah root ke folder hasil build frontend Anda:
            root /home/apps/myapp/frontend/build;
            index index.html;
            try_files $uri $uri/ /index.html;
        }
    }
```

---

### 🔄 Menerapkan Perubahan Web Server
Setiap kali Anda selesai mengubah `Caddyfile` atau `nginx.conf`, restart service user Anda agar perubahan langsung aktif:

```bash
systemctl --user restart myapp
```

Atau jika menggunakan Nginx dan ingin reload tanpa downtime:
```bash
systemctl --user reload myapp
```

Periksa apakah konfigurasi baru berjalan normal:
```bash
systemctl --user status myapp
```

---

## 5. Manajemen Service Systemd (User-Level)

Sebagai developer, Anda mengontrol aplikasi Anda sendiri melalui systemd user:

| Tindakan | Perintah |
|---|---|
| **Melihat Status Service** | `systemctl --user status myapp` |
| **Me-restart Service** | `systemctl --user restart myapp` |
| **Menghentikan Service** | `systemctl --user stop myapp` |
| **Menyalakan Service** | `systemctl --user start myapp` |
| **Reload Konfigurasi Systemd** | `systemctl --user daemon-reload` |

> ⚠️ **PENTING**: Jangan gunakan `sudo` saat menjalankan `systemctl --user`. Service ini berada di user scope Anda sendiri.

---

## 6. Monitoring & Membaca Log Aplikasi

Untuk melihat output pesan, error, atau traffic aplikasi secara *real-time*:

### A. Live Log Systemd (Journalctl)
```bash
# Menampilkan log langsung (follow log)
journalctl --user -u myapp -f

# Menampilkan 100 baris log terakhir
journalctl --user -u myapp -n 100 --no-pager
```

### B. Log Error Laravel (Khusus Stack Laravel)
```bash
tail -f /home/apps/myapp/storage/logs/laravel.log
# Atau pada stack node-laravel:
tail -f /home/apps/myapp/be/storage/logs/laravel.log
```

---

## 7. Verifikasi Akses Aplikasi

1. **Uji Coba Melalui Terminal (Curl)**:
   ```bash
   curl -I http://localhost:8080
   ```
   *Respon harus mengembalikan status `HTTP/1.1 200 OK` (atau `301/302 Redirect`).*

2. **Akses Melalui Browser**:
   Buka browser di komputer Anda dan kunjungi:
   ```text
   http://103.xx.xx.xx:8080
   ```
   *(Ganti `103.xx.xx.xx` dengan IP server dan `8080` dengan port aplikasi Anda).*

---

## 8. Troubleshooting & Solusi Masalah Umum

### Q1: Muncul error `$XDG_RUNTIME_DIR not defined` saat cek status
**Penyebab:** Anda berpindah user menggunakan `su myapp` (tanpa tanda `-`).
**Solusi:**
Ketikkan perintah ini di terminal Anda:
- Di shell **Fish**:
  ```fish
  set -x XDG_RUNTIME_DIR /run/user/(id -u)
  ```
- Di shell **Bash**:
  ```bash
  export XDG_RUNTIME_DIR=/run/user/$(id -u)
  ```
*Lain kali, selalu gunakan `su - myapp` atau langsung login via SSH.*

### Q2: Laravel Error: "The stream or file "...laravel.log" could not be opened: failed to open stream: Permission denied"
**Penyebab:** Web server / PHP-FPM tidak memiliki hak menulis di folder storage.
**Solusi:**
```bash
cd /home/apps/myapp # (atau /home/apps/myapp/be)
chmod -R 775 storage bootstrap/cache
```

### Q3: Frontend menampilkan halaman kosong / 404 pada routing SPA
**Penyebab:** File build belum berada di direktori yang tepat (`dist/` atau `fe/dist/`).
**Solusi:**
Pastikan file `index.html` benar-benar ada di:
- Stack Node Fullstack: `/home/apps/myapp/dist/index.html`
- Stack Node + Laravel: `/home/apps/myapp/fe/dist/index.html`

Lakukan build ulang frontend Anda:
```bash
npm run build
systemctl --user restart myapp
```

### Q4: Database Error: "Access denied for user 'myapp'@'localhost'"
**Penyebab:** Password database di file `.env` tidak sesuai dengan password yang dibuat oleh Administrator.
**Solusi:**
Periksa kembali catatan kredensial database dari admin dan sesuaikan nilai `DB_PASSWORD` pada file `.env`.
