package symlink

import (
	"os"
	"path/filepath"

	"github.com/rafliaditya0125/server-project-management/pkg/logger"
)

type SymlinkManager struct{}

func NewSymlinkManager() *SymlinkManager {
	return &SymlinkManager{}
}

func (s *SymlinkManager) CreateGlobalSymlink(sourceBinaryPath string) error {
	if sourceBinaryPath == "" {
		execPath, err := os.Executable()
		if err == nil {
			sourceBinaryPath = execPath
		}
	}

	absPath, err := filepath.Abs(sourceBinaryPath)
	if err != nil {
		absPath = sourceBinaryPath
	}

	logger.Info("Membuat symlink global CLI ke /usr/local/bin/project...")
	_ = os.MkdirAll("/usr/local/bin", 0755)
	_ = os.Chmod(absPath, 0755)

	targetLink := "/usr/local/bin/project"
	_ = os.Remove(targetLink)
	if err := os.Symlink(absPath, targetLink); err != nil {
		return err
	}

	logger.Success("Symlink global berhasil dibuat: /usr/local/bin/project -> %s", absPath)
	logger.Info("Anda sekarang dapat menjalankan perintah secara langsung: 'sudo project <command>'")
	return nil
}
