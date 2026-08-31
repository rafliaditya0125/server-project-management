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

	return nil
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

	// 1. Try systemctl --user -M <username>@ <action> <serviceName>
	cmd1 := exec.Command("systemctl", "--user", "-M", fmt.Sprintf("%s@", username), action, serviceName)
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
		busAddress := fmt.Sprintf("unix:path=%s/bus", runtimeDir)
		cmd2 := exec.Command("runuser", "-u", username, "--", "env",
			fmt.Sprintf("XDG_RUNTIME_DIR=%s", runtimeDir),
			fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=%s", busAddress),
			"systemctl", "--user", action, serviceName,
		)
		var out2 bytes.Buffer
		cmd2.Stdout = &out2
		if err := cmd2.Run(); err == nil {
			return out2.String(), nil
		}
	}

	// 3. Fallback standard runuser
	cmd3 := exec.Command("runuser", "-u", username, "--", "systemctl", "--user", action, serviceName)
	var out3 bytes.Buffer
	cmd3.Stdout = &out3
	if err := cmd3.Run(); err == nil {
		return out3.String(), nil
	}

	return out1.String() + errOut1.String(), fmt.Errorf("failed to run systemctl --user %s %s for %s", action, serviceName, username)
}

func (s *SystemManager) IsServiceActive(username string, serviceName string) bool {
	out, err := s.RunSystemctlUser(username, "is-active", serviceName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "active"
}

func (s *SystemManager) GetJournalLogs(username string, serviceName string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}

	cmd1 := exec.Command("journalctl", "--user", "-M", fmt.Sprintf("%s@", username), "-u", serviceName, "-n", strconv.Itoa(lines), "--no-pager")
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
