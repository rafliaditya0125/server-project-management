package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
)

type AppUsecase struct {
	appRepo       domain.AppRepository
	configRepo    domain.ConfigRepository
	systemManager domain.SystemManager
	dbManager     domain.DatabaseManager
	webGen        domain.WebServerConfigGenerator
	gitManager    domain.GitManager
	appsBaseDir   string
}

func NewAppUsecase(
	appRepo domain.AppRepository,
	configRepo domain.ConfigRepository,
	systemManager domain.SystemManager,
	dbManager domain.DatabaseManager,
	webGen domain.WebServerConfigGenerator,
	gitManager domain.GitManager,
	appsBaseDir string,
) *AppUsecase {
	if appsBaseDir == "" {
		appsBaseDir = "/home/apps"
	}
	return &AppUsecase{
		appRepo:       appRepo,
		configRepo:    configRepo,
		systemManager: systemManager,
		dbManager:     dbManager,
		webGen:        webGen,
		gitManager:    gitManager,
		appsBaseDir:   appsBaseDir,
	}
}

func (u *AppUsecase) validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ErrInvalidAppName
	}
	matched, _ := regexp.MatchString(`^[a-z0-9_-]+$`, name)
	if !matched {
		return domain.ErrInvalidAppName
	}
	return nil
}

func (u *AppUsecase) validatePort(portStr string) error {
	p, err := strconv.Atoi(portStr)
	if err != nil || p < 1 || p > 65535 {
		return domain.ErrInvalidPort
	}
	return nil
}

func (u *AppUsecase) cleanupAbortedApp(appName string) {
	logger.Warn("Membersihkan resource yang sempat terbuat untuk '%s'...", appName)
	_, _ = u.systemManager.RunSystemctlUser(appName, "stop", fmt.Sprintf("%s.service", appName))
	_, _ = u.systemManager.RunSystemctlUser(appName, "disable", fmt.Sprintf("%s.service", appName))
	_ = u.systemManager.DisableLinger(appName)
	_ = u.systemManager.KillUserProcesses(appName)
	_ = u.systemManager.DeleteUser(appName, true)
	homeDir := filepath.Join(u.appsBaseDir, appName)
	_ = os.RemoveAll(homeDir)
}

func (u *AppUsecase) Create(dto *domain.CreateAppDTO) (*domain.App, error) {
	if !u.systemManager.IsRoot() {
		return nil, domain.ErrPermissionDenied
	}

	// 1. Validasi Nama
	dto.Name = strings.TrimSpace(dto.Name)
	if err := u.validateName(dto.Name); err != nil {
		return nil, err
	}

	if u.systemManager.UserExists(dto.Name) {
		return nil, domain.ErrUserAlreadyExists
	}

	homeDir := filepath.Join(u.appsBaseDir, dto.Name)
	if _, err := os.Stat(homeDir); err == nil {
		return nil, domain.ErrHomeDirAlreadyExists
	}

	if strings.TrimSpace(dto.UserPassword) == "" {
		return nil, domain.ErrInvalidPassword
	}

	// 2. Validasi Stack & Port
	var stackName string
	var portFE, portBE, portSingle string
	var caddyFastcgiTarget, nginxFastcgiTarget string

	switch dto.Stack {
	case domain.StackLaravel:
		stackName = "Laravel (PHP-FPM)"
		if err := u.validatePort(dto.PortFE); err != nil {
			return nil, err
		}
		portFE = dto.PortFE
		portSingle = dto.PortFE

		if dto.PhpFpmMode == domain.PhpModePort {
			if dto.PhpTcpPort == "" {
				dto.PhpTcpPort = "9000"
			}
			if err := u.validatePort(dto.PhpTcpPort); err != nil {
				return nil, err
			}
			portBE = dto.PhpTcpPort
			caddyFastcgiTarget = fmt.Sprintf("127.0.0.1:%s", dto.PhpTcpPort)
			nginxFastcgiTarget = fmt.Sprintf("127.0.0.1:%s", dto.PhpTcpPort)
		} else {
			dto.PhpFpmMode = domain.PhpModeSocket
			if dto.PhpSockPath == "" {
				cfg, _ := u.configRepo.Get()
				if cfg != nil && cfg.PhpSockPath != "" {
					dto.PhpSockPath = cfg.PhpSockPath
				} else {
					dto.PhpSockPath = "/run/php/php8.3-fpm.sock"
				}
			}
			portBE = fmt.Sprintf("socket (%s)", dto.PhpSockPath)

			cleanSock := strings.TrimPrefix(dto.PhpSockPath, "unix:")
			cleanSock = strings.TrimPrefix(cleanSock, "unix/")
			if strings.HasPrefix(cleanSock, "/") {
				caddyFastcgiTarget = fmt.Sprintf("unix/%s", cleanSock)
			} else {
				caddyFastcgiTarget = fmt.Sprintf("unix//%s", cleanSock)
			}
			nginxFastcgiTarget = fmt.Sprintf("unix:%s", cleanSock)
		}

		if dto.WebServer == "" {
			dto.WebServer = domain.WebServerCaddy
		}

	case domain.StackNodeFullstack:
		stackName = "Node.js (Fullstack / Static FE + API BE)"
		if err := u.validatePort(dto.PortFE); err != nil {
			return nil, err
		}
		if err := u.validatePort(dto.PortBE); err != nil {
			return nil, err
		}
		if dto.PortFE == dto.PortBE {
			return nil, domain.ErrPortConflict
		}
		portFE = dto.PortFE
		portBE = dto.PortBE
		if dto.WebServer == "" {
			dto.WebServer = domain.WebServerCaddy
		}

	case domain.StackNodeAPI:
		stackName = "Node.js (Standalone API Only - Direct Node Runtime)"
		if err := u.validatePort(dto.PortSingle); err != nil {
			return nil, err
		}
		portSingle = dto.PortSingle
		dto.WebServer = domain.WebServerNone

	default:
		return nil, domain.ErrInvalidStack
	}

	// 3. Database Defaults
	if dto.DBName == "" {
		dto.DBName = dto.Name
	}
	if dto.DBUser == "" {
		dto.DBUser = dto.Name
	}
	if dto.DBPassword == "" {
		return nil, fmt.Errorf("database password is required")
	}
	if dto.DBRootUser == "" {
		dto.DBRootUser = "root"
	}

	// 4. Eksekusi Pembuatan Database Terlebih Dahulu
	logger.Info("Memverifikasi koneksi root database...")
	if err := u.dbManager.VerifyRootConnection(dto.DBRootUser, dto.DBRootPassword); err != nil {
		logger.Error("Gagal terhubung ke database sebagai root ('%s'): %v", dto.DBRootUser, err)
		return nil, err
	}

	logger.Info("Membuat database '%s' dan user database '%s'...", dto.DBName, dto.DBUser)
	if err := u.dbManager.CreateDatabaseAndUser(dto.DBRootUser, dto.DBRootPassword, dto.DBName, dto.DBUser, dto.DBPassword); err != nil {
		logger.Error("Gagal membuat database '%s' atau user '%s': %v", dto.DBName, dto.DBUser, err)
		return nil, err
	}
	logger.Success("Database '%s' dan hak akses user '%s' berhasil dibuat!", dto.DBName, dto.DBUser)

	// 5. Membuat User Sistem & Direktori Home
	shellPath := u.systemManager.GetFishOrBashShell()
	logger.Info("Membuat user sistem '%s'...", dto.Name)
	if err := u.systemManager.CreateUser(dto.Name, dto.UserPassword, homeDir, shellPath); err != nil {
		// Rollback DB
		_ = u.dbManager.DropDatabaseAndUser(dto.DBRootUser, dto.DBRootPassword, dto.DBName, dto.DBUser)
		return nil, fmt.Errorf("failed to create system user: %w", err)
	}
	logger.Success("User '%s' berhasil dibuat dengan shell '%s'.", dto.Name, shellPath)

	logger.Info("Mengaktifkan linger untuk user '%s'...", dto.Name)
	if err := u.systemManager.EnableLinger(dto.Name); err != nil {
		logger.Warn("Gagal mengaktifkan linger secara otomatis via loginctl: %v", err)
	}

	systemdUserDir := filepath.Join(homeDir, ".config", "systemd", "user")
	_ = os.MkdirAll(systemdUserDir, 0755)

	// 6. Git Clone jika ada
	if dto.GitRepo != "" {
		logger.Info("Melakukan git clone dari repositori '%s' ke '%s'...", dto.GitRepo, homeDir)
		if err := u.gitManager.Clone(dto.GitRepo, homeDir); err != nil {
			logger.Error("Gagal meng-clone repositori git: %v", err)
			_ = u.dbManager.DropDatabaseAndUser(dto.DBRootUser, dto.DBRootPassword, dto.DBName, dto.DBUser)
			u.cleanupAbortedApp(dto.Name)
			return nil, fmt.Errorf("git clone failed, app creation aborted: %w", err)
		}
		logger.Success("Repositori berhasil di-clone.")
	}

	// 7. Setup Konfigurasi Web Server / Fallback Sesuai Stack
	_ = u.webGen.CreatePlaceholders(homeDir, dto.Stack, dto.Name, portBE)

	if dto.Stack == domain.StackLaravel {
		if dto.WebServer == domain.WebServerCaddy {
			logger.Info("Membuat konfigurasi Caddyfile...")
			_ = u.webGen.GenerateLaravelCaddyfile(homeDir, portFE, caddyFastcgiTarget)
			_ = u.webGen.GenerateCaddySystemdService(systemdUserDir, dto.Name)
		} else if dto.WebServer == domain.WebServerNginx {
			logger.Info("Membuat direktori temporary Nginx dan file nginx.conf...")
			_ = u.webGen.GenerateLaravelNginxConfig(homeDir, portFE, nginxFastcgiTarget)
			_ = u.webGen.GenerateNginxSystemdService(systemdUserDir, dto.Name)
		}
	} else if dto.Stack == domain.StackNodeFullstack {
		if dto.WebServer == domain.WebServerCaddy {
			logger.Info("Membuat konfigurasi Caddyfile...")
			_ = u.webGen.GenerateNodeFullstackCaddyfile(homeDir, portFE, portBE)
			_ = u.webGen.GenerateCaddySystemdService(systemdUserDir, dto.Name)
		} else if dto.WebServer == domain.WebServerNginx {
			logger.Info("Membuat direktori temporary Nginx dan file nginx.conf...")
			_ = u.webGen.GenerateNodeFullstackNginxConfig(homeDir, portFE, portBE)
			_ = u.webGen.GenerateNginxSystemdService(systemdUserDir, dto.Name)
		}
	} else if dto.Stack == domain.StackNodeAPI {
		logger.Info("Membuat service direct Node.js runtime dan run.sh...")
		_ = u.webGen.GenerateNodeDirectRunScript(homeDir, portSingle)
		_ = u.webGen.GenerateNodeDirectSystemdService(systemdUserDir, dto.Name, portSingle)
	}

	// 8. Perbaiki Kepemilikan & Izin File
	logger.Info("Mengatur perizinan file...")
	_ = u.systemManager.SetPermissions(homeDir, dto.Name, 0750)

	// 9. Enable & Start Systemd Service
	serviceName := fmt.Sprintf("%s.service", dto.Name)
	logger.Info("Memulai user service systemd '%s'...", serviceName)
	_, _ = u.systemManager.RunSystemctlUser(dto.Name, "daemon-reload", "")
	_, _ = u.systemManager.RunSystemctlUser(dto.Name, "enable", serviceName)
	if _, err := u.systemManager.RunSystemctlUser(dto.Name, "start", serviceName); err == nil {
		logger.Success("Service '%s' berhasil diaktifkan dan dijalankan!", serviceName)
	} else {
		logger.Warn("Service dibuat, namun belum dapat dimulai otomatis (mungkin memerlukan active user bus / binary web server).")
	}

	// 10. Simpan Metadata ke Registry
	newApp := &domain.App{
		Name:      dto.Name,
		User:      dto.Name,
		Home:      homeDir,
		Stack:     dto.Stack,
		StackName: stackName,
		WebServer: string(dto.WebServer),
		PortFE:    valueOrNA(portFE),
		PortBE:    valueOrNA(portBE),
		Port:      valueOrNA(portSingle),
		PhpMode:   dto.PhpFpmMode,
		DBName:    dto.DBName,
		DBUser:    dto.DBUser,
		GitRepo:   valueOrDefault(dto.GitRepo, "None"),
		CreatedAt: domain.NowUTCFormatted(),
	}

	if err := u.appRepo.Save(newApp); err != nil {
		return nil, fmt.Errorf("failed to save app metadata: %w", err)
	}

	return newApp, nil
}

func (u *AppUsecase) Delete(dto *domain.DeleteAppDTO) error {
	if !u.systemManager.IsRoot() {
		return domain.ErrPermissionDenied
	}

	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return domain.ErrInvalidAppName
	}

	app, _ := u.appRepo.FindByName(dto.Name)

	var dbName, dbUser string
	if app != nil {
		dbName = app.DBName
		dbUser = app.DBUser
	}
	if dbName == "" {
		dbName = dto.Name
	}
	if dbUser == "" {
		dbUser = dto.Name
	}

	serviceName := fmt.Sprintf("%s.service", dto.Name)
	logger.Info("Menghentikan service dan proses user '%s'...", dto.Name)
	_, _ = u.systemManager.RunSystemctlUser(dto.Name, "stop", serviceName)
	_, _ = u.systemManager.RunSystemctlUser(dto.Name, "disable", serviceName)
	_ = u.systemManager.DisableLinger(dto.Name)
	_ = u.systemManager.KillUserProcesses(dto.Name)

	logger.Info("Menghapus user '%s' dan direktori home...", dto.Name)
	_ = u.systemManager.DeleteUser(dto.Name, true)

	homeDir := filepath.Join(u.appsBaseDir, dto.Name)
	if _, err := os.Stat(homeDir); err == nil {
		_ = os.RemoveAll(homeDir)
	}

	if dto.DBRootUser == "" {
		dto.DBRootUser = "root"
	}
	if u.dbManager.IsDatabaseClientAvailable() {
		if err := u.dbManager.DropDatabaseAndUser(dto.DBRootUser, dto.DBRootPassword, dbName, dbUser); err != nil {
			logger.Warn("Gagal menghapus database otomatis: %v", err)
		} else {
			logger.Success("Database '%s' dan user '%s' berhasil dihapus.", dbName, dbUser)
		}
	}

	_ = u.appRepo.Delete(dto.Name)
	logger.Success("Aplikasi '%s' berhasil dihapus secara menyeluruh.", dto.Name)
	return nil
}

func (u *AppUsecase) List() ([]domain.AppStatusInfo, error) {
	apps, err := u.appRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var results []domain.AppStatusInfo
	for _, a := range apps {
		isActive := u.systemManager.IsServiceActive(a.User, fmt.Sprintf("%s.service", a.Name))
		status := "INACTIVE"
		if isActive {
			status = "ACTIVE"
		}

		var displayPort string
		if a.Stack == domain.StackLaravel {
			if a.PortFE != "N/A" && a.PortFE != "" {
				displayPort = a.PortFE
			} else {
				displayPort = valueOrDefault(a.Port, "-")
			}
		} else if a.Stack == domain.StackNodeFullstack {
			displayPort = fmt.Sprintf("%s/%s", a.PortFE, a.PortBE)
		} else if a.Stack == domain.StackNodeAPI {
			if a.Port != "N/A" && a.Port != "" {
				displayPort = a.Port
			} else {
				displayPort = "-"
			}
		} else {
			if a.Port != "N/A" && a.Port != "" {
				displayPort = a.Port
			} else {
				displayPort = fmt.Sprintf("%s/%s", a.PortFE, a.PortBE)
			}
		}

		results = append(results, domain.AppStatusInfo{
			App:           a,
			Status:        status,
			ServiceActive: isActive,
			DisplayPort:   displayPort,
		})
	}

	return results, nil
}

func (u *AppUsecase) GetByName(name string) (*domain.AppStatusInfo, error) {
	app, err := u.appRepo.FindByName(name)
	if err != nil {
		return nil, err
	}

	isActive := u.systemManager.IsServiceActive(app.User, fmt.Sprintf("%s.service", app.Name))
	status := "INACTIVE"
	if isActive {
		status = "ACTIVE"
	}

	return &domain.AppStatusInfo{
		App:           *app,
		Status:        status,
		ServiceActive: isActive,
	}, nil
}

func (u *AppUsecase) GetLogs(appName string, lines int) (string, error) {
	if !u.systemManager.UserExists(appName) {
		return "", domain.ErrAppNotFound
	}
	serviceName := fmt.Sprintf("%s.service", appName)
	return u.systemManager.GetJournalLogs(appName, serviceName, lines)
}

func valueOrNA(val string) string {
	if val == "" {
		return "N/A"
	}
	return val
}

func valueOrDefault(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}
