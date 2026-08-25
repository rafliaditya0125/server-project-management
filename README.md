# Script Automasi Project Management pada Server
`project.sh` merupakan script otomasi untuk manajemen aplikasi multi-tenant dengan konsep isolasi user penuh (per-user systemd, per-user web server instance/reverse proxy, dan isolasi izin file/database). Script ini mendukung **Multi-OS** (Arch Linux, Ubuntu/Debian, Fedora/RHEL).

---

## Daftar Opsi Command

Script ini menyediakan beberapa opsi command:

- **`help`** : Menampilkan panduan dan daftar opsi command.
- **`setup [opsi...]`** : Melakukan setup dependensi server (PHP, Composer, Node.js, NPM, Caddy, Nginx, MariaDB Client, Fish Shell, Symlink global `/usr/local/bin/project`) serta konfigurasi mode FastCGI PHP-FPM (Unix Socket / TCP Port).
  - Tanpa opsi: Menjalankan semua tahap secara otomatis dan membuat symlink `project` global.
  - Opsi inklusi: `--php`, `--node`, `--web`, `--db`, `--fastcgi`, `--symlink`, `--all`.
  - Opsi pengecualian: `-e` atau `--except=<tahap1,tahap2>` (contoh: `--except=db` atau `-e php,fastcgi`).
- **`create`** : Membuat user sistem terisolasi, home folder `/home/apps/<nama-aplikasi>`, database & user MySQL, serta service systemd user sesuai stack dan pilihan web server (Caddy / Nginx).
- **`delete <nama>`** : Menghapus user sistem, direktori home aplikasi, database & user MySQL, dan service systemd user.
- **`list`** : Menampilkan daftar aplikasi yang terdaftar beserta status service, web server, port, dan database terkait.
- **`logs <nama>`** : Menampilkan log journalctl service systemd user aplikasi.
- **`manage <nama> <opsi>`** : Mengelola service aplikasi:
  - `restart` : Merestart service aplikasi
  - `stop` : Menghentikan service aplikasi
  - `start` : Menjalankan service aplikasi
  - `status` : Menampilkan status detail service aplikasi

> [!TIP]
> Setelah menjalankan `setup`, script otomatis membuat symlink `/usr/local/bin/project`. Anda dapat menjalankan perintah langsung dengan `sudo project <command>` dari direktori mana pun.

---

## Sistem Operasi yang Didukung

- **Arch Linux / Manjaro / EndeavourOS** (via `pacman`)
- **Ubuntu / Debian** (via `apt`)
- **Fedora / RHEL / AlmaLinux / Rocky** (via `dnf`)

---

## Alur Kerja Script

### 1. `setup` (Setup Dependensi Server & FastCGI Multi-OS)
Command ini digunakan saat inisialisasi server:
1. Mendeteksi distro Linux secara otomatis dari `/etc/os-release`.
2. **Dukungan Opsi Tahapan**:
   - **Tanpa opsi (`sudo ./project.sh setup`)**: Menjalankan semua tahap setup secara otomatis (Setup Lengkap termasuk pembuatan symlink `/usr/local/bin/project`).
   - **Dengan opsi pengecualian (`--except` / `-e`)**: Menjalankan semua tahap KECUALI yang ditentukan, misalnya:
     - `sudo ./project.sh setup --except=db` : Menjalankan seluruh tahap KECUALI database/shell client.
     - `sudo ./project.sh setup -e php,fastcgi` : Menjalankan seluruh tahap KECUALI PHP dan FastCGI.
   - **Dengan opsi inklusi tertentu**: Hanya menjalankan tahap yang dipilih, misalnya:
     - `sudo ./project.sh setup --php` : Menginstal PHP, ekstensi umum, dan Composer.
     - `sudo ./project.sh setup --node` : Menginstal Node.js dan NPM.
     - `sudo ./project.sh setup --web` : Menginstal Web Server (Caddy dan Nginx).
     - `sudo ./project.sh setup --db` : Menginstal Fish Shell dan MariaDB/MySQL Client.
     - `sudo ./project.sh setup --fastcgi` : Konfigurasi FastCGI PHP-FPM (Unix Socket vs TCP Port).
     - `sudo ./project.sh setup --symlink` : Hanya membuat symlink `/usr/local/bin/project`.
     - `sudo ./project.sh setup --interactive` (atau `-i`) : Menjalankan menu interaktif pemilihan tahap.
     - **Kombinasi Opsi**: Dapat menjalankan beberapa tahap sekaligus, contoh: `sudo ./project.sh setup --php --node` atau `sudo ./project.sh setup php web`.

---

### 2. `create` (Pembuatan Aplikasi Terisolasi)
1. **Input Data**:
   - Nama aplikasi (sebagai username sistem terisolasi).
   - Password user sistem.
   - Pilihan Stack:
     1. **Laravel (PHP-FPM)**
     2. **Node.js (Fullstack / Static FE + API BE)**
     3. **Node.js (Standalone API Only - Direct Node Runtime)**
   - Link repositori Git (Opsional).

2. **Konfigurasi Port & Web Server**:
   - **Jika Stack 1 (Laravel)**:
     - Meminta Port Web/HTTP (misal: `8000`).
     - Meminta mode koneksi FastCGI PHP-FPM (Unix Socket atau TCP Port, otomatis mengambil nilai default dari hasil `setup`).
     - Memilih Web Server: **Caddy** atau **Nginx**.
   - **Jika Stack 2 (Node.js Fullstack)**:
     - Meminta Port Frontend (misal: `8080`) dan Port Backend API (misal: `3001`).
     - Memilih Web Server: **Caddy** atau **Nginx**.
   - **Jika Stack 3 (Node.js Direct API)**:
     - Meminta 1 Port Aplikasi (misal: `3000`).

3. **Database**:
   - Meminta nama database, username database, dan password database.
   - Memverifikasi koneksi root DB, lalu membuat database dan user MySQL serta memberikan full privileges.

4. **User & Lingkungan**:
   - Membuat user tanpa akses sudo dengan home direktori `/home/apps/<nama-aplikasi>`.
   - Mengaktifkan systemd user lingering (`loginctl enable-linger <nama>`).
   - Melakukan `git clone` jika link repositori disertakan.

5. **Konfigurasi Web Server & Service**:
   - **Caddy (Laravel)**:
     File `/home/apps/<nama-aplikasi>/Caddyfile`:
     ```caddyfile
     :8000 {
         root * /home/apps/<nama-aplikasi>/public
         php_fastcgi unix//run/php/php8.3-fpm.sock
         file_server
         encode gzip
     }
     ```
     *(Jika mode TCP Port, target berupa `127.0.0.1:9000`)*.
     Unit file systemd user: `~/.config/systemd/user/<nama-aplikasi>.service`:
     ```ini
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
     ```

   - **Nginx (Laravel)**:
     File `/home/apps/<nama-aplikasi>/nginx.conf`:
     ```nginx
     worker_processes 1;
     pid /home/apps/<nama-aplikasi>/tmp/nginx.pid;
     error_log /home/apps/<nama-aplikasi>/tmp/error.log;

     events {
         worker_connections 1024;
     }

     http {
         include /etc/nginx/mime.types;
         default_type application/octet-stream;
         access_log /home/apps/<nama-aplikasi>/tmp/access.log;

         client_body_temp_path /home/apps/<nama-aplikasi>/tmp/client_body;
         proxy_temp_path /home/apps/<nama-aplikasi>/tmp/proxy;
         fastcgi_temp_path /home/apps/<nama-aplikasi>/tmp/fastcgi;
         uwsgi_temp_path /home/apps/<nama-aplikasi>/tmp/uwsgi;
         scgi_temp_path /home/apps/<nama-aplikasi>/tmp/scgi;

         server {
             listen 8000;
             server_name _;
             root /home/apps/<nama-aplikasi>/public;
             index index.php index.html;

             location / {
                 try_files $uri $uri/ /index.php?$query_string;
             }

             location ~ \.php$ {
                 fastcgi_pass unix:/run/php/php8.3-fpm.sock;
                 fastcgi_index index.php;
                 fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
                 include /etc/nginx/fastcgi_params;
             }
         }
     }
     ```
     *(Jika mode TCP Port, fastcgi_pass berupa `127.0.0.1:9000;`)*.

   - **Node.js Direct Runtime (Stack 3)**:
     Unit file systemd user: `~/.config/systemd/user/<nama-aplikasi>.service` dan script `~/run.sh`.

6. **Aktivasi Service & Registry**:
   - Mengatur perizinan direktori (`chown -R <nama>:<nama>`, `chmod 750`).
   - Menjalankan `systemctl --user enable --now <nama-aplikasi>.service`.
   - Mencatat metadata aplikasi ke `/etc/project-manager/apps.json`.
