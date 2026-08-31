package system

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type OSDetector struct{}

func NewOSDetector() *OSDetector {
	return &OSDetector{}
}

func (d *OSDetector) DetectOS() (string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var id, idLike string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		} else if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}

	archRegex := regexp.MustCompile(`(?i)(arch|manjaro|endeavouros|artix|garuda)`)
	debianRegex := regexp.MustCompile(`(?i)(ubuntu|debian|pop|linuxmint|kali|raspbian|elementary)`)
	fedoraRegex := regexp.MustCompile(`(?i)(fedora|rhel|centos|rocky|almalinux)`)

	if archRegex.MatchString(id) || strings.Contains(strings.ToLower(idLike), "arch") {
		return "arch", nil
	}
	if debianRegex.MatchString(id) || strings.Contains(strings.ToLower(idLike), "debian") || strings.Contains(strings.ToLower(idLike), "ubuntu") {
		return "debian", nil
	}
	if fedoraRegex.MatchString(id) || strings.Contains(strings.ToLower(idLike), "fedora") || strings.Contains(strings.ToLower(idLike), "rhel") {
		return "fedora", nil
	}

	if id != "" {
		return id, nil
	}
	return "unknown", nil
}
