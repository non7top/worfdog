package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"worfdog/config"
	"worfdog/plugins"
	"worfdog/reboot"
)

// Version is set at build time via ldflags
var Version = "dev"

// Watchdog is the main service that monitors and manages services
type Watchdog struct {
	cfg        *config.Config
	plugins    []plugins.Plugin
	rebootTracker *reboot.Tracker
	interval   time.Duration
	logger     *log.Logger
}

// NewWatchdog creates a new watchdog instance
func NewWatchdog(cfg *config.Config, interval time.Duration) *Watchdog {
	w := &Watchdog{
		cfg:      cfg,
		plugins:  []plugins.Plugin{},
		interval: interval,
		logger:   log.New(os.Stdout, "[worfdog] ", log.LstdFlags),
	}

	// Initialize reboot tracker
	w.rebootTracker = reboot.NewTracker(
		cfg.Reboot.MaxReboots,
		cfg.Reboot.WindowHours,
		cfg.Reboot.SudoPassword,
		"",
	)

	// Initialize plugins based on configuration
	for _, svcCfg := range cfg.Services {
		var p plugins.Plugin

		switch svcCfg.Type {
		case "systemd":
			p = plugins.NewSystemdPlugin(svcCfg)
		case "https", "http":
			p = plugins.NewHTTPSPlugin(svcCfg)
		default:
			w.logger.Printf("WARNING: Unknown service type '%s' for %s, skipping", svcCfg.Type, svcCfg.Name)
			continue
		}

		w.plugins = append(w.plugins, p)
		w.logger.Printf("Registered plugin: %s (type: %s)", svcCfg.Name, svcCfg.Type)
	}

	return w
}

// Run starts the watchdog monitoring loop
func (w *Watchdog) Run() {
	w.logger.Printf("Starting watchdog with %d plugins, check interval: %v", len(w.plugins), w.interval)
	w.logger.Println(w.rebootTracker.Status())

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run initial check
	w.checkAll()

	for range ticker.C {
		w.checkAll()
	}
}

// checkAll runs health checks on all plugins
func (w *Watchdog) checkAll() {
	for _, p := range w.plugins {
		result := p.Check()
		w.handleResult(result)
	}
}

// handleResult processes a check result and takes appropriate action
func (w *Watchdog) handleResult(result plugins.CheckResult) {
	switch result.Status {
	case plugins.StatusOK:
		w.logger.Printf("[%s] %s: %s", result.Service, result.Status, result.Message)
	case plugins.StatusWarning:
		w.logger.Printf("[%s] %s: %s", result.Service, result.Status, result.Message)
	case plugins.StatusCritical:
		w.logger.Printf("[%s] %s: %s - attempting recovery", result.Service, result.Status, result.Message)
		w.attemptRecovery(result.Service)
	case plugins.StatusUnknown:
		w.logger.Printf("[%s] %s: %s", result.Service, result.Status, result.Message)
	}
}

// attemptRecovery tries to recover a failed service
func (w *Watchdog) attemptRecovery(serviceName string) {
	// Find the plugin for this service
	var targetPlugin plugins.Plugin
	for _, p := range w.plugins {
		if p.Name() == serviceName {
			targetPlugin = p
			break
		}
	}

	if targetPlugin == nil {
		w.logger.Printf("Cannot recover %s: plugin not found", serviceName)
		return
	}

	// Try to restart the service
	w.logger.Printf("Attempting to restart service: %s", serviceName)
	if err := targetPlugin.Restart(); err != nil {
		w.logger.Printf("Failed to restart %s: %v", serviceName, err)

		// If restart failed and reboot is enabled, consider rebooting
		if w.cfg.Reboot.Enabled {
			w.attemptReboot(serviceName)
		}
		return
	}

	w.logger.Printf("Successfully restarted %s", serviceName)

	// Verify the service is now healthy
	time.Sleep(5 * time.Second)
	result := targetPlugin.Check()
	if result.Status == plugins.StatusOK {
		w.logger.Printf("Service %s recovered successfully", serviceName)
	} else {
		w.logger.Printf("Service %s still unhealthy after restart: %s", serviceName, result.Message)

		// Consider reboot if still failing
		if w.cfg.Reboot.Enabled {
			w.attemptReboot(serviceName)
		}
	}
}

// attemptReboot tries to reboot the system if allowed
func (w *Watchdog) attemptReboot(serviceName string) {
	w.logger.Printf("Considering system reboot due to persistent failure of %s", serviceName)

	if allowed, reason := w.rebootTracker.CanReboot(); !allowed {
		w.logger.Printf("Reboot blocked: %s", reason)
		return
	}

	w.logger.Printf("Initiating system reboot...")
	if err := w.rebootTracker.Reboot(); err != nil {
		w.logger.Printf("Reboot failed: %v", err)
	}
	// Note: On success, the system will reboot and this process will terminate
}

// GetRebootTracker returns the reboot tracker for external access
func (w *Watchdog) GetRebootTracker() *reboot.Tracker {
	return w.rebootTracker
}

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	interval := flag.Duration("interval", 30*time.Second, "Health check interval")
	showStatus := flag.Bool("status", false, "Show current status and exit")
	resetReboots := flag.Bool("reset-reboots", false, "Reset reboot counter")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// Handle version request
	if *showVersion {
		fmt.Printf("worfdog %s\n", Version)
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadDefault()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create watchdog
	watchdog := NewWatchdog(cfg, *interval)

	// Handle status request
	if *showStatus {
		fmt.Println("Warfdog Status")
		fmt.Println("==============")
		fmt.Printf("Plugins: %d\n", len(watchdog.plugins))
		fmt.Println(watchdog.rebootTracker.Status())
		os.Exit(0)
	}

	// Handle reset reboots request
	if *resetReboots {
		if err := watchdog.rebootTracker.Reset(); err != nil {
			fmt.Fprintf(os.Stderr, "Error resetting reboot counter: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Reboot counter reset successfully")
		os.Exit(0)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutting down watchdog...")
		os.Exit(0)
	}()

	// Run the watchdog
	watchdog.Run()
}
