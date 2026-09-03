package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
)

type SystemManager struct{}

func NewSystemManager() *SystemManager {
	return &SystemManager{}
}

func (s *SystemManager) IsRoot() bool {
	return os.Geteuid() == 0
}

func (s *SystemManager) UserExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

func (s *SystemManager) GetFishOrBashShell() string {
	if path, err := exec.LookPath("fish"); err == nil && path != "" {
		return path
	}
	if _, err := os.Stat("/usr/bin/fish"); err == nil {
		return "/usr/bin/fish"
	}
	return "/bin/bash"
}

func (s *SystemManager) CreateUser(username, password, homeDir, shellPath string) error {
	if !s.IsRoot() {
		return domain.ErrPermissionDenied
	}

	if s.UserExists(username) {
		return domain.ErrUserAlreadyExists
	}

	// 1. useradd -m -d <homeDir> -s <shellPath> <username>
	cmd := exec.Command("useradd", "-m", "-d", homeDir, "-s", shellPath, username)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd failed: %s (%w)", string(out), err)
	}

	// 2. Set password via chpasswd
	chpasswdCmd := exec.Command("chpasswd")
	chpasswdCmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s\n", username, password))
	if out, err := chpasswdCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chpasswd failed: %s (%w)", string(out), err)
	}

	// 3. Setup Shell profile agar $XDG_RUNTIME_DIR otomatis terpasang saat su/login
	s.setupUserShellProfile(homeDir)

	return nil
}

func (s *SystemManager) setupUserShellProfile(homeDir string) {
	// Fish config
	fishConfDir := filepath.Join(homeDir, ".config", "fish")
	_ = os.MkdirAll(fishConfDir, 0755)
	fishConfPath := filepath.Join(fishConfDir, "config.fish")
	fishSnippet := "\n# Auto-export XDG_RUNTIME_DIR for user systemd session\nif test -z \"$XDG_RUNTIME_DIR\"\n    set -x XDG_RUNTIME_DIR /run/user/(id -u)\nend\n"
	if data, err := os.ReadFile(fishConfPath); err == nil {
		if !strings.Contains(string(data), "XDG_RUNTIME_DIR") {
			_ = os.WriteFile(fishConfPath, append(data, []byte(fishSnippet)...), 0644)
		}
	} else {
		_ = os.WriteFile(fishConfPath, []byte(fishSnippet), 0644)
	}

	// Bash config
	bashrcPath := filepath.Join(homeDir, ".bashrc")
	bashSnippet := "\n# Auto-export XDG_RUNTIME_DIR for user systemd session\nif [ -z \"$XDG_RUNTIME_DIR\" ]; then\n    export XDG_RUNTIME_DIR=/run/user/$(id -u)\nfi\n"
	if data, err := os.ReadFile(bashrcPath); err == nil {
		if !strings.Contains(string(data), "XDG_RUNTIME_DIR") {
			_ = os.WriteFile(bashrcPath, append(data, []byte(bashSnippet)...), 0644)
		}
	} else {
		_ = os.WriteFile(bashrcPath, []byte(bashSnippet), 0644)
	}
}

func (s *SystemManager) DeleteUser(username string, removeHome bool) error {
	if !s.IsRoot() {
		return domain.ErrPermissionDenied
	}

	if !s.UserExists(username) {
		return nil
	}

	args := []string{"-r", username}
	if !removeHome {
		args = []string{username}
	}

	cmd := exec.Command("userdel", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback with force -f
		cmdForce := exec.Command("userdel", "-f", username)
		if outForce, errForce := cmdForce.CombinedOutput(); errForce != nil {
			return fmt.Errorf("userdel failed: %s | force failed: %s (%w)", string(out), string(outForce), err)
		}
	}

	return nil
}

func (s *SystemManager) EnableLinger(username string) error {
	cmd := exec.Command("loginctl", "enable-linger", username)
	return cmd.Run()
}

func (s *SystemManager) DisableLinger(username string) error {
	cmd := exec.Command("loginctl", "disable-linger", username)
	return cmd.Run()
}

func (s *SystemManager) KillUserProcesses(username string) error {
	cmd := exec.Command("pkill", "-u", username)
	_ = cmd.Run()
	return nil
}

func (s *SystemManager) SetPermissions(rootPath string, username string, mode uint32) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("lookup user %s failed: %w", username, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}

	// Walk recursively and chown/chmod
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		_ = os.Chown(path, uid, gid)
		return nil
	})
	if err != nil {
		return err
	}

	return os.Chmod(rootPath, os.FileMode(mode))
}

func (s *SystemManager) RunSystemctlUser(username string, action string, serviceName string) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("user lookup failed: %w", err)
	}

	// Build arguments: do not append empty serviceName
	baseArgs := []string{"--user", "-M", fmt.Sprintf("%s@.host", username), action}
	if serviceName != "" {
		baseArgs = append(baseArgs, serviceName)
	}

	// 1. Try systemctl --user -M <username>@.host <action> [serviceName]
	cmd1 := exec.Command("systemctl", baseArgs...)
	var out1 bytes.Buffer
	var errOut1 bytes.Buffer
	cmd1.Stdout = &out1
	cmd1.Stderr = &errOut1
	if err := cmd1.Run(); err == nil {
		return out1.String(), nil
	}

	// 2. Fallback via XDG_RUNTIME_DIR and runuser
	runtimeDir := fmt.Sprintf("/run/user/%s", u.Uid)
	if _, statErr := os.Stat(runtimeDir); statErr == nil {
		runuserArgs := []string{
			"-u", username, "--", "env",
			fmt.Sprintf("XDG_RUNTIME_DIR=%s", runtimeDir),
			"systemctl", "--user", action,
		}
		if serviceName != "" {
			runuserArgs = append(runuserArgs, serviceName)
		}
		cmd2 := exec.Command("runuser", runuserArgs...)
		var out2 bytes.Buffer
		cmd2.Stdout = &out2
		if err := cmd2.Run(); err == nil {
			return out2.String(), nil
		}
	}

	// 3. Fallback standard runuser
	cmd3Args := []string{"-u", username, "--", "systemctl", "--user", action}
	if serviceName != "" {
		cmd3Args = append(cmd3Args, serviceName)
	}
	cmd3 := exec.Command("runuser", cmd3Args...)
	var out3 bytes.Buffer
	cmd3.Stdout = &out3
	if err := cmd3.Run(); err == nil {
		return out3.String(), nil
	}

	return out1.String() + errOut1.String(), fmt.Errorf("failed to run systemctl --user %s %s for %s", action, serviceName, username)
}

func (s *SystemManager) IsServiceActive(username string, serviceName string) bool {
	u, err := user.Lookup(username)
	if err != nil {
		return false
	}

	// 1. Cek langsung via cgroup v2 (/sys/fs/cgroup)
	// Sangat cepat, akurat, dan bisa dibaca oleh user non-root maupun root tanpa D-Bus permission error!
	cgroupProcsPath := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%s.slice/user@%s.service/app.slice/%s/cgroup.procs", u.Uid, u.Uid, serviceName)
	if data, err := os.ReadFile(cgroupProcsPath); err == nil {
		if len(bytes.TrimSpace(data)) > 0 {
			return true
		}
	}

	checkActive := func(out string) bool {
		return strings.Contains(out, "ActiveState=active")
	}

	// 2. systemctl show via machinectl (-M username@.host)
	cmd1 := exec.Command("systemctl", "--user", "-M", fmt.Sprintf("%s@.host", username),
		"show", "--property=ActiveState", serviceName)
	if out, err := cmd1.Output(); err == nil {
		if checkActive(string(out)) {
			return true
		}
	}

	// 3. Fallback via XDG_RUNTIME_DIR + runuser
	runtimeDir := fmt.Sprintf("/run/user/%s", u.Uid)
	if _, statErr := os.Stat(runtimeDir); statErr == nil {
		cmd2 := exec.Command("runuser", "-u", username, "--", "env",
			fmt.Sprintf("XDG_RUNTIME_DIR=%s", runtimeDir),
			"systemctl", "--user", "show", "--property=ActiveState", serviceName,
		)
		if out, err := cmd2.Output(); err == nil {
			if checkActive(string(out)) {
				return true
			}
		}
	}

	// 4. Fallback is-active via machinectl
	cmd3 := exec.Command("systemctl", "--user", "-M", fmt.Sprintf("%s@.host", username),
		"is-active", serviceName)
	out3, _ := cmd3.Output()
	if strings.TrimSpace(string(out3)) == "active" {
		return true
	}

	return false
}

func (s *SystemManager) GetJournalLogs(username string, serviceName string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}

	cmd1 := exec.Command("journalctl", "--user", "-M", fmt.Sprintf("%s@.host", username), "-u", serviceName, "-n", strconv.Itoa(lines), "--no-pager")
	out1, err1 := cmd1.CombinedOutput()
	if err1 == nil && len(out1) > 0 {
		return string(out1), nil
	}

	u, err := user.Lookup(username)
	if err == nil {
		cmd2 := exec.Command("journalctl", "-u", fmt.Sprintf("user@%s.service", u.Uid), "-n", strconv.Itoa(lines), "--no-pager")
		out2, err2 := cmd2.CombinedOutput()
		if err2 == nil && len(out2) > 0 {
			return string(out2), nil
		}
	}

	if len(out1) > 0 {
		return string(out1), nil
	}
	return "", fmt.Errorf("could not read journalctl logs: %w", err1)
}
