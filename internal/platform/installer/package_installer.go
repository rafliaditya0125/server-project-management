package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/pkg/logger"
)

type PackageInstaller struct {
	fastcgiDetector *FastCGIDetector
}

func NewPackageInstaller(detector *FastCGIDetector) *PackageInstaller {
	return &PackageInstaller{fastcgiDetector: detector}
}

func (i *PackageInstaller) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (i *PackageInstaller) InstallPHPAndComposer(osDistro string) error {
	logger.Info("Menginstal PHP, ekstensi umum, dan Composer untuk OS: %s...", osDistro)

	switch osDistro {
	case "arch":
		err := i.runCommand("pacman", "-Sy", "--noconfirm", "--needed",
			"php", "php-fpm", "php-gd", "php-intl", "php-sodium", "php-sqlite", "composer", "curl", "git", "unzip")
		if err != nil {
			return err
		}

		// Enable common extensions in /etc/php/php.ini
		phpIniPath := "/etc/php/php.ini"
		if data, err := os.ReadFile(phpIniPath); err == nil {
			logger.Info("Mengaktifkan ekstensi PHP umum pada /etc/php/php.ini...")
			content := string(data)
			exts := []string{"curl", "pdo_mysql", "mysqli", "mbstring", "openssl", "zip", "fileinfo", "gd", "intl", "sodium", "iconv", "bcmath"}
			for _, ext := range exts {
				content = strings.ReplaceAll(content, fmt.Sprintf(";extension=%s", ext), fmt.Sprintf("extension=%s", ext))
			}
			_ = os.WriteFile(phpIniPath, []byte(content), 0644)
		}

	case "debian":
		_ = i.runCommand("apt-get", "update")
		envCmd := exec.Command("apt-get", "install", "-y",
			"php-cli", "php-fpm", "php-mysql", "php-mbstring", "php-xml", "php-curl", "php-zip", "php-intl", "php-bcmath", "php-gd", "php-sqlite3",
			"curl", "git", "unzip", "composer")
		envCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		envCmd.Stdout = os.Stdout
		envCmd.Stderr = os.Stderr
		if err := envCmd.Run(); err != nil {
			logger.Warn("Composer via apt gagal/tidak tersedia. Mengunduh installer resmi Composer...")
			_ = exec.Command("sh", "-c", "curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer").Run()
		}

	case "fedora":
		err := i.runCommand("dnf", "install", "-y",
			"php", "php-fpm", "php-mysqlnd", "php-mbstring", "php-xml", "php-curl", "php-zip", "php-intl", "php-bcmath", "php-gd", "composer", "curl", "git", "unzip")
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("%w: %s", domain.ErrUnknownOS, osDistro)
	}

	logger.Success("PHP dan Composer berhasil diinstal!")
	return nil
}

func (i *PackageInstaller) InstallNodeAndNPM(osDistro string) error {
	logger.Info("Menginstal Node.js dan NPM untuk OS: %s...", osDistro)

	switch osDistro {
	case "arch":
		if err := i.runCommand("pacman", "-Sy", "--noconfirm", "--needed", "nodejs", "npm"); err != nil {
			return err
		}
	case "debian":
		_ = i.runCommand("apt-get", "update")
		cmd := exec.Command("apt-get", "install", "-y", "nodejs", "npm")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	case "fedora":
		if err := i.runCommand("dnf", "install", "-y", "nodejs", "npm"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s", domain.ErrUnknownOS, osDistro)
	}

	logger.Success("Node.js dan NPM berhasil diinstal!")
	return nil
}

func (i *PackageInstaller) InstallWebServers(osDistro string) error {
	logger.Info("Menginstal Web Server (Caddy & Nginx) untuk OS: %s...", osDistro)

	switch osDistro {
	case "arch":
		if err := i.runCommand("pacman", "-Sy", "--noconfirm", "--needed", "caddy", "nginx"); err != nil {
			return err
		}
	case "debian":
		_ = i.runCommand("apt-get", "update")
		cmdNginx := exec.Command("apt-get", "install", "-y", "nginx")
		cmdNginx.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		_ = cmdNginx.Run()

		if _, err := exec.LookPath("caddy"); err != nil {
			logger.Info("Mengonfigurasi repositori resmi Caddy untuk Debian/Ubuntu...")
			cmdDep := exec.Command("apt-get", "install", "-y", "debian-keyring", "debian-archive-keyring", "apt-transport-https", "curl", "gnupg")
			cmdDep.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
			_ = cmdDep.Run()

			_ = exec.Command("sh", "-c", "curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg --yes").Run()
			_ = exec.Command("sh", "-c", "curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list").Run()
			_ = i.runCommand("apt-get", "update")

			cmdCaddy := exec.Command("apt-get", "install", "-y", "caddy")
			cmdCaddy.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
			_ = cmdCaddy.Run()
		}
	case "fedora":
		if err := i.runCommand("dnf", "install", "-y", "caddy", "nginx"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s", domain.ErrUnknownOS, osDistro)
	}

	logger.Success("Web Server (Caddy & Nginx) berhasil diinstal!")
	return nil
}

func (i *PackageInstaller) InstallShellAndDBClient(osDistro string) error {
	logger.Info("Menginstal Fish Shell dan MariaDB/MySQL Client untuk OS: %s...", osDistro)

	switch osDistro {
	case "arch":
		if err := i.runCommand("pacman", "-Sy", "--noconfirm", "--needed", "fish", "mariadb-clients"); err != nil {
			return err
		}
	case "debian":
		_ = i.runCommand("apt-get", "update")
		cmd := exec.Command("apt-get", "install", "-y", "fish", "default-mysql-client")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		if err := cmd.Run(); err != nil {
			cmdFallback := exec.Command("apt-get", "install", "-y", "fish", "mariadb-client")
			cmdFallback.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
			_ = cmdFallback.Run()
		}
	case "fedora":
		if err := i.runCommand("dnf", "install", "-y", "fish", "mariadb"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s", domain.ErrUnknownOS, osDistro)
	}

	logger.Success("Fish shell dan Database client berhasil diinstal!")
	return nil
}

func (i *PackageInstaller) DetectPhpFpmSocket() string {
	return i.fastcgiDetector.DetectPhpFpmSocket()
}

func (i *PackageInstaller) DetectPhpFpmService() string {
	return i.fastcgiDetector.DetectPhpFpmService()
}

func (i *PackageInstaller) EnableAndStartPhpService(serviceName string) error {
	cmd := exec.Command("systemctl", "enable", "--now", serviceName)
	return cmd.Run()
}
