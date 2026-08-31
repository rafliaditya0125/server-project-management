package cli

import (
	"fmt"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/rafliaditya0125/server-project-management/pkg/terminal"
	"github.com/spf13/cobra"
)

func (c *CLI) newSetupCmd() *cobra.Command {
	var (
		flagAll         bool
		flagPHP         bool
		flagNode        bool
		flagWeb         bool
		flagDB          bool
		flagFastCGI     bool
		flagSymlink     bool
		flagCompletion  bool
		flagInteractive bool
		flagExcept      string
	)

	setupCmd := &cobra.Command{
		Use:   "setup [flags] [stages...]",
		Short: "Setup dependensi server (PHP, Node.js, Web, DB, FastCGI, Symlink, Shell Completion)",
		Long: `DESKRIPSI:
  Jika tanpa opsi, semua tahap setup akan dijalankan secara otomatis.
  Jika diberikan opsi tahap, hanya tahap yang dipilih yang akan dijalankan.
  Gunakan flag -e atau --except untuk mengecualikan tahap tertentu.

DAFTAR OPSI TAHAP SETUP:
  --all, all               : Jalankan semua tahap setup (default)
  --php, php               : Install PHP, ekstensi umum, dan Composer
  --node, node             : Install Node.js dan NPM
  --web, web               : Install Web Server (Caddy & Nginx)
  --db, db                 : Install Fish Shell dan MariaDB/MySQL Client
  --fastcgi, fastcgi       : Konfigurasi koneksi PHP-FPM FastCGI (Socket / TCP)
  --symlink, symlink       : Buat symlink global ke /usr/local/bin/project
  --completion, completion : Pasang shell autocompletion (Bash, Zsh, Fish)
  -e=<list>, --except=<list>: Kecualikan tahap tertentu (pisahkan koma jika jamak)
  --interactive, -i        : Jalankan setup melalui menu interaktif`,
		ValidArgs: []string{"all", "php", "composer", "node", "npm", "web", "caddy", "nginx", "db", "shell", "fastcgi", "symlink", "completion"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRoot(); err != nil {
				return err
			}

			opts := &domain.SetupOptions{
				All:         flagAll,
				PHP:         flagPHP,
				Node:        flagNode,
				Web:         flagWeb,
				DB:          flagDB,
				FastCGI:     flagFastCGI,
				Symlink:     flagSymlink,
				Completion:  flagCompletion,
				Interactive: flagInteractive,
			}

			// Handle positional stage names
			for _, arg := range args {
				switch strings.ToLower(arg) {
				case "all":
					opts.All = true
				case "php", "composer":
					opts.PHP = true
				case "node", "nodejs", "npm":
					opts.Node = true
				case "web", "caddy", "nginx":
					opts.Web = true
				case "db", "shell", "mariadb", "mysql":
					opts.DB = true
				case "fastcgi", "fpm", "php-fpm":
					opts.FastCGI = true
				case "symlink", "bin":
					opts.Symlink = true
				case "completion", "completions":
					opts.Completion = true
				case "interactive":
					opts.Interactive = true
				}
			}

			if flagExcept != "" {
				parts := strings.Split(flagExcept, ",")
				opts.Except = append(opts.Except, parts...)
			}

			if opts.Interactive {
				return c.runInteractiveSetup()
			}

			return c.setupUsecase.Execute(opts)
		},
	}

	setupCmd.Flags().BoolVar(&flagAll, "all", false, "Jalankan semua tahap setup")
	setupCmd.Flags().BoolVar(&flagPHP, "php", false, "Install PHP, ekstensi umum, dan Composer")
	setupCmd.Flags().BoolVar(&flagNode, "node", false, "Install Node.js dan NPM")
	setupCmd.Flags().BoolVar(&flagWeb, "web", false, "Install Web Server (Caddy & Nginx)")
	setupCmd.Flags().BoolVar(&flagDB, "db", false, "Install Fish Shell dan MariaDB/MySQL Client")
	setupCmd.Flags().BoolVar(&flagFastCGI, "fastcgi", false, "Konfigurasi koneksi PHP-FPM FastCGI (Socket / TCP)")
	setupCmd.Flags().BoolVar(&flagSymlink, "symlink", false, "Buat symlink global ke /usr/local/bin/project")
	setupCmd.Flags().BoolVar(&flagCompletion, "completion", false, "Pasang shell autocompletion (Bash, Zsh, Fish)")
	setupCmd.Flags().BoolVarP(&flagInteractive, "interactive", "i", false, "Jalankan setup via menu interaktif")
	setupCmd.Flags().StringVarP(&flagExcept, "except", "e", "", "Kecualikan tahap tertentu (pisahkan koma jika jamak)")

	_ = setupCmd.RegisterFlagCompletionFunc("except", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		stages := []string{"php", "composer", "node", "npm", "web", "caddy", "nginx", "db", "shell", "fastcgi", "symlink", "completion"}
		return stages, cobra.ShellCompDirectiveNoFileComp
	})

	return setupCmd
}

func (c *CLI) runInteractiveSetup() error {
	cfg, osDistro, _ := c.setupUsecase.GetStatus()

	logger.Raw("\n%s%s=================================================================%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
	logger.Raw("%s%s       SETUP DEPENDENSI & FASTCGI PROJECT MANAGER MULTI-OS      %s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
	logger.Raw("%s%s=================================================================%s\n", logger.ColorBold, logger.ColorCyan, logger.ColorReset)
	logger.Raw("Distro / OS Terdeteksi : %s%s%s\n\n", logger.ColorBold, osDistro, logger.ColorReset)

	fmt.Println("Pilih menu setup yang ingin dijalankan:")
	fmt.Printf("  1. %sSetup Lengkap (Semua Tahap)%s\n", logger.ColorGreen, logger.ColorReset)
	fmt.Println("     (Symlink CLI + Autocompletion + PHP + Node.js + Web Server + Fish/DB + FastCGI)")
	fmt.Println("  2. Install PHP, Ekstensi & Composer")
	fmt.Println("  3. Install Node.js & NPM")
	fmt.Println("  4. Install Web Server (Caddy & Nginx)")
	fmt.Println("  5. Install Fish Shell & Database Client")
	fmt.Println("  6. Konfigurasi PHP-FPM FastCGI (Unix Socket vs TCP Port)")
	fmt.Println("  7. Buat Symlink Global (/usr/local/bin/project)")
	fmt.Println("  8. Pasang Shell Autocompletion (Bash, Zsh, Fish)")
	fmt.Println("  9. Keluar")
	fmt.Println()

	choice := terminal.ReadPrompt("Pilihan menu (1-9)", "1")
	switch choice {
	case "1":
		return c.setupUsecase.Execute(&domain.SetupOptions{All: true})
	case "2":
		return c.setupUsecase.Execute(&domain.SetupOptions{PHP: true})
	case "3":
		return c.setupUsecase.Execute(&domain.SetupOptions{Node: true})
	case "4":
		return c.setupUsecase.Execute(&domain.SetupOptions{Web: true})
	case "5":
		return c.setupUsecase.Execute(&domain.SetupOptions{DB: true})
	case "6":
		currentSock := "/run/php/php8.3-fpm.sock"
		currentPort := "9000"
		if cfg != nil {
			if cfg.PhpSockPath != "" {
				currentSock = cfg.PhpSockPath
			}
			if cfg.PhpPort != "" {
				currentPort = cfg.PhpPort
			}
		}

		fmt.Println("\nPilih mode koneksi PHP-FPM FastCGI default:")
		fmt.Printf("  1. Unix Socket (Default/Rekomendasi, misal: %s)\n", currentSock)
		fmt.Printf("  2. TCP Port (misal: 127.0.0.1:%s)\n", currentPort)

		modeChoice := terminal.ReadPrompt("Pilihan mode (1/2)", "1")
		var mode domain.PhpMode
		var sockPath, port string
		if modeChoice == "2" {
			mode = domain.PhpModePort
			port = terminal.ReadPrompt("Masukkan port TCP PHP-FPM", currentPort)
		} else {
			mode = domain.PhpModeSocket
			sockPath = terminal.ReadPrompt("Masukkan path Unix Socket PHP-FPM", currentSock)
		}

		_, err := c.setupUsecase.ConfigureFastCGI(osDistro, mode, sockPath, port)
		return err
	case "7":
		return c.setupUsecase.Execute(&domain.SetupOptions{Symlink: true})
	case "8":
		return c.setupUsecase.Execute(&domain.SetupOptions{Completion: true})
	case "9":
		logger.Info("Keluar dari menu setup.")
		return nil
	default:
		logger.Error("Pilihan tidak valid.")
		return nil
	}
}
