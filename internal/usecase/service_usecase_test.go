package usecase_test

import (
	"testing"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/internal/usecase"
)

func TestServiceUsecase(t *testing.T) {
	appRepo := &mockAppRepo{
		apps: []domain.App{
			{Name: "app1", User: "app1"},
		},
	}
	sysMgr := &mockSystemManager{
		root: true,
		users: map[string]bool{
			"app1": true,
		},
		services: map[string]bool{
			"app1.service": false,
		},
	}

	uc := usecase.NewServiceUsecase(sysMgr, appRepo)

	// Start
	out, err := uc.Manage("app1", domain.ActionStart)
	if err != nil {
		t.Fatalf("Manage Start failed: %v", err)
	}
	if out != "ok" {
		t.Errorf("expected out 'ok', got %s", out)
	}
	if !sysMgr.services["app1.service"] {
		t.Errorf("expected service to be active")
	}

	// Stop
	_, err = uc.Manage("app1", domain.ActionStop)
	if err != nil {
		t.Fatalf("Manage Stop failed: %v", err)
	}
	if sysMgr.services["app1.service"] {
		t.Errorf("expected service to be inactive")
	}

	// Non-existent app
	_, err = uc.Manage("nonexistent", domain.ActionStart)
	if err != domain.ErrAppNotFound {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}
}
