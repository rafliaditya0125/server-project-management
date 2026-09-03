package completion

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rafliaditya0125/server-project-management/pkg/logger"
	"github.com/spf13/cobra"
)

type ShellCompletionManager struct {
	rootCmd *cobra.Command
}

func NewShellCompletionManager(rootCmd *cobra.Command) *ShellCompletionManager {
	return &ShellCompletionManager{rootCmd: rootCmd}
}

func (m *ShellCompletionManager) InstallShellCompletions() error {
	logger.Info("Mengonfigurasi shell autocompletion untuk Bash, Zsh, dan Fish...")

	// 1. Bash Completion
	var bashBuf bytes.Buffer
	if m.rootCmd != nil {
		_ = m.rootCmd.GenBashCompletion(&bashBuf)
	}

	bashDirs := []string{
		"/etc/bash_completion.d",
		"/usr/share/bash-completion/completions",
		"/etc/profile.d",
	}
	for _, bdir := range bashDirs {
		_ = os.MkdirAll(bdir, 0755)
		if bdir == "/etc/profile.d" {
			_ = os.WriteFile(filepath.Join(bdir, "project_completion.sh"), bashBuf.Bytes(), 0644)
		} else {
			_ = os.WriteFile(filepath.Join(bdir, "project"), bashBuf.Bytes(), 0644)
		}
	}

	// 2. Zsh Completion
	var zshBuf bytes.Buffer
	if m.rootCmd != nil {
		_ = m.rootCmd.GenZshCompletion(&zshBuf)
	}

	zshDirs := []string{
		"/usr/share/zsh/site-functions",
		"/usr/local/share/zsh/site-functions",
		"/usr/share/zsh/vendor-completions",
		"/etc/zsh/site-functions",
	}
	for _, zdir := range zshDirs {
		_ = os.MkdirAll(zdir, 0755)
		_ = os.WriteFile(filepath.Join(zdir, "_project"), zshBuf.Bytes(), 0644)
	}

	// 3. Fish Completion
	var fishBuf bytes.Buffer
	if m.rootCmd != nil {
		_ = m.rootCmd.GenFishCompletion(&fishBuf, true)
	}

	fishDirs := []string{
		"/usr/share/fish/vendor_completions.d",
		"/etc/fish/completions",
	}
	for _, fdir := range fishDirs {
		_ = os.MkdirAll(fdir, 0755)
		_ = os.WriteFile(filepath.Join(fdir, "project.fish"), fishBuf.Bytes(), 0644)
	}

	// Also install in user home directories for fish
	homeMatches, _ := filepath.Glob("/home/*")
	homeMatches = append(homeMatches, "/root")
	for _, uHome := range homeMatches {
		fishUserDir := filepath.Join(uHome, ".config", "fish", "completions")
		if _, err := os.Stat(filepath.Join(uHome, ".config", "fish")); err == nil {
			_ = os.MkdirAll(fishUserDir, 0755)
			_ = os.WriteFile(filepath.Join(fishUserDir, "project.fish"), fishBuf.Bytes(), 0644)
		}
	}

	logger.Success("Autocompletion berhasil dipasang untuk Bash, Zsh, dan Fish!")
	logger.Info("Buka terminal baru atau jalankan: source /etc/profile.d/project_completion.sh")
	return nil
}

func (m *ShellCompletionManager) Generate(shellType string) (string, error) {
	var buf bytes.Buffer
	switch shellType {
	case "bash":
		err := m.rootCmd.GenBashCompletion(&buf)
		return buf.String(), err
	case "zsh":
		err := m.rootCmd.GenZshCompletion(&buf)
		return buf.String(), err
	case "fish":
		err := m.rootCmd.GenFishCompletion(&buf, true)
		return buf.String(), err
	default:
		return "", fmt.Errorf("unsupported shell: %s", shellType)
	}
}
