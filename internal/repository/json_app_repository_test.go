package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/internal/repository"
)

func TestJSONAppRepository(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repo_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "apps.json")
	repo := repository.NewJSONAppRepository(filePath)

	// 1. Initial list should be empty
	apps, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("expected 0 apps, got %d", len(apps))
	}

	// 2. Save an app
	app := &domain.App{
		Name:      "myapp",
		User:      "myapp",
		Home:      "/home/apps/myapp",
		Stack:     domain.StackLaravel,
		StackName: "Laravel (PHP-FPM)",
		WebServer: "caddy",
		PortFE:    "8000",
		DBName:    "myapp",
		DBUser:    "myapp",
		CreatedAt: domain.NowUTCFormatted(),
	}

	if err := repo.Save(app); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 3. Exists
	exists, err := repo.Exists("myapp")
	if err != nil || !exists {
		t.Errorf("expected app to exist, got %v (err: %v)", exists, err)
	}

	// 4. FindByName
	found, err := repo.FindByName("myapp")
	if err != nil {
		t.Fatalf("FindByName failed: %v", err)
	}
	if found.Name != "myapp" {
		t.Errorf("expected name myapp, got %s", found.Name)
	}

	// 5. Delete
	if err := repo.Delete("myapp"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	existsAfter, err := repo.Exists("myapp")
	if err != nil || existsAfter {
		t.Errorf("expected app to not exist after delete, got %v", existsAfter)
	}
}
