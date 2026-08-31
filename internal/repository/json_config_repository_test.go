package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/internal/repository"
)

func TestJSONConfigRepository(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "config.json")
	repo := repository.NewJSONConfigRepository(filePath)

	cfg := &domain.SystemConfig{
		PhpMode:     domain.PhpModeSocket,
		PhpSockPath: "/run/php/php8.3-fpm.sock",
		PhpPort:     "9000",
		PhpService:  "php8.3-fpm",
		OS:          "debian",
		UpdatedAt:   domain.NowUTCFormatted(),
	}

	if err := repo.Save(cfg); err != nil {
		t.Fatalf("Save config failed: %v", err)
	}

	loaded, err := repo.Get()
	if err != nil {
		t.Fatalf("Get config failed: %v", err)
	}
	if loaded.PhpSockPath != "/run/php/php8.3-fpm.sock" {
		t.Errorf("expected sock path /run/php/php8.3-fpm.sock, got %s", loaded.PhpSockPath)
	}

	val, err := repo.GetValue("php_port", "default")
	if err != nil || val != "9000" {
		t.Errorf("expected php_port 9000, got %s (err: %v)", val, err)
	}

	if err := repo.SaveValue("custom_key", "custom_value"); err != nil {
		t.Fatalf("SaveValue failed: %v", err)
	}

	customVal, err := repo.GetValue("custom_key", "")
	if err != nil || customVal != "custom_value" {
		t.Errorf("expected custom_value, got %s", customVal)
	}
}
