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

	// 1d. Caddy Node+Python (Reverse Proxy /api/*)
	nodePythonTmpDir, _ := os.MkdirTemp("", "caddy_node_python_test_*")
	defer os.RemoveAll(nodePythonTmpDir)
	if err := gen.GenerateNodePythonCaddyfile(nodePythonTmpDir, "8080", "8001"); err != nil {
		t.Fatalf("GenerateNodePythonCaddyfile failed: %v", err)
	}
	caddyNPyData, _ := os.ReadFile(filepath.Join(nodePythonTmpDir, "Caddyfile"))
	caddyNPyStr := string(caddyNPyData)
	if !strings.Contains(caddyNPyStr, ":8080") || !strings.Contains(caddyNPyStr, "reverse_proxy 127.0.0.1:8001") || !strings.Contains(caddyNPyStr, "handle /api/*") {
		t.Errorf("unexpected Node-Python Caddyfile: %s", caddyNPyStr)
	}

	// 2c. Nginx Node+Python (Reverse Proxy /api)
	nginxPyTmpDir, _ := os.MkdirTemp("", "nginx_node_python_test_*")
	defer os.RemoveAll(nginxPyTmpDir)
	if err := gen.GenerateNodePythonNginxConfig(nginxPyTmpDir, "8080", "8001"); err != nil {
		t.Fatalf("GenerateNodePythonNginxConfig failed: %v", err)
	}
	nginxPyData, _ := os.ReadFile(filepath.Join(nginxPyTmpDir, "nginx.conf"))
	nginxPyStr := string(nginxPyData)
	if !strings.Contains(nginxPyStr, "listen 8080;") || !strings.Contains(nginxPyStr, "proxy_pass http://127.0.0.1:8001;") {
		t.Errorf("unexpected Node-Python nginx.conf: %s", nginxPyStr)
	}

	// 1e. Caddy Node Frontend Static Only
	nodeFeTmpDir, _ := os.MkdirTemp("", "caddy_node_fe_test_*")
	defer os.RemoveAll(nodeFeTmpDir)
	if err := gen.GenerateNodeFrontendCaddyfile(nodeFeTmpDir, "8080"); err != nil {
		t.Fatalf("GenerateNodeFrontendCaddyfile failed: %v", err)
	}
	caddyFeData, _ := os.ReadFile(filepath.Join(nodeFeTmpDir, "Caddyfile"))
	caddyFeStr := string(caddyFeData)
	if !strings.Contains(caddyFeStr, ":8080") || !strings.Contains(caddyFeStr, "file_server") || !strings.Contains(caddyFeStr, "try_files {path} /index.html") {
		t.Errorf("unexpected Node-Frontend Caddyfile: %s", caddyFeStr)
	}

	// 2d. Nginx Node Frontend Static Only
	nginxFeTmpDir, _ := os.MkdirTemp("", "nginx_node_fe_test_*")
	defer os.RemoveAll(nginxFeTmpDir)
	if err := gen.GenerateNodeFrontendNginxConfig(nginxFeTmpDir, "8080"); err != nil {
		t.Fatalf("GenerateNodeFrontendNginxConfig failed: %v", err)
	}
	nginxFeData, _ := os.ReadFile(filepath.Join(nginxFeTmpDir, "nginx.conf"))
	nginxFeStr := string(nginxFeData)
	if !strings.Contains(nginxFeStr, "listen 8080;") || !strings.Contains(nginxFeStr, "try_files $uri $uri/ /index.html;") {
		t.Errorf("unexpected Node-Frontend nginx.conf: %s", nginxFeStr)
	}

	// 3b. Python Direct run.sh
	pyTmpDir, _ := os.MkdirTemp("", "python_direct_test_*")
	defer os.RemoveAll(pyTmpDir)
	if err := gen.GeneratePythonDirectRunScript(pyTmpDir, "8001"); err != nil {
		t.Fatalf("GeneratePythonDirectRunScript failed: %v", err)
	}
	pyRunData, _ := os.ReadFile(filepath.Join(pyTmpDir, "run.sh"))
	if !strings.Contains(string(pyRunData), "8001") || !strings.Contains(string(pyRunData), ".venv/bin/activate") {
		t.Errorf("unexpected Python run.sh content: %s", string(pyRunData))
	}

	// 3c. Python Direct systemd service
	if err := gen.GeneratePythonDirectSystemdService(pyTmpDir, "testpy", "8001"); err != nil {
		t.Fatalf("GeneratePythonDirectSystemdService failed: %v", err)
	}
	pySvcData, _ := os.ReadFile(filepath.Join(pyTmpDir, "testpy.service"))
	if !strings.Contains(string(pySvcData), "Environment=PORT=8001") || !strings.Contains(string(pySvcData), "PYTHONUNBUFFERED=1") {
		t.Errorf("unexpected Python systemd service content: %s", string(pySvcData))
	}

	// 4. Placeholders
	if err := gen.CreatePlaceholders(tmpDir, domain.StackLaravel, "testapp", ""); err != nil {
		t.Fatalf("CreatePlaceholders failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "public", "index.php")); err != nil {
		t.Errorf("index.php placeholder missing: %v", err)
	}

	// 4b. Python Placeholders
	if err := gen.CreatePlaceholders(pyTmpDir, domain.StackPythonAPI, "testpy", "8001"); err != nil {
		t.Fatalf("CreatePlaceholders Python failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pyTmpDir, "run.sh")); err != nil {
		t.Errorf("run.sh placeholder missing: %v", err)
	}

	if err := gen.CreatePlaceholders(nodePythonTmpDir, domain.StackNodePython, "testnodepy", "8001"); err != nil {
		t.Fatalf("CreatePlaceholders NodePython failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nodePythonTmpDir, "fe", "dist", "index.html")); err != nil {
		t.Errorf("Node-Python fe index.html placeholder missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nodePythonTmpDir, "be", "run.sh")); err != nil {
		t.Errorf("Node-Python be run.sh placeholder missing: %v", err)
	}

	if err := gen.CreatePlaceholders(nodeFeTmpDir, domain.StackNodeFrontend, "testnodefe", ""); err != nil {
		t.Fatalf("CreatePlaceholders NodeFrontend failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nodeFeTmpDir, "dist", "index.html")); err != nil {
		t.Errorf("Node-Frontend dist index.html placeholder missing: %v", err)
	}
}
