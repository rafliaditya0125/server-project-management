package domain

// AppRepository defines persistence operations for application metadata.
type AppRepository interface {
	FindAll() ([]App, error)
	FindByName(name string) (*App, error)
	Save(app *App) error
	Delete(name string) error
	Exists(name string) (bool, error)
}

// ConfigRepository defines persistence operations for global project manager configuration.
type ConfigRepository interface {
	Get() (*SystemConfig, error)
	Save(config *SystemConfig) error
	GetValue(key string, defaultVal string) (string, error)
	SaveValue(key string, value string) error
}
