package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	deliveryHTTP "github.com/rafliaditya0125/server-project-management/internal/delivery/http"
	"github.com/rafliaditya0125/server-project-management/internal/delivery/http/handler"
	"github.com/rafliaditya0125/server-project-management/internal/domain"
	"github.com/rafliaditya0125/server-project-management/internal/usecase"
)

// Reusable mock implementations for HTTP testing
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
	return nil
}
func (m *mockAppRepo) Exists(name string) (bool, error) {
	return false, nil
}

type mockConfigRepo struct{}

func (m *mockConfigRepo) Get() (*domain.SystemConfig, error) {
	return &domain.SystemConfig{OS: "arch"}, nil
}
func (m *mockConfigRepo) Save(cfg *domain.SystemConfig) error {
	return nil
}
func (m *mockConfigRepo) GetValue(k, d string) (string, error) {
	return d, nil
}
func (m *mockConfigRepo) SaveValue(k, v string) error {
	return nil
}

type mockSystemManager struct{}

func (m *mockSystemManager) IsRoot() bool {
	return true
}
func (m *mockSystemManager) UserExists(u string) bool {
	return true
}
func (m *mockSystemManager) CreateUser(u, p, h, s string) error {
	return nil
}
func (m *mockSystemManager) DeleteUser(u string, r bool) error {
	return nil
}
func (m *mockSystemManager) EnableLinger(u string) error {
	return nil
}
func (m *mockSystemManager) DisableLinger(u string) error {
	return nil
}
func (m *mockSystemManager) KillUserProcesses(u string) error {
	return nil
}
func (m *mockSystemManager) SetPermissions(p string, u string, mode uint32) error {
	return nil
}
func (m *mockSystemManager) RunSystemctlUser(u, a, s string) (string, error) {
	return "active", nil
}
func (m *mockSystemManager) IsServiceActive(u, s string) bool {
	return true
}
func (m *mockSystemManager) GetJournalLogs(u, s string, l int) (string, error) {
	return "sample log", nil
}
func (m *mockSystemManager) GetFishOrBashShell() string {
	return "/bin/bash"
}

type mockDBManager struct{}

func (m *mockDBManager) VerifyRootConnection(u, p string) error {
	return nil
}
func (m *mockDBManager) CreateDatabaseAndUser(ru, rp, dn, du, dp string) error {
	return nil
}
func (m *mockDBManager) DropDatabaseAndUser(ru, rp, dn, du string) error {
	return nil
}
func (m *mockDBManager) IsDatabaseClientAvailable() bool {
	return true
}

type mockWebGen struct{}

func (m *mockWebGen) GenerateLaravelCaddyfile(h, p, f string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeFullstackCaddyfile(h, p, b string) error {
	return nil
}
func (m *mockWebGen) GenerateCaddySystemdService(s, a string) error {
	return nil
}
func (m *mockWebGen) GenerateLaravelNginxConfig(h, p, f string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeFullstackNginxConfig(h, p, b string) error {
	return nil
}
func (m *mockWebGen) GenerateNginxSystemdService(s, a string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeDirectRunScript(h, p string) error {
	return nil
}
func (m *mockWebGen) GenerateNodeDirectSystemdService(s, a, p string) error {
	return nil
}
func (m *mockWebGen) CreatePlaceholders(h string, s domain.StackType, a, p string) error {
	return nil
}
func (m *mockWebGen) GetCaddyPath() string {
	return "/usr/bin/caddy"
}
func (m *mockWebGen) GetNginxPath() string {
	return "/usr/sbin/nginx"
}

type mockGitManager struct{}

func (m *mockGitManager) Clone(r, t string) error {
	return nil
}

type mockOSDetector struct{}

func (m *mockOSDetector) DetectOS() (string, error) {
	return "arch", nil
}

type mockPkgInstaller struct{}

func (m *mockPkgInstaller) InstallPHPAndComposer(o string) error {
	return nil
}
func (m *mockPkgInstaller) InstallNodeAndNPM(o string) error {
	return nil
}
func (m *mockPkgInstaller) InstallWebServers(o string) error {
	return nil
}
func (m *mockPkgInstaller) InstallShellAndDBClient(o string) error {
	return nil
}
func (m *mockPkgInstaller) DetectPhpFpmSocket() string {
	return "/run/php/php-fpm.sock"
}
func (m *mockPkgInstaller) DetectPhpFpmService() string {
	return "php-fpm"
}
func (m *mockPkgInstaller) EnableAndStartPhpService(s string) error {
	return nil
}

type mockCompletionMgr struct{}

func (m *mockCompletionMgr) InstallShellCompletions() error {
	return nil
}

type mockSymlinkMgr struct{}

func (m *mockSymlinkMgr) CreateGlobalSymlink(s string) error {
	return nil
}

func TestHTTPRouter(t *testing.T) {
	appRepo := &mockAppRepo{
		apps: []domain.App{
			{Name: "testapp", User: "testapp", Stack: domain.StackLaravel},
		},
	}
	cfgRepo := &mockConfigRepo{}
	sysMgr := &mockSystemManager{}
	dbMgr := &mockDBManager{}
	webGen := &mockWebGen{}
	gitMgr := &mockGitManager{}
	osDet := &mockOSDetector{}
	pkgInst := &mockPkgInstaller{}
	compMgr := &mockCompletionMgr{}
	symMgr := &mockSymlinkMgr{}

	appUc := usecase.NewAppUsecase(appRepo, cfgRepo, sysMgr, dbMgr, webGen, gitMgr, "/tmp/apps")
	setupUc := usecase.NewSetupUsecase(cfgRepo, osDet, pkgInst, compMgr, symMgr, sysMgr)
	svcUc := usecase.NewServiceUsecase(sysMgr, appRepo)

	appHandler := handler.NewAppHandler(appUc, svcUc)
	setupHandler := handler.NewSetupHandler(setupUc)

	router := deliveryHTTP.NewRouter(appHandler, setupHandler)

	// 1. Health Check
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	router.ServeHTTP(recHealth, reqHealth)

	if recHealth.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", recHealth.Code)
	}

	// 2. List Apps
	reqApps := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
	recApps := httptest.NewRecorder()
	router.ServeHTTP(recApps, reqApps)

	if recApps.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/apps, got %d", recApps.Code)
	}

	var resp map[string]interface{}
	_ = json.NewDecoder(recApps.Body).Decode(&resp)
	if resp["success"] != true {
		t.Errorf("expected success true, got %v", resp["success"])
	}

	// 3. Setup Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	recStatus := httptest.NewRecorder()
	router.ServeHTTP(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/setup/status, got %d", recStatus.Code)
	}
}
