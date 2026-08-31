package webserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
)

type ConfigGenerator struct{}

func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

func (g *ConfigGenerator) GetCaddyPath() string {
	if path, err := exec.LookPath("caddy"); err == nil && path != "" {
		return path
	}
	return "/usr/bin/caddy"
}

func (g *ConfigGenerator) GetNginxPath() string {
	if path, err := exec.LookPath("nginx"); err == nil && path != "" {
		return path
	}
	if _, err := os.Stat("/usr/sbin/nginx"); err == nil {
		return "/usr/sbin/nginx"
	}
	return "/usr/bin/nginx"
}

func (g *ConfigGenerator) findMimeTypesPath() string {
	paths := []string{"/etc/nginx/mime.types", "/usr/share/nginx/mime.types"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/etc/nginx/mime.types"
}

func (g *ConfigGenerator) findFastcgiParamsPath() string {
	paths := []string{"/etc/nginx/fastcgi_params", "/etc/nginx/fastcgi.conf"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/etc/nginx/fastcgi_params"
}

func (g *ConfigGenerator) GenerateLaravelCaddyfile(homeDir, portFE, fastcgiTarget string) error {
	content := fmt.Sprintf(`:%s {
    root * %s/public
    php_fastcgi %s
    file_server
    encode gzip
}
`, portFE, homeDir, fastcgiTarget)

	return os.WriteFile(filepath.Join(homeDir, "Caddyfile"), []byte(content), 0644)
}

func (g *ConfigGenerator) GenerateNodeFullstackCaddyfile(homeDir, portFE, portBE string) error {
	feRoot := filepath.Join(homeDir, "dist")
	if _, err := os.Stat(feRoot); os.IsNotExist(err) {
		if _, errPublic := os.Stat(filepath.Join(homeDir, "public")); errPublic == nil {
			feRoot = filepath.Join(homeDir, "public")
		}
	}

	content := fmt.Sprintf(`:%s {
    root * %s
    file_server
    handle /api/* {
        reverse_proxy 127.0.0.1:%s
    }
    try_files {path} /index.html
}
`, portFE, feRoot, portBE)

	return os.WriteFile(filepath.Join(homeDir, "Caddyfile"), []byte(content), 0644)
}

func (g *ConfigGenerator) GenerateCaddySystemdService(systemdDir, appName string) error {
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return err
	}

	caddyBin := g.GetCaddyPath()
	content := fmt.Sprintf(`[Unit]
Description=Caddy Web Server - %%u
After=network.target

[Service]
Type=simple
WorkingDirectory=%%h
ExecStart=%s run --config %%h/Caddyfile
ExecReload=%s reload --config %%h/Caddyfile
Restart=always

[Install]
WantedBy=default.target
`, caddyBin, caddyBin)

	unitPath := filepath.Join(systemdDir, fmt.Sprintf("%s.service", appName))
	return os.WriteFile(unitPath, []byte(content), 0644)
}

func (g *ConfigGenerator) GenerateLaravelNginxConfig(homeDir, portFE, fastcgiTarget string) error {
	// Create tmp directories
	for _, d := range []string{"client_body", "proxy", "fastcgi", "uwsgi", "scgi"} {
		if err := os.MkdirAll(filepath.Join(homeDir, "tmp", d), 0755); err != nil {
			return err
		}
	}

	mimeTypes := g.findMimeTypesPath()
	fastcgiParams := g.findFastcgiParamsPath()

	content := fmt.Sprintf(`worker_processes 1;
pid %s/tmp/nginx.pid;
error_log %s/tmp/error.log;

events {
    worker_connections 1024;
}

http {
    include %s;
    default_type application/octet-stream;
    access_log %s/tmp/access.log;

    client_body_temp_path %s/tmp/client_body;
    proxy_temp_path %s/tmp/proxy;
    fastcgi_temp_path %s/tmp/fastcgi;
    uwsgi_temp_path %s/tmp/uwsgi;
    scgi_temp_path %s/tmp/scgi;

    server {
        listen %s;
        server_name _;
        root %s/public;
        index index.php index.html;

        location / {
            try_files $uri $uri/ /index.php?$query_string;
        }

        location ~ \.php$ {
            fastcgi_pass %s;
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
            include %s;
        }
    }
}
`, homeDir, homeDir, mimeTypes, homeDir, homeDir, homeDir, homeDir, homeDir, homeDir, portFE, homeDir, fastcgiTarget, fastcgiParams)

	return os.WriteFile(filepath.Join(homeDir, "nginx.conf"), []byte(content), 0644)
}

func (g *ConfigGenerator) GenerateNodeFullstackNginxConfig(homeDir, portFE, portBE string) error {
	// Create tmp directories
	for _, d := range []string{"client_body", "proxy", "fastcgi", "uwsgi", "scgi"} {
		if err := os.MkdirAll(filepath.Join(homeDir, "tmp", d), 0755); err != nil {
			return err
		}
	}

	feRoot := filepath.Join(homeDir, "dist")
	if _, err := os.Stat(feRoot); os.IsNotExist(err) {
		if _, errPublic := os.Stat(filepath.Join(homeDir, "public")); errPublic == nil {
			feRoot = filepath.Join(homeDir, "public")
		}
	}

	mimeTypes := g.findMimeTypesPath()

	content := fmt.Sprintf(`worker_processes 1;
pid %s/tmp/nginx.pid;
error_log %s/tmp/error.log;

events {
    worker_connections 1024;
}

http {
    include %s;
    default_type application/octet-stream;
    access_log %s/tmp/access.log;

    client_body_temp_path %s/tmp/client_body;
    proxy_temp_path %s/tmp/proxy;
    fastcgi_temp_path %s/tmp/fastcgi;
    uwsgi_temp_path %s/tmp/uwsgi;
    scgi_temp_path %s/tmp/scgi;

    server {
        listen %s;
        server_name _;
        root %s;
        index index.html;

        location /api/ {
            proxy_pass http://127.0.0.1:%s/;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_cache_bypass $http_upgrade;
        }

        location / {
            try_files $uri $uri/ /index.html;
        }
    }
}
`, homeDir, homeDir, mimeTypes, homeDir, homeDir, homeDir, homeDir, homeDir, homeDir, portFE, feRoot, portBE)

	return os.WriteFile(filepath.Join(homeDir, "nginx.conf"), []byte(content), 0644)
}

func (g *ConfigGenerator) GenerateNginxSystemdService(systemdDir, appName string) error {
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return err
	}

	nginxBin := g.GetNginxPath()
	content := fmt.Sprintf(`[Unit]
Description=Nginx User Web Server - %%u
After=network.target

[Service]
Type=simple
WorkingDirectory=%%h
ExecStartPre=/usr/bin/mkdir -p %%h/tmp/client_body %%h/tmp/proxy %%h/tmp/fastcgi
ExecStart=%s -p %%h -c %%h/nginx.conf -g "daemon off;"
ExecReload=%s -p %%h -c %%h/nginx.conf -s reload
Restart=always

[Install]
WantedBy=default.target
`, nginxBin, nginxBin)

	unitPath := filepath.Join(systemdDir, fmt.Sprintf("%s.service", appName))
	return os.WriteFile(unitPath, []byte(content), 0644)
}

func (g *ConfigGenerator) GenerateNodeDirectRunScript(homeDir, portSingle string) error {
	content := fmt.Sprintf(`#!/bin/bash
set -e

# 1. Fallback Server jika dependensi / repo belum siap
if [ ! -f "package.json" ] && [ ! -f "backend/package.json" ] && [ ! -f "api/package.json" ] && [ ! -f "server/package.json" ]; then
    echo "[INFO] Project files not found. Starting placeholder server on port $PORT..."
    exec node -e "
        const http = require('http');
        http.createServer((req, res) => {
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end('<h1>Application Ready</h1><p>Awaiting deployment in user home directory.</p>');
        }).listen(process.env.PORT || %s, '0.0.0.0');
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
`, portSingle)

	runScriptPath := filepath.Join(homeDir, "run.sh")
	if err := os.WriteFile(runScriptPath, []byte(content), 0755); err != nil {
		return err
	}
	return nil
}

func (g *ConfigGenerator) GenerateNodeDirectSystemdService(systemdDir, appName, portSingle string) error {
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`[Unit]
Description=NodeJS App (Direct Runtime) - %%u
After=network.target

[Service]
Type=simple
WorkingDirectory=%%h
ExecStart=%%h/run.sh
Restart=always
RestartSec=5s

# Alokasi environment
Environment=PORT=%s
Environment=NODE_ENV=production
Environment=PATH=/usr/local/bin:/usr/bin:/bin:%%h/.nvm/versions/node/current/bin

# Membaca .env milik user jika tersedia
EnvironmentFile=-%%h/.env

[Install]
WantedBy=default.target
`, portSingle)

	unitPath := filepath.Join(systemdDir, fmt.Sprintf("%s.service", appName))
	return os.WriteFile(unitPath, []byte(content), 0644)
}

func (g *ConfigGenerator) CreatePlaceholders(homeDir string, stack domain.StackType, appName string, portBE string) error {
	switch stack {
	case domain.StackLaravel:
		publicDir := filepath.Join(homeDir, "public")
		_ = os.MkdirAll(publicDir, 0755)
		indexPath := filepath.Join(publicDir, "index.php")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			content := fmt.Sprintf("<?php\necho \"<h1>Aplikasi Laravel (%s) Siap</h1><p>Menunggu deployment.</p>\";\n", appName)
			_ = os.WriteFile(indexPath, []byte(content), 0644)
		}
	case domain.StackNodeFullstack:
		distDir := filepath.Join(homeDir, "dist")
		_ = os.MkdirAll(distDir, 0755)
		indexPath := filepath.Join(distDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			content := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s</title></head>
<body>
<h1>Aplikasi Node.js Fullstack (%s) Siap</h1>
<p>Frontend Static + Reverse Proxy API ke port %s.</p>
</body>
</html>
`, appName, appName, portBE)
			_ = os.WriteFile(indexPath, []byte(content), 0644)
		}
	}
	return nil
}
