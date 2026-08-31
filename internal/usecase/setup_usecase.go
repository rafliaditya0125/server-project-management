package usecase

import (
	"fmt"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
)

type SetupUsecase struct {
	configRepo        domain.ConfigRepository
	osDetector        domain.OSDetector
	pkgInstaller      domain.PackageManager
	completionManager domain.ShellCompletionManager
	symlinkManager    domain.SymlinkManager
	systemManager     domain.SystemManager
}

func NewSetupUsecase(
	configRepo domain.ConfigRepository,
	osDetector domain.OSDetector,
	pkgInstaller domain.PackageManager,
	completionManager domain.ShellCompletionManager,
	symlinkManager domain.SymlinkManager,
	systemManager domain.SystemManager,
) *SetupUsecase {
	return &SetupUsecase{
		configRepo:        configRepo,
		osDetector:        osDetector,
		pkgInstaller:      pkgInstaller,
		completionManager: completionManager,
		symlinkManager:    symlinkManager,
		systemManager:     systemManager,
	}
}

func (u *SetupUsecase) GetStatus() (*domain.SystemConfig, string, error) {
	osDistro, err := u.osDetector.DetectOS()
	if err != nil {
		osDistro = "unknown"
	}
	cfg, err := u.configRepo.Get()
	return cfg, osDistro, err
}

func (u *SetupUsecase) ConfigureFastCGI(osDistro string, mode domain.PhpMode, sockPath, port string) (*domain.SystemConfig, error) {
	if mode == "" {
		mode = domain.PhpModeSocket
	}
	if sockPath == "" {
		sockPath = u.pkgInstaller.DetectPhpFpmSocket()
	}
	if port == "" {
		port = "9000"
	}

	phpSvc := u.pkgInstaller.DetectPhpFpmService()
	logger.Info("Mengaktifkan dan menjalankan service PHP-FPM '%s'...", phpSvc)
	if err := u.pkgInstaller.EnableAndStartPhpService(phpSvc); err != nil {
		logger.Warn("Gagal mengaktifkan service '%s'. Pastikan PHP-FPM sudah terpasang.", phpSvc)
	} else {
		logger.Success("Service '%s' berhasil diaktifkan dan dijalankan!", phpSvc)
	}

	cfg := &domain.SystemConfig{
		PhpMode:     mode,
		PhpSockPath: sockPath,
		PhpPort:     port,
		PhpService:  phpSvc,
		OS:          osDistro,
		UpdatedAt:   domain.NowUTCFormatted(),
	}

	if err := u.configRepo.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save system config: %w", err)
	}

	return cfg, nil
}

func (u *SetupUsecase) Execute(opts *domain.SetupOptions) error {
	if !u.systemManager.IsRoot() {
		return domain.ErrPermissionDenied
	}

	osDistro, _ := u.osDetector.DetectOS()

	runSymlink := opts.Symlink
	runCompletion := opts.Completion
	runPHP := opts.PHP
	runNode := opts.Node
	runWeb := opts.Web
	runDB := opts.DB
	runFastCGI := opts.FastCGI

	// If no specific flag set or All is true, enable all stages
	if opts.All || (!runSymlink && !runCompletion && !runPHP && !runNode && !runWeb && !runDB && !runFastCGI) {
		runSymlink = true
		runCompletion = true
		runPHP = true
		runNode = true
		runWeb = true
		runDB = true
		runFastCGI = true
	}

	// Process except list
	for _, exc := range opts.Except {
		exc = strings.TrimSpace(strings.ToLower(exc))
		switch exc {
		case "php", "composer":
			runPHP = false
		case "node", "nodejs", "npm":
			runNode = false
		case "web", "webserver", "webservers", "caddy", "nginx":
			runWeb = false
		case "db", "shell", "shell-db", "fish", "mariadb", "mysql":
			runDB = false
		case "fastcgi", "fpm", "php-fpm":
			runFastCGI = false
		case "symlink", "bin", "link":
			runSymlink = false
		case "completion", "completions", "autocomplete", "autocompletion":
			runCompletion = false
		default:
			if exc != "" {
				logger.Warn("Tahap pengecualian '%s' pada --except tidak dikenal. Diabaikan.", exc)
			}
		}
	}

	if !runSymlink && !runCompletion && !runPHP && !runNode && !runWeb && !runDB && !runFastCGI {
		logger.Warn("Tidak ada tahap setup yang dijalankan karena semua tahap dikecualikan.")
		return nil
	}

	logger.Raw("\n%s%s=================================================================%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
	logger.Raw("%s%s       SETUP DEPENDENSI & FASTCGI PROJECT MANAGER MULTI-OS      %s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
	logger.Raw("%s%s=================================================================%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
	logger.Raw("Distro / OS Terdeteksi : %s%s%s\n\n", logger.ColorBold, osDistro, logger.ColorReset)

	if runSymlink {
		_ = u.symlinkManager.CreateGlobalSymlink("")
	}

	if runCompletion {
		_ = u.completionManager.InstallShellCompletions()
	}

	if runPHP {
		if err := u.pkgInstaller.InstallPHPAndComposer(osDistro); err != nil {
			logger.Error("Gagal menginstal PHP & Composer: %v", err)
		}
	}

	if runNode {
		if err := u.pkgInstaller.InstallNodeAndNPM(osDistro); err != nil {
			logger.Error("Gagal menginstal Node.js & NPM: %v", err)
		}
	}

	if runWeb {
		if err := u.pkgInstaller.InstallWebServers(osDistro); err != nil {
			logger.Error("Gagal menginstal Web Servers: %v", err)
		}
	}

	if runDB {
		if err := u.pkgInstaller.InstallShellAndDBClient(osDistro); err != nil {
			logger.Error("Gagal menginstal Shell & DB Client: %v", err)
		}
	}

	if runFastCGI {
		mode := opts.ChosenPhpMode
		if mode == "" {
			mode = domain.PhpModeSocket
		}
		_, _ = u.ConfigureFastCGI(osDistro, mode, opts.ChosenPhpSockPath, opts.ChosenPhpPort)
	}

	logger.Raw("\n%s%s=================================================================%s\n", logger.ColorBold, logger.ColorGreen, logger.ColorReset)
	logger.Raw("%s%s                   SETUP SELESAI DILAKUKAN                       %s\n", logger.ColorBold, logger.ColorGreen, logger.ColorReset)
	logger.Raw("%s%s=================================================================%s\n", logger.ColorBold, logger.ColorGreen, logger.ColorReset)
	logger.Raw("Anda sekarang dapat menjalankan: %ssudo project create%s untuk membuat aplikasi baru.\n\n", logger.ColorBold, logger.ColorReset)

	return nil
}
