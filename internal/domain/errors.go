package domain

import "errors"

var (
	ErrPermissionDenied         = errors.New("root privileges required to execute this operation")
	ErrAppNotFound              = errors.New("application not found")
	ErrAppAlreadyExists         = errors.New("application already exists in registry")
	ErrUserAlreadyExists        = errors.New("system user already exists")
	ErrHomeDirAlreadyExists     = errors.New("home directory already exists")
	ErrInvalidAppName           = errors.New("invalid application name: must only contain lowercase letters, numbers, hyphens, and underscores")
	ErrInvalidPassword          = errors.New("password cannot be empty")
	ErrInvalidStack             = errors.New("invalid application stack selection")
	ErrInvalidPort              = errors.New("invalid port number: must be between 1 and 65535")
	ErrPortConflict             = errors.New("frontend and backend ports cannot be identical")
	ErrDatabaseConnectionFailed = errors.New("failed to connect to database with provided root credentials")
	ErrDatabaseCreationFailed   = errors.New("failed to create database or grant privileges")
	ErrDatabaseDropFailed       = errors.New("failed to drop database or user")
	ErrInvalidServiceAction     = errors.New("invalid service action: must be start, stop, restart, or status")
	ErrWebserverNotInstalled    = errors.New("required web server binary is not installed")
	ErrUnknownOS                = errors.New("operating system distribution is unsupported for automatic package installation")
)
