package domain

// AppUsecase defines application lifecycle business logic operations.
type AppUsecase interface {
	Create(dto *CreateAppDTO) (*App, error)
	Delete(dto *DeleteAppDTO) error
	List() ([]AppStatusInfo, error)
	GetLogs(appName string, lines int) (string, error)
	GetByName(name string) (*AppStatusInfo, error)
}

// SetupUsecase defines system dependency and environment setup business logic.
type SetupUsecase interface {
	Execute(opts *SetupOptions) error
	GetStatus() (*SystemConfig, string, error)
	ConfigureFastCGI(osDistro string, mode PhpMode, sockPath, port string) (*SystemConfig, error)
}

// ServiceUsecase defines systemd service control operations.
type ServiceUsecase interface {
	Manage(appName string, action ServiceAction) (string, error)
}
