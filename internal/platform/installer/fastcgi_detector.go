package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FastCGIDetector struct{}

func NewFastCGIDetector() *FastCGIDetector {
	return &FastCGIDetector{}
}

func (d *FastCGIDetector) DetectPhpFpmSocket() string {
	possibleSocks := []string{
		"/run/php/php8.4-fpm.sock",
		"/run/php/php8.3-fpm.sock",
		"/run/php/php8.2-fpm.sock",
		"/run/php/php8.1-fpm.sock",
		"/run/php/php-fpm.sock",
		"/run/php-fpm/php-fpm.sock",
		"/run/php-fpm/www.sock",
		"/var/run/php/php8.3-fpm.sock",
		"/var/run/php/php8.2-fpm.sock",
		"/var/run/php/php-fpm.sock",
	}

	for _, sock := range possibleSocks {
		if fi, err := os.Stat(sock); err == nil {
			if fi.Mode()&os.ModeSocket != 0 || fi.Mode().IsRegular() {
				return sock
			}
		}
	}

	// Wildcard scan
	for _, dir := range []string{"/run/php", "/run/php-fpm", "/var/run/php"} {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.sock"))
		if len(matches) > 0 {
			return matches[0]
		}
	}

	return "/run/php/php8.3-fpm.sock"
}

func (d *FastCGIDetector) DetectPhpFpmService() string {
	candidates := []string{
		"php8.4-fpm",
		"php8.3-fpm",
		"php8.2-fpm",
		"php8.1-fpm",
		"php-fpm",
		"php7.4-fpm",
	}

	for _, svc := range candidates {
		cmd := exec.Command("systemctl", "list-unit-files", svc+".service")
		out, err := cmd.Output()
		if err == nil && strings.Contains(string(out), svc) {
			return svc
		}
	}

	return "php-fpm"
}
