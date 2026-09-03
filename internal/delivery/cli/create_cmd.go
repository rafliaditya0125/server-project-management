package cli

import (
	"fmt"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/rafliaditya0125/server-project-management/pkg/terminal"
	"github.com/spf13/cobra"
)

func (c *CLI) newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Membuat user sistem terisolasi, database, dan konfigurasi aplikasi",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRoot(); err != nil {
				return err
			}

			logger.Raw("\n%s%s=== Wizard Pembuatan Aplikasi Terisolasi ===%s\n\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)

			// 1. Input Nama Aplikasi
			var appName string
			for {
				appName = terminal.ReadPrompt("Masukkan nama aplikasi (username)", "")
				appName = strings.TrimSpace(appName)
				if appName == "" {
					logger.Error("Nama aplikasi tidak boleh kosong.")
					continue
				}
				exists, _ := c.appRepo.Exists(appName)
				if exists {
					logger.Error("Aplikasi '%s' sudah terdaftar di registry.", appName)
					continue
				}
				break
			}

			// 2. Input Password User
			var userPassword string
			for {
				p1, err := terminal.ReadPassword(fmt.Sprintf("Masukkan password untuk user '%s': ", appName))
				if err != nil || p1 == "" {
					logger.Error("Password tidak boleh kosong.")
					continue
				}
				p2, err := terminal.ReadPassword("Konfirmasi password: ")
				if err != nil || p1 != p2 {
					logger.Error("Konfirmasi password tidak cocok. Silakan coba lagi.")
					continue
				}
				userPassword = p1
				break
			}

			// 3. Pilihan Stack
			fmt.Println("\nPilih Stack Aplikasi:")
			fmt.Println("  1. Laravel (PHP-FPM)")
			fmt.Println("  2. Node.js (Fullstack / Static FE + API BE)")
			fmt.Println("  3. Node.js (Standalone API Only - Direct Node Runtime)")
			fmt.Println("  4. Node.js FE + Laravel BE (FastCGI Reverse Proxy)")

			var stackType domain.StackType
			for {
				choice := terminal.ReadPrompt("Pilihan (1/2/3/4)", "1")
				switch choice {
				case "1":
					stackType = domain.StackLaravel
				case "2":
					stackType = domain.StackNodeFullstack
				case "3":
					stackType = domain.StackNodeAPI
				case "4":
					stackType = domain.StackNodeLaravel
				default:
					logger.Error("Pilihan tidak valid. Masukkan 1, 2, 3, atau 4.")
					continue
				}
				break
			}

			// 4. Link Git Repository (Opsional)
			gitRepo := terminal.ReadPrompt("Link repositori git (Opsional, tekan Enter untuk lewati)", "")

			// 5. Port & Web Server Configuration
			var portFE, portBE, portSingle string
			var webServer domain.WebServerType = domain.WebServerCaddy
			var phpMode domain.PhpMode = domain.PhpModeSocket
			var phpSockPath, phpTcpPort string

			cfg, _ := c.configRepo.Get()
			defaultSock := "/run/php/php8.3-fpm.sock"
			defaultPhpPort := "9000"
			if cfg != nil {
				if cfg.PhpSockPath != "" {
					defaultSock = cfg.PhpSockPath
				}
				if cfg.PhpPort != "" {
					defaultPhpPort = cfg.PhpPort
				}
			}

			if stackType == domain.StackLaravel {
				for {
					portFE = terminal.ReadPrompt("Masukkan Port Web / HTTP Aplikasi", "8000")
					if portFE != "" {
						break
					}
				}

				fmt.Println("\nPilih Mode Koneksi PHP-FPM FastCGI:")
				fmt.Printf("  1. Unix Socket (Default: %s)\n", defaultSock)
				fmt.Printf("  2. TCP Port (Default: 127.0.0.1:%s)\n", defaultPhpPort)

				defaultFpmChoice := "1"
				if cfg != nil && cfg.PhpMode == domain.PhpModePort {
					defaultFpmChoice = "2"
				}

				fpmChoice := terminal.ReadPrompt("Pilihan Mode FastCGI (1/2)", defaultFpmChoice)
				if fpmChoice == "2" {
					phpMode = domain.PhpModePort
					phpTcpPort = terminal.ReadPrompt("Masukkan Port TCP PHP-FPM", defaultPhpPort)
				} else {
					phpMode = domain.PhpModeSocket
					phpSockPath = terminal.ReadPrompt("Masukkan Path Unix Socket PHP-FPM", defaultSock)
				}

				fmt.Println("\nPilih Web Server:")
				fmt.Println("  1. Caddy (Ringkas & zero-temp folder)")
				fmt.Println("  2. Nginx (User-space instance)")
				for {
					wsChoice := terminal.ReadPrompt("Pilihan Web Server (1/2)", "1")
					if wsChoice == "1" {
						webServer = domain.WebServerCaddy
						break
					} else if wsChoice == "2" {
						webServer = domain.WebServerNginx
						break
					}
					logger.Error("Pilihan tidak valid. Masukkan 1 atau 2.")
				}

			} else if stackType == domain.StackNodeFullstack {
				for {
					portFE = terminal.ReadPrompt("Masukkan Port Frontend", "8080")
					if portFE != "" {
						break
					}
				}
				for {
					portBE = terminal.ReadPrompt("Masukkan Port Backend API", "3001")
					if portBE == portFE {
						logger.Error("Port Backend API tidak boleh sama dengan Port Frontend.")
						continue
					}
					if portBE != "" {
						break
					}
				}

				fmt.Println("\nPilih Web Server:")
				fmt.Println("  1. Caddy (Ringkas & zero-temp folder)")
				fmt.Println("  2. Nginx (User-space instance)")
				for {
					wsChoice := terminal.ReadPrompt("Pilihan Web Server (1/2)", "1")
					if wsChoice == "1" {
						webServer = domain.WebServerCaddy
						break
					} else if wsChoice == "2" {
						webServer = domain.WebServerNginx
						break
					}
					logger.Error("Pilihan tidak valid. Masukkan 1 atau 2.")
				}

			} else if stackType == domain.StackNodeAPI {
				for {
					portSingle = terminal.ReadPrompt("Masukkan Port Aplikasi", "3000")
					if portSingle != "" {
						break
					}
				}
			} else if stackType == domain.StackNodeLaravel {
				for {
					portFE = terminal.ReadPrompt("Masukkan Port Web / HTTP Frontend", "8080")
					if portFE != "" {
						break
					}
				}

				fmt.Println("\nPilih Mode Koneksi PHP-FPM FastCGI (untuk Laravel BE):")
				fmt.Printf("  1. Unix Socket (Default: %s)\n", defaultSock)
				fmt.Printf("  2. TCP Port (Default: 127.0.0.1:%s)\n", defaultPhpPort)

				defaultFpmChoice := "1"
				if cfg != nil && cfg.PhpMode == domain.PhpModePort {
					defaultFpmChoice = "2"
				}

				fpmChoice := terminal.ReadPrompt("Pilihan Mode FastCGI (1/2)", defaultFpmChoice)
				if fpmChoice == "2" {
					phpMode = domain.PhpModePort
					phpTcpPort = terminal.ReadPrompt("Masukkan Port TCP PHP-FPM", defaultPhpPort)
				} else {
					phpMode = domain.PhpModeSocket
					phpSockPath = terminal.ReadPrompt("Masukkan Path Unix Socket PHP-FPM", defaultSock)
				}

				fmt.Println("\nPilih Web Server:")
				fmt.Println("  1. Caddy (Ringkas & zero-temp folder)")
				fmt.Println("  2. Nginx (User-space instance)")
				for {
					wsChoice := terminal.ReadPrompt("Pilihan Web Server (1/2)", "1")
					if wsChoice == "1" {
						webServer = domain.WebServerCaddy
						break
					} else if wsChoice == "2" {
						webServer = domain.WebServerNginx
						break
					}
					logger.Error("Pilihan tidak valid. Masukkan 1 atau 2.")
				}
			}

			// 6. Database Inputs
			logger.Raw("\n%s%s--- Konfigurasi Database ---%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
			dbName := terminal.ReadPrompt("Masukkan nama database", appName)
			dbUser := terminal.ReadPrompt("Masukkan username database", appName)

			var dbPassword string
			for {
				p, err := terminal.ReadPassword(fmt.Sprintf("Masukkan password database untuk user '%s': ", dbUser))
				if err != nil || p == "" {
					logger.Error("Password database tidak boleh kosong.")
					continue
				}
				dbPassword = p
				break
			}

			logger.Raw("\n%sKredensial Root Database (untuk membuat database & user):%s\n", logger.ColorYellow, logger.ColorReset)
			dbRootUser := terminal.ReadPrompt("Username Root Database", "root")
			dbRootPass, _ := terminal.ReadPassword("Password Root Database: ")

			dto := &domain.CreateAppDTO{
				Name:           appName,
				UserPassword:   userPassword,
				Stack:          stackType,
				GitRepo:        gitRepo,
				PortFE:         portFE,
				PortBE:         portBE,
				PortSingle:     portSingle,
				PhpFpmMode:     phpMode,
				PhpSockPath:    phpSockPath,
				PhpTcpPort:     phpTcpPort,
				WebServer:      webServer,
				DBName:         dbName,
				DBUser:         dbUser,
				DBPassword:     dbPassword,
				DBRootUser:     dbRootUser,
				DBRootPassword: dbRootPass,
			}

			createdApp, err := c.appUsecase.Create(dto)
			if err != nil {
				return err
			}

			logger.Raw("\n%s%s=================================================================%s\n", logger.ColorBold, logger.ColorGreen, logger.ColorReset)
			logger.Raw("%s%s        APLIKASI '%s' BERHASIL DIBUAT!        %s\n", logger.ColorBold, logger.ColorGreen, createdApp.Name, logger.ColorReset)
			logger.Raw("%s%s=================================================================%s\n", logger.ColorBold, logger.ColorGreen, logger.ColorReset)
			fmt.Printf("  - Username / App : %s%s%s\n", logger.ColorBold, createdApp.Name, logger.ColorReset)
			fmt.Printf("  - Home Directory : %s\n", createdApp.Home)
			fmt.Printf("  - Stack          : %s\n", createdApp.StackName)
			if createdApp.Stack == domain.StackLaravel {
				fmt.Printf("  - Web Server     : %s\n", createdApp.WebServer)
				fmt.Printf("  - Port Web/HTTP  : %s%s%s\n", logger.ColorBold, createdApp.PortFE, logger.ColorReset)
				fmt.Printf("  - FastCGI Target : %s%s%s\n", logger.ColorBold, createdApp.PortBE, logger.ColorReset)
			} else if createdApp.Stack == domain.StackNodeFullstack {
				fmt.Printf("  - Web Server     : %s\n", createdApp.WebServer)
				fmt.Printf("  - Port Frontend  : %s%s%s\n", logger.ColorBold, createdApp.PortFE, logger.ColorReset)
				fmt.Printf("  - Port Backend   : %s%s%s\n", logger.ColorBold, createdApp.PortBE, logger.ColorReset)
			} else if createdApp.Stack == domain.StackNodeLaravel {
				fmt.Printf("  - Web Server     : %s\n", createdApp.WebServer)
				fmt.Printf("  - Port Web/HTTP  : %s%s%s\n", logger.ColorBold, createdApp.PortFE, logger.ColorReset)
				fmt.Printf("  - FastCGI BE     : %s%s%s\n", logger.ColorBold, createdApp.PortBE, logger.ColorReset)
				fmt.Printf("  - Direktori FE   : %s/fe/dist/\n", createdApp.Home)
				fmt.Printf("  - Direktori BE   : %s/be/public/\n", createdApp.Home)
			} else if createdApp.Stack == domain.StackNodeAPI {
				fmt.Printf("  - Port Aplikasi  : %s%s%s\n", logger.ColorBold, createdApp.Port, logger.ColorReset)
			}
			fmt.Printf("  - Database Name  : %s\n", createdApp.DBName)
			fmt.Printf("  - Database User  : %s\n", createdApp.DBUser)
			fmt.Printf("  - Service Name   : %s.service (systemd user)\n", createdApp.Name)
			fmt.Println("-----------------------------------------------------------------")

			return nil
		},
	}
}
