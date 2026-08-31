package config

import (
	"os"
)

type Config struct {
	RegistryDir  string
	RegistryFile string
	ConfigFile   string
	AppsBaseDir  string
	HTTPPort     string
}

func Load() *Config {
	registryDir := getEnvOrDefault("PROJECT_REGISTRY_DIR", "/etc/project-manager")
	registryFile := getEnvOrDefault("PROJECT_REGISTRY_FILE", registryDir+"/apps.json")
	configFile := getEnvOrDefault("PROJECT_CONFIG_FILE", registryDir+"/config.json")
	appsBaseDir := getEnvOrDefault("PROJECT_APPS_BASE_DIR", "/home/apps")
	httpPort := getEnvOrDefault("PROJECT_HTTP_PORT", ":8080")

	return &Config{
		RegistryDir:  registryDir,
		RegistryFile: registryFile,
		ConfigFile:   configFile,
		AppsBaseDir:  appsBaseDir,
		HTTPPort:     httpPort,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
