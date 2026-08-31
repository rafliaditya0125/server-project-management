package domain_test

import (
	"testing"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
)

func TestAppEntities(t *testing.T) {
	app := domain.App{
		Name:      "testapp",
		User:      "testapp",
		Home:      "/home/apps/testapp",
		Stack:     domain.StackLaravel,
		StackName: "Laravel (PHP-FPM)",
		WebServer: string(domain.WebServerCaddy),
		PortFE:    "8000",
		PhpMode:   domain.PhpModeSocket,
		DBName:    "testapp_db",
		DBUser:    "testapp_u",
		CreatedAt: domain.NowUTCFormatted(),
	}

	if app.Name != "testapp" {
		t.Errorf("expected app name testapp, got %s", app.Name)
	}

	if app.CreatedAt == "" {
		t.Errorf("expected CreatedAt to be non-empty")
	}
}
