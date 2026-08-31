package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type GitManager struct{}

func NewGitManager() *GitManager {
	return &GitManager{}
}

func (g *GitManager) Clone(repoURL, targetDir string) error {
	tmpDir := filepath.Join("/tmp", fmt.Sprintf("project_git_clone_%d", os.Getpid()))
	_ = os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "clone", repoURL, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s (%w)", string(out), err)
	}

	// Copy files from tmpDir to targetDir
	copyCmd := exec.Command("cp", "-a", tmpDir+"/.", targetDir+"/")
	if out, err := copyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy files from clone failed: %s (%w)", string(out), err)
	}

	return nil
}
