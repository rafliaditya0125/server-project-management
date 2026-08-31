package usecase

import (
	"fmt"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
)

type ServiceUsecase struct {
	systemManager domain.SystemManager
	appRepo       domain.AppRepository
}

func NewServiceUsecase(systemManager domain.SystemManager, appRepo domain.AppRepository) *ServiceUsecase {
	return &ServiceUsecase{
		systemManager: systemManager,
		appRepo:       appRepo,
	}
}

func (u *ServiceUsecase) Manage(appName string, action domain.ServiceAction) (string, error) {
	if !u.systemManager.UserExists(appName) {
		return "", domain.ErrAppNotFound
	}

	serviceName := fmt.Sprintf("%s.service", appName)

	switch action {
	case domain.ActionRestart:
		logger.Info("Merestart service '%s'...", serviceName)
		out, err := u.systemManager.RunSystemctlUser(appName, "restart", serviceName)
		if err != nil {
			logger.Error("Gagal merestart service '%s': %v", serviceName, err)
			return out, err
		}
		logger.Success("Service '%s' berhasil di-restart.", serviceName)
		return out, nil

	case domain.ActionStop:
		logger.Info("Menghentikan service '%s'...", serviceName)
		out, err := u.systemManager.RunSystemctlUser(appName, "stop", serviceName)
		if err != nil {
			logger.Error("Gagal menghentikan service '%s': %v", serviceName, err)
			return out, err
		}
		logger.Success("Service '%s' berhasil dihentikan.", serviceName)
		return out, nil

	case domain.ActionStart:
		logger.Info("Menjalankan service '%s'...", serviceName)
		out, err := u.systemManager.RunSystemctlUser(appName, "start", serviceName)
		if err != nil {
			logger.Error("Gagal menjalankan service '%s': %v", serviceName, err)
			return out, err
		}
		logger.Success("Service '%s' berhasil dijalankan.", serviceName)
		return out, nil

	case domain.ActionStatus:
		logger.Info("Status service '%s':", serviceName)
		out, _ := u.systemManager.RunSystemctlUser(appName, "status", serviceName)
		return out, nil

	default:
		return "", domain.ErrInvalidServiceAction
	}
}
