# Script Automasi
project.sh merupakan script untuk automasi konsep isolasi user dalam arsitektur app server
## Daftar Opsi Command
script ini ada beberapa opsi command, yaitu:
- help : Menampilkan daftar opsi command
- create : Membuat user, home folder, dan hak akses diambil, menambahkan database dengan nama database yang sama seperti user, menambahkan user sebagai user database tersebut. Membuat service aplikasi yang akan menjalankan sesuai dengan stack
- delete [nama]	: Menghapus user, home folder, database dan user yang diambil dari variabel [nama]
- list : Menampilkan list aplikasi yang ada berserta dengan informasi database dan user yang terkait
- logs [nama] : menampilkan log aplikasi yang diambil dari variabel [nama]
- manage [nama] [opsi] : 
	- restart : merestart service aplikasi
	- stop : menghentikan service aplikasi
	- start : menjalankan service aplikasi	
## Alur Script
1. create : 
    - Meminta input untuk nama aplikasi(yang jadi username), password user, stack (
         Pilihan : 
            1. Laravel (PHP-FPM)
            2. Node.js (Fullstack / Static FE + API BE)
            3. Node.js (Standalone API Only - Direct Node Runtime)
    ), link repositori git (Opsional), meminta port jika pilih stack 1 dan 3
    - Jika pilih stack 1 atau 2
        - Minta input port fe lalu minta port be(Jika pilih stack 2).
        - ada pilihan web server:
            1. Caddy (Ringkas & zero-temp folder)
            2. Nginx (User-space instance)
    - useradd tanpa akses sudo dengan username <nama-aplikasi> yang tadi diinput, password yang tadi diinput, shell fish
    - Membuat folder home dan aplikasi di /home/apps/<nama-aplikasi>
    - Meminta input nama database, username database, dan password database
    - Membuat database, menambahkan user, memberikan hak akses penuh ke database yang baru dibuat ke user yang dibuat tadi(saat membuat database nanti diminta username dan password root database)
    - Jika pilih stack 1 atau 2 : 
        - Jika pilih Caddy:
            - Script men-generate file /home/apps/<nama-aplikasi>/Caddyfile(dengan port fe tadi, disesuaikan sesuai stack yang dipilih, jika stack 2 berarti ada reverse proxy apinya).
            - Menulis unit file ~/.config/systemd/user/<nama-aplikasi>.service:
                ``` TOML
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
        - jika pilih nginx:
            - Script membuat direktori temporary: mkdir -p /home/apps/<nama-aplikasi>/tmp/{client_body,proxy,fastcgi,uwsgi,scgi}.
            - Script men-generate file /home/apps/<nama-aplikasi>/nginx.conf(dengan port fe tadi, disesuaikan sesuai stack yang dipilih, jika stack 2 berarti ada reverse proxy apinya).
            - Menulis unit file ~/.config/systemd/user/<nama-aplikasi>.service:
                ``` TOML
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
                ```
    - Jika Pilih Stack 3:
        - Menulis unit file ~/.config/systemd/user/<nama-aplikasi>.service:
            ``` TOML
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
            Environment=PORT=<Port yang tadi diinput>
            Environment=NODE_ENV=production
            Environment=PATH=/usr/local/bin:/usr/bin:/bin:%h/.nvm/versions/node/current/bin

            # Membaca .env milik user jika tersedia
            EnvironmentFile=-%h/.env

            [Install]
            WantedBy=default.target
            ```
        - Menulis shell script run.sh:
            ``` bash
            #!/bin/bash
            set -e

            # 1. Fallback Server jika dependensi / repo belum siap
            if [ ! -f "package.json" ] && [ ! -f "backend/package.json" ] && [ ! -f "api/package.json" ] && [ ! -f "server/package.json" ]; then
                echo "[INFO] Project files not found. Starting placeholder server on port $PORT..."
                exec node -e "
                    const http = require('http');
                    http.createServer((req, res) => {
                        res.writeHead(200, { 'Content-Type': 'text/html' });
                        res.end('<h1>Application Ready</h1><p>Awaiting deployment in user home directory.</p>');
                    }).listen(process.env.PORT || [Port yang tadi diinput], '0.0.0.0');
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
            ```
    - Jika tadi memasukkan link repository maka otomatis menjalankan `git clone [link-repo]` di /home/apps/<nama-aplikasi>/. Jika tidak, skip proses ini