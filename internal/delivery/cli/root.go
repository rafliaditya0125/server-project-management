package cli

import (
	"fmt"
	"os"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/spf13/cobra"
)

type CLI struct {
	rootCmd        *cobra.Command
	appUsecase     domain.AppUsecase
	setupUsecase   domain.SetupUsecase
	serviceUsecase domain.ServiceUsecase
	appRepo        domain.AppRepository
	configRepo     domain.ConfigRepository
	httpPort       string
}

func NewCLI(
	appUsecase domain.AppUsecase,
	setupUsecase domain.SetupUsecase,
	serviceUsecase domain.ServiceUsecase,
	appRepo domain.AppRepository,
	configRepo domain.ConfigRepository,
	httpPort string,
) *CLI {
	c := &CLI{
		appUsecase:     appUsecase,
		setupUsecase:   setupUsecase,
		serviceUsecase: serviceUsecase,
		appRepo:        appRepo,
		configRepo:     configRepo,
		httpPort:       httpPort,
	}

	rootCmd := &cobra.Command{
		Use:   "project",
		Short: "Script Otomasi Isolasi User & Manajemen Aplikasi Multi-Tenant",
		Long: fmt.Sprintf(`%s%s=================================================================%s
%s%s            PROJECT MANAGER - ISOLASI USER AUTOMATION           %s
%s%s=================================================================%s
PENGGUNAAN:
  sudo project [command] [argumen...]

STACK YANG DIDUKUNG:
  1. Laravel (PHP-FPM: Unix Socket / TCP Port)
  2. Node.js (Fullstack / Static FE + API BE)
  3. Node.js (Standalone API Only - Direct Node Runtime)

WEB SERVER YANG DIDUKUNG (Stack 1 & 2):
  1. Caddy (Ringkas & zero-temp folder)
  2. Nginx (User-space instance)

SISTEM OPERASI YANG DIDUKUNG:
  - Arch Linux / Manjaro / EndeavourOS (pacman)
  - Ubuntu / Debian (apt)
  - Fedora / RHEL / AlmaLinux / Rocky (dnf)`,
			logger.ColorBold, logger.ColorCyan, logger.ColorReset,
			logger.ColorBold, logger.ColorCyan, logger.ColorReset,
			logger.ColorBold, logger.ColorCyan, logger.ColorReset),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	c.rootCmd = rootCmd
	c.registerCommands()
	return c
}

func (c *CLI) GetRootCmd() *cobra.Command {
	return c.rootCmd
}

func (c *CLI) Execute() error {
	return c.rootCmd.Execute()
}

func (c *CLI) registerCommands() {
	c.rootCmd.AddCommand(c.newSetupCmd())
	c.rootCmd.AddCommand(c.newCreateCmd())
	c.rootCmd.AddCommand(c.newDeleteCmd())
	c.rootCmd.AddCommand(c.newListCmd())
	c.rootCmd.AddCommand(c.newLogsCmd())
	c.rootCmd.AddCommand(c.newManageCmd())
	c.rootCmd.AddCommand(c.newCompletionCmd())
	c.rootCmd.AddCommand(c.newServeCmd())
}

// checkRoot ensures current user is root before executing privileged commands
func checkRoot() error {
	if os.Geteuid() != 0 {
		logger.Error("Script ini memerlukan hak akses root. Jalankan dengan: sudo project <command>")
		return domain.ErrPermissionDenied
	}
	return nil
}
