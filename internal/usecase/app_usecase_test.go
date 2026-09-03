package usecase_test

import (
	"testing"

	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/internal/usecase"
)

// Mock AppRepository
type mockAppRepo struct {
	apps []domain.App
}

func (m *mockAppRepo) FindAll() ([]domain.App, error) {
	return m.apps, nil
}
func (m *mockAppRepo) FindByName(name string) (*domain.App, error) {
	for _, a := range m.apps {
		if a.Name == name {
			return &a, nil
		}
	}
	return nil, domain.ErrAppNotFound
}
func (m *mockAppRepo) Save(app *domain.App) error {
	m.apps = append(m.apps, *app)
	return nil
}
func (m *mockAppRepo) Delete(name string) error {
	var updated []domain.App
	for _, a := range m.apps {
		if a.Name != name {
			updated = append(updated, a)
		}
	}
	m.apps = updated
	return nil
}
func (m *mockAppRepo) Exists(name string) (bool, error) {
	_, err := m.FindByName(name)
	return err == nil, nil
}

// Mock ConfigRepository
type mockConfigRepo struct {
	cfg *domain.SystemConfig
}

func (m *mockConfigRepo) Get() (*domain.SystemConfig, error) {
	return m.cfg, nil
}
func (m *mockConfigRepo) Save(cfg *domain.SystemConfig) error {
	m.cfg = cfg
	return nil
}
func (m *mockConfigRepo) GetValue(key string, def string) (string, error) {
	return def, nil
}
func (m *mockConfigRepo) SaveValue(key string, val string) error {
	return nil
}

// Mock SystemManager
type mockSystemManager struct {
	root     bool
	users    map[string]bool
	services map[string]bool
}

func (m *mockSystemManager) IsRoot() bool {
	return m.root
}
func (m *mockSystemManager) UserExists(username string) bool {
	return m.users[username]
}
func (m *mockSystemManager) CreateUser(username, password, homeDir, shellPath string) error {
	m.users[username] = true
	return nil
}
func (m *mockSystemManager) DeleteUser(username string, removeHome bool) error {
	delete(m.users, username)
	return nil
}
func (m *mockSystemManager) EnableLinger(username string) error {
	return nil
}
func (m *mockSystemManager) DisableLinger(username string) error {
	return nil
}
func (m *mockSystemManager) KillUserProcesses(username string) error {
	return nil
}
func (m *mockSystemManager) SetPermissions(path string, username string, mode uint32) error {
	return nil
}
func (m *mockSystemManager) RunSystemctlUser(username string, action string, serviceName string) (string, error) {
	if action == "start" || action == "restart" {
		m.services[serviceName] = true
	} else if action == "stop" || action == "disable" {
		m.services[serviceName] = false
	}
	return "ok", nil
}
func (m *mockSystemManager) IsServiceActive(username string, serviceName string) bool {
	return m.services[serviceName]
}
func (m *mockSystemManager) GetJournalLogs(username string, serviceName string, lines int) (string, error) {
	return "sample logs", nil
}
func (m *mockSystemManager) GetFishOrBashShell() string {
	return "/bin/bash"
}

// Mock DatabaseManager
type mockDBManager struct{}

func (m *mockDBManager) VerifyRootConnection(rootUser, rootPassword string) error {
	return nil
}
func (m *mockDBManager) CreateDatabaseAndUser(rootUser, rootPassword, dbName, dbUser, dbPassword string) error {
	return nil
}
func (m *mockDBManager) DropDatabaseAndUser(rootUser, rootPassword, dbName, dbUser string) error {
	return nil
}
func (m *mockDBManager) IsDatabaseClientAvailable() bool {
	return true
}

// Mock WebServerConfigGenerator
type mockWebGen struct{}

func (m *mockWebGen) GenerateLaravelCaddyfile(homeDir, portFE, fastcgiTarget string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeFullstackCaddyfile(homeDir, portFE, portBE string) error {
	return nil
}
func (m *mockWebGen) GenerateCaddySystemdService(systemdDir, appName string) error {
	return nil
}
func (m *mockWebGen) GenerateLaravelNginxConfig(homeDir, portFE, fastcgiTarget string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeFullstackNginxConfig(homeDir, portFE, portBE string) error {
	return nil
}
func (m *mockWebGen) GenerateNginxSystemdService(systemdDir, appName string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeDirectRunScript(homeDir, portSingle string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeDirectSystemdService(systemdDir, appName, portSingle string) error {
	return nil
}
func (m *mockWebGen) CreatePlaceholders(homeDir string, stack domain.StackType, appName string, portBE string) error {
	return nil
}
func (m *mockWebGen) GetCaddyPath() string {
	return "/usr/bin/caddy"
}
func (m *mockWebGen) GetNginxPath() string {
	return "/usr/sbin/nginx"
}

// Mock GitManager
type mockGitManager struct{}

func (m *mockGitManager) Clone(repoURL, targetDir string) error {
	return nil
}

func TestAppUsecase(t *testing.T) {
	appRepo := &mockAppRepo{}
	configRepo := &mockConfigRepo{cfg: &domain.SystemConfig{PhpSockPath: "/run/php/php8.3-fpm.sock"}}
	sysMgr := &mockSystemManager{
		root:     true,
		users:    make(map[string]bool),
		services: make(map[string]bool),
	}
	dbMgr := &mockDBManager{}
	webGen := &mockWebGen{}
	gitMgr := &mockGitManager{}

	uc := usecase.NewAppUsecase(appRepo, configRepo, sysMgr, dbMgr, webGen, gitMgr, "/tmp/test_apps")

	// 1. Create App
	dto := &domain.CreateAppDTO{
		Name:         "app1",
		UserPassword: "password123",
		Stack:        domain.StackLaravel,
		PortFE:       "8000",
		PhpFpmMode:   domain.PhpModeSocket,
		WebServer:    domain.WebServerCaddy,
		DBName:       "app1_db",
		DBUser:       "app1_u",
		DBPassword:   "dbpass123",
	}

	app, err := uc.Create(dto)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if app.Name != "app1" {
		t.Errorf("expected app name app1, got %s", app.Name)
	}

	// 2. List Apps
	list, err := uc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 app in list, got %d", len(list))
	}
	if list[0].Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", list[0].Status)
	}

	// 3. Delete App
	delDto := &domain.DeleteAppDTO{
		Name: "app1",
	}
	if err := uc.Delete(delDto); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	listAfter, _ := uc.List()
	if len(listAfter) != 0 {
		t.Errorf("expected 0 apps after delete, got %d", len(listAfter))
	}
}
