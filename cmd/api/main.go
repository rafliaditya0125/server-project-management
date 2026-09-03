package main

import (
	"log"
	"net/http"

	"github.com/rafliaditya0125/server-project-management/internal/config"
	deliveryHTTP "github.com/rafliaditya0125/server-project-management/internal/delivery/http"
	"github.com/rafliaditya0125/server-project-management/internal/delivery/http/handler"
	"github.com/rafliaditya0125/server-project-management/internal/platform/completion"
	"github.com/rafliaditya0125/server-project-management/internal/platform/database"
	"github.com/rafliaditya0125/server-project-management/internal/platform/git"
	"github.com/rafliaditya0125/server-project-management/internal/platform/installer"
	"github.com/rafliaditya0125/server-project-management/internal/platform/symlink"
	"github.com/rafliaditya0125/server-project-management/internal/platform/system"
	"github.com/rafliaditya0125/server-project-management/internal/platform/webserver"
	"github.com/rafliaditya0125/server-project-management/internal/repository"
	"github.com/rafliaditya0125/server-project-management/internal/usecase"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
)

func main() {
	cfg := config.Load()

	// 1. Repositories
	appRepo := repository.NewJSONAppRepository(cfg.RegistryFile)
	configRepo := repository.NewJSONConfigRepository(cfg.ConfigFile)

	// 2. Platforms
	osDetector := system.NewOSDetector()
	systemManager := system.NewSystemManager()
	dbManager := database.NewMySQLManager()
	webGen := webserver.NewConfigGenerator()
	gitManager := git.NewGitManager()
	fastcgiDetector := installer.NewFastCGIDetector()
	pkgInstaller := installer.NewPackageInstaller(fastcgiDetector)
	symlinkManager := symlink.NewSymlinkManager()
	completionManager := completion.NewShellCompletionManager(nil)

	// 3. Usecases
	appUsecase := usecase.NewAppUsecase(appRepo, configRepo, systemManager, dbManager, webGen, gitManager, cfg.AppsBaseDir)
	setupUsecase := usecase.NewSetupUsecase(configRepo, osDetector, pkgInstaller, completionManager, symlinkManager, systemManager)
	serviceUsecase := usecase.NewServiceUsecase(systemManager, appRepo)

	// 4. Handlers & Delivery
	appHandler := handler.NewAppHandler(appUsecase, serviceUsecase)
	setupHandler := handler.NewSetupHandler(setupUsecase)

	// 5. Server Start
	r := deliveryHTTP.NewRouter(appHandler, setupHandler)

	logger.Success("Starting Project Management API server on port %s", cfg.HTTPPort)
	if err := http.ListenAndServe(cfg.HTTPPort, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
