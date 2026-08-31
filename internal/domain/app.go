package domain

import "time"

// StackType represents the application stack type.
type StackType string

const (
	StackLaravel       StackType = "laravel"
	StackNodeFullstack StackType = "node-fullstack"
	StackNodeAPI       StackType = "node-api"
)

// WebServerType represents the web server chosen for an application.
type WebServerType string

const (
	WebServerCaddy WebServerType = "caddy"
	WebServerNginx WebServerType = "nginx"
	WebServerNone  WebServerType = "None"
)

// PhpMode represents FastCGI connection mode (socket or port).
type PhpMode string

const (
	PhpModeSocket PhpMode = "socket"
	PhpModePort   PhpMode = "port"
)

// App represents an isolated multi-tenant application metadata entity.
type App struct {
	Name       string    `json:"name"`
	User       string    `json:"user"`
	Home       string    `json:"home"`
	Stack      StackType `json:"stack"`
	StackName  string    `json:"stack_name"`
	WebServer  string    `json:"webserver"`
	PortFE     string    `json:"port_fe"`
	PortBE     string    `json:"port_be"`
	Port       string    `json:"port"`
	PhpMode    PhpMode   `json:"php_mode"`
	DBName     string    `json:"db_name"`
	DBUser     string    `json:"db_user"`
	GitRepo    string    `json:"git_repo"`
	CreatedAt  string    `json:"created_at"`
}

// AppStatusInfo represents dynamic runtime status of an application.
type AppStatusInfo struct {
	App
	Status        string `json:"status"` // ACTIVE, INACTIVE, UNKNOWN
	ServiceActive bool   `json:"service_active"`
	DisplayPort   string `json:"display_port"`
}

// CreateAppDTO contains the payload required to create a new isolated application.
type CreateAppDTO struct {
	Name            string        `json:"name"`
	UserPassword    string        `json:"user_password"`
	Stack           StackType     `json:"stack"`
	GitRepo         string        `json:"git_repo"`
	PortFE          string        `json:"port_fe"`
	PortBE          string        `json:"port_be"`
	PortSingle      string        `json:"port_single"`
	PhpFpmMode      PhpMode       `json:"php_fpm_mode"`
	PhpSockPath     string        `json:"php_sock_path"`
	PhpTcpPort      string        `json:"php_tcp_port"`
	WebServer       WebServerType `json:"webserver"`
	DBName          string        `json:"db_name"`
	DBUser          string        `json:"db_user"`
	DBPassword      string        `json:"db_password"`
	DBRootUser      string        `json:"db_root_user"`
	DBRootPassword  string        `json:"db_root_password"`
}

// DeleteAppDTO contains the payload required to delete an application.
type DeleteAppDTO struct {
	Name           string `json:"name"`
	DBRootUser     string `json:"db_root_user"`
	DBRootPassword string `json:"db_root_password"`
	Force          bool   `json:"force"`
}

// ServiceAction represents an operation on a systemd service.
type ServiceAction string

const (
	ActionStart   ServiceAction = "start"
	ActionStop    ServiceAction = "stop"
	ActionRestart ServiceAction = "restart"
	ActionStatus  ServiceAction = "status"
)

// SetupOptions represents specific stages to execute during system setup.
type SetupOptions struct {
	All         bool     `json:"all"`
	PHP         bool     `json:"php"`
	Node        bool     `json:"node"`
	Web         bool     `json:"web"`
	DB          bool     `json:"db"`
	FastCGI     bool     `json:"fastcgi"`
	Symlink     bool     `json:"symlink"`
	Completion  bool     `json:"completion"`
	Interactive bool     `json:"interactive"`
	Except      []string `json:"except"`

	// Optional FastCGI interactive overrides
	ChosenPhpMode     PhpMode `json:"chosen_php_mode,omitempty"`
	ChosenPhpSockPath string  `json:"chosen_php_sock_path,omitempty"`
	ChosenPhpPort     string  `json:"chosen_php_port,omitempty"`
}

// NowUTCFormatted returns the current timestamp in ISO-8601 UTC.
func NowUTCFormatted() string {
	return time.Now().UTC().Format(time.RFC3339)
}
