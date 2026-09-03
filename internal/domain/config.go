package domain

// SystemConfig represents global project manager configuration stored in /etc/project-manager/config.json.
type SystemConfig struct {
	PhpMode     PhpMode `json:"php_mode"`
	PhpSockPath string  `json:"php_sock_path"`
	PhpPort     string  `json:"php_port"`
	PhpService  string  `json:"php_service"`
	OS          string  `json:"os"`
	UpdatedAt   string  `json:"updated_at"`
}
