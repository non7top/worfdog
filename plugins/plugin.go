package plugins

import (
	"worfdog/config"
)

// PluginStatus represents the health status of a monitored service
type PluginStatus int

const (
	StatusOK PluginStatus = iota
	StatusWarning
	StatusCritical
	StatusUnknown
)

func (s PluginStatus) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusWarning:
		return "WARNING"
	case StatusCritical:
		return "CRITICAL"
	case StatusUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

// CheckResult holds the result of a plugin health check
type CheckResult struct {
	Status  PluginStatus
	Message string
	Service string
}

// Plugin defines the interface for all monitoring plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Check performs a health check and returns the result
	Check() CheckResult

	// Restart attempts to restart the service
	Restart() error

	// GetConfig returns the service configuration
	GetConfig() config.ServiceConfig
}
