package domain

// OSDetector detects Linux distribution.
type OSDetector interface {
	DetectOS() (string, error)
}

// SystemManager handles Linux OS user creation, deletion, linger, process killing, file permissions, and systemd user services.
type SystemManager interface {
	IsRoot() bool
	UserExists(username string) bool
	CreateUser(username, password, homeDir, shellPath string) error
	DeleteUser(username string, removeHome bool) error
	EnableLinger(username string) error
	DisableLinger(username string) error
	KillUserProcesses(username string) error
	SetPermissions(path string, username string, mode uint32) error
	RunSystemctlUser(username string, action string, serviceName string) (string, error)
	IsServiceActive(username string, serviceName string) bool
	GetJournalLogs(username string, serviceName string, lines int) (string, error)
	GetFishOrBashShell() string
}

// DatabaseManager handles MySQL/MariaDB database verification, database creation, user creation, and grants.
type DatabaseManager interface {
	VerifyRootConnection(rootUser, rootPassword string) error
	CreateDatabaseAndUser(rootUser, rootPassword, dbName, dbUser, dbPassword string) error
	DropDatabaseAndUser(rootUser, rootPassword, dbName, dbUser string) error
	IsDatabaseClientAvailable() bool
}

// WebServerConfigGenerator generates configuration files and systemd unit files for web servers and run scripts.
type WebServerConfigGenerator interface {
	GenerateLaravelCaddyfile(homeDir, portFE, fastcgiTarget string) error
	GenerateNodeFullstackCaddyfile(homeDir, portFE, portBE string) error
	GenerateCaddySystemdService(systemdDir, appName string) error

	GenerateLaravelNginxConfig(homeDir, portFE, fastcgiTarget string) error
	GenerateNodeFullstackNginxConfig(homeDir, portFE, portBE string) error
	GenerateNginxSystemdService(systemdDir, appName string) error

	GenerateNodeDirectRunScript(homeDir, portSingle string) error
	GenerateNodeDirectSystemdService(systemdDir, appName, portSingle string) error

	CreatePlaceholders(homeDir string, stack StackType, appName string, portBE string) error
	GetCaddyPath() string
	GetNginxPath() string
}

// PackageManager handles multi-OS package installation and PHP configuration.
type PackageManager interface {
	InstallPHPAndComposer(osDistro string) error
	InstallNodeAndNPM(osDistro string) error
	InstallWebServers(osDistro string) error
	InstallShellAndDBClient(osDistro string) error
	DetectPhpFpmSocket() string
	DetectPhpFpmService() string
	EnableAndStartPhpService(serviceName string) error
}

// ShellCompletionManager manages generation and installation of autocompletion scripts for Bash, Zsh, and Fish.
type ShellCompletionManager interface {
	InstallShellCompletions() error
}

// SymlinkManager manages creating global symlink at /usr/local/bin/project.
type SymlinkManager interface {
	CreateGlobalSymlink(sourceBinaryPath string) error
}

// GitManager clones repositories into isolated home directories.
type GitManager interface {
	Clone(repoURL, targetDir string) error
}
