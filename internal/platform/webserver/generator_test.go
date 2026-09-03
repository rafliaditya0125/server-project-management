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

	// 4. Placeholders
	if err := gen.CreatePlaceholders(tmpDir, domain.StackLaravel, "testapp", ""); err != nil {
		t.Fatalf("CreatePlaceholders failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "public", "index.php")); err != nil {
		t.Errorf("index.php placeholder missing: %v", err)
	}
}
