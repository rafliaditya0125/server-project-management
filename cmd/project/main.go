package main

import (
	"os"

	"github.com/rafliaditya0125/server-project-management/internal/config"
	"github.com/rafliaditya0125/server-project-management/internal/delivery/cli"
	"github.com/rafliaditya0125/server-project-management/internal/platform/completion"
	"github.com/rafliaditya0125/server-project-management/internal/platform/database"
	"github.com/rafliaditya0125/server-project-management/internal/platform/git"
	"github.com/rafliaditya0125/server-project-management/internal/platform/installer"
	"github.com/rafliaditya0125/server-project-management/internal/platform/symlink"
	"github.com/rafliaditya0125/server-project-management/internal/platform/system"
	"github.com/rafliaditya0125/server-project-management/internal/platform/webserver"
	"github.com/rafliaditya0125/server-project-management/internal/repository"
	"github.com/rafliaditya0125/server-project-management/internal/usecase"
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

	// 3. Usecases
	appUsecase := usecase.NewAppUsecase(appRepo, configRepo, systemManager, dbManager, webGen, gitManager, cfg.AppsBaseDir)
	serviceUsecase := usecase.NewServiceUsecase(systemManager, appRepo)

	// CLI Layer setup with completion manager wiring
	cliApp := cli.NewCLI(appUsecase, nil, serviceUsecase, appRepo, configRepo, cfg.HTTPPort)
	completionManager := completion.NewShellCompletionManager(cliApp.GetRootCmd())

	setupUsecase := usecase.NewSetupUsecase(configRepo, osDetector, pkgInstaller, completionManager, symlinkManager, systemManager)

	// Re-instantiate CLI with wired SetupUsecase
	cliApp = cli.NewCLI(appUsecase, setupUsecase, serviceUsecase, appRepo, configRepo, cfg.HTTPPort)

	if err := cliApp.Execute(); err != nil {
		os.Exit(1)
	}
}
