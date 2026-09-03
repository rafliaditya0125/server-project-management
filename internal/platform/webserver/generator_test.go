package webserver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/internal/platform/webserver"
)

func TestConfigGenerator(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "webserver_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := webserver.NewConfigGenerator()

	// 1. Caddy Laravel
	if err := gen.GenerateLaravelCaddyfile(tmpDir, "8000", "unix//run/php/php8.3-fpm.sock"); err != nil {
		t.Fatalf("GenerateLaravelCaddyfile failed: %v", err)
	}
	caddyData, _ := os.ReadFile(filepath.Join(tmpDir, "Caddyfile"))
	if !strings.Contains(string(caddyData), "admin off") || !strings.Contains(string(caddyData), ":8000") || !strings.Contains(string(caddyData), "php_fastcgi unix//run/php/php8.3-fpm.sock") {
		t.Errorf("unexpected Caddyfile content: %s", string(caddyData))
	}

	// 1b. Caddy Node Fullstack
	nodeTmpDir, _ := os.MkdirTemp("", "caddy_node_test_*")
	defer os.RemoveAll(nodeTmpDir)
	if err := gen.GenerateNodeFullstackCaddyfile(nodeTmpDir, "8000", "3000"); err != nil {
		t.Fatalf("GenerateNodeFullstackCaddyfile failed: %v", err)
	}
	caddyNodeData, _ := os.ReadFile(filepath.Join(nodeTmpDir, "Caddyfile"))
	if !strings.Contains(string(caddyNodeData), "admin off") || !strings.Contains(string(caddyNodeData), ":8000") || !strings.Contains(string(caddyNodeData), "reverse_proxy 127.0.0.1:3000") {
		t.Errorf("unexpected Caddyfile content: %s", string(caddyNodeData))
	}

	// 2. Nginx Laravel
	if err := gen.GenerateLaravelNginxConfig(tmpDir, "8000", "unix:/run/php/php8.3-fpm.sock"); err != nil {
		t.Fatalf("GenerateLaravelNginxConfig failed: %v", err)
	}
	nginxData, _ := os.ReadFile(filepath.Join(tmpDir, "nginx.conf"))
	if !strings.Contains(string(nginxData), "listen 8000;") || !strings.Contains(string(nginxData), "fastcgi_pass unix:/run/php/php8.3-fpm.sock;") {
		t.Errorf("unexpected nginx.conf content: %s", string(nginxData))
	}

	// 3. Node Direct run.sh
	if err := gen.GenerateNodeDirectRunScript(tmpDir, "3000"); err != nil {
		t.Fatalf("GenerateNodeDirectRunScript failed: %v", err)
	}
	runScriptData, _ := os.ReadFile(filepath.Join(tmpDir, "run.sh"))
	if !strings.Contains(string(runScriptData), "3000") {
		t.Errorf("unexpected run.sh content: %s", string(runScriptData))
	}

	// 1c. Caddy Node+Laravel (FastCGI Reverse Proxy)
	nodeLaravelTmpDir, _ := os.MkdirTemp("", "caddy_node_laravel_test_*")
	defer os.RemoveAll(nodeLaravelTmpDir)
	if err := gen.GenerateNodeLaravelCaddyfile(nodeLaravelTmpDir, "8080", "unix//run/php/php8.3-fpm.sock"); err != nil {
		t.Fatalf("GenerateNodeLaravelCaddyfile failed: %v", err)
	}
	caddyNLData, _ := os.ReadFile(filepath.Join(nodeLaravelTmpDir, "Caddyfile"))
	caddyNLStr := string(caddyNLData)
	if !strings.Contains(caddyNLStr, "admin off") {
		t.Errorf("Caddyfile missing 'admin off': %s", caddyNLStr)
	}
	if !strings.Contains(caddyNLStr, ":8080") {
		t.Errorf("Caddyfile missing port ':8080': %s", caddyNLStr)
	}
	if !strings.Contains(caddyNLStr, "handle /api/*") {
		t.Errorf("Caddyfile missing 'handle /api/*': %s", caddyNLStr)
	}
	if !strings.Contains(caddyNLStr, "php_fastcgi unix//run/php/php8.3-fpm.sock") {
		t.Errorf("Caddyfile missing php_fastcgi directive: %s", caddyNLStr)
	}
	if !strings.Contains(caddyNLStr, "uri strip_prefix /api") {
		t.Errorf("Caddyfile missing uri strip_prefix: %s", caddyNLStr)
	}
	if !strings.Contains(caddyNLStr, "try_files {path} /index.html") {
		t.Errorf("Caddyfile missing SPA fallback: %s", caddyNLStr)
	}

	// 2b. Nginx Node+Laravel (FastCGI Reverse Proxy)
	nginxNLTmpDir, _ := os.MkdirTemp("", "nginx_node_laravel_test_*")
	defer os.RemoveAll(nginxNLTmpDir)
	if err := gen.GenerateNodeLaravelNginxConfig(nginxNLTmpDir, "8080", "unix:/run/php/php8.3-fpm.sock"); err != nil {
		t.Fatalf("GenerateNodeLaravelNginxConfig failed: %v", err)
	}
	nginxNLData, _ := os.ReadFile(filepath.Join(nginxNLTmpDir, "nginx.conf"))
	nginxNLStr := string(nginxNLData)
	if !strings.Contains(nginxNLStr, "listen 8080;") {
		t.Errorf("nginx.conf missing 'listen 8080;': %s", nginxNLStr)
	}
	if !strings.Contains(nginxNLStr, "location /api/") {
		t.Errorf("nginx.conf missing 'location /api/': %s", nginxNLStr)
	}
	if !strings.Contains(nginxNLStr, "fastcgi_pass unix:/run/php/php8.3-fpm.sock;") {
		t.Errorf("nginx.conf missing fastcgi_pass: %s", nginxNLStr)
	}
	if !strings.Contains(nginxNLStr, "try_files $uri $uri/ /index.html;") {
		t.Errorf("nginx.conf missing SPA fallback: %s", nginxNLStr)
	}

	// 4. Placeholders
	if err := gen.CreatePlaceholders(tmpDir, domain.StackLaravel, "testapp", ""); err != nil {
		t.Fatalf("CreatePlaceholders failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "public", "index.php")); err != nil {
		t.Errorf("index.php placeholder missing: %v", err)
	}
}
