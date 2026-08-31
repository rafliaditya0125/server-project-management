package database

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
)

type MySQLManager struct{}

func NewMySQLManager() *MySQLManager {
	return &MySQLManager{}
}

func (m *MySQLManager) getClientCmd() (string, error) {
	if path, err := exec.LookPath("mariadb"); err == nil && path != "" {
		return path, nil
	}
	if path, err := exec.LookPath("mysql"); err == nil && path != "" {
		return path, nil
	}
	return "", fmt.Errorf("neither 'mariadb' nor 'mysql' client binary found in PATH")
}

func (m *MySQLManager) IsDatabaseClientAvailable() bool {
	_, err := m.getClientCmd()
	return err == nil
}

func (m *MySQLManager) buildAuthArgs(rootUser, rootPassword string) []string {
	if rootUser == "" {
		rootUser = "root"
	}
	args := []string{"-u", rootUser}
	if rootPassword != "" {
		args = append(args, fmt.Sprintf("-p%s", rootPassword))
	}
	return args
}

func (m *MySQLManager) VerifyRootConnection(rootUser, rootPassword string) error {
	client, err := m.getClientCmd()
	if err != nil {
		return err
	}

	args := append(m.buildAuthArgs(rootUser, rootPassword), "-e", "SELECT 1;")
	cmd := exec.Command(client, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", domain.ErrDatabaseConnectionFailed, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *MySQLManager) CreateDatabaseAndUser(rootUser, rootPassword, dbName, dbUser, dbPassword string) error {
	client, err := m.getClientCmd()
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		CREATE DATABASE IF NOT EXISTS `+"`%s`"+` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
		CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';
		CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';
		GRANT ALL PRIVILEGES ON `+"`%s`"+`.* TO '%s'@'localhost';
		GRANT ALL PRIVILEGES ON `+"`%s`"+`.* TO '%s'@'%%';
		FLUSH PRIVILEGES;
	`, dbName, dbUser, dbPassword, dbUser, dbPassword, dbName, dbUser, dbName, dbUser)

	args := append(m.buildAuthArgs(rootUser, rootPassword), "-e", query)
	cmd := exec.Command(client, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", domain.ErrDatabaseCreationFailed, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *MySQLManager) DropDatabaseAndUser(rootUser, rootPassword, dbName, dbUser string) error {
	client, err := m.getClientCmd()
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		DROP DATABASE IF EXISTS `+"`%s`"+`;
		DROP USER IF EXISTS '%s'@'localhost';
		DROP USER IF EXISTS '%s'@'%%';
		FLUSH PRIVILEGES;
	`, dbName, dbUser, dbUser)

	args := append(m.buildAuthArgs(rootUser, rootPassword), "-e", query)
	cmd := exec.Command(client, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", domain.ErrDatabaseDropFailed, strings.TrimSpace(string(out)))
	}
	return nil
}
