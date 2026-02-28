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
	cfg           *config.Config
	plugins       []plugins.Plugin
	rebootTracker *reboot.Tracker
	restartCounts map[string]int // track restart attempts per service
	failureCounts map[string]int // track consecutive failures per service
	interval      time.Duration
	dryRun        bool // if true, only log actions without executing
	logger        *log.Logger
}

// NewWatchdog creates a new watchdog instance
func NewWatchdog(cfg *config.Config, interval time.Duration, dryRun bool) *Watchdog {
	w := &Watchdog{
		cfg:           cfg,
		plugins:       []plugins.Plugin{},
		restartCounts: make(map[string]int),
		failureCounts: make(map[string]int),
		interval:      interval,
		dryRun:        dryRun,
		logger:        log.New(os.Stdout, "[worfdog] ", log.LstdFlags),
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
	// Print version first
	w.logger.Printf("Version: %s", Version)

	// Dump reboot config
	w.logger.Printf("Reboot config: enabled=%v, max_restarts=%d, max_reboots=%d, window_hours=%d",
		w.cfg.Reboot.Enabled,
		w.cfg.Reboot.MaxRestarts,
		w.cfg.Reboot.MaxReboots,
		w.cfg.Reboot.WindowHours)

	// Dump service configs
	for _, svc := range w.cfg.Services {
		w.logger.Printf("Service [%s]: type=%s, timeout=%d, max_restarts=%d, max_retries=%d",
			svc.Name, svc.Type, svc.Timeout, svc.MaxRestarts, svc.MaxRetries)
	}

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
		// Reset failure count on success
		w.failureCounts[result.Service] = 0
	case plugins.StatusWarning:
		w.logger.Printf("[%s] %s: %s", result.Service, result.Status, result.Message)
		w.failureCounts[result.Service] = 0
	case plugins.StatusCritical:
		// Increment failure count
		w.failureCounts[result.Service]++
		failCount := w.failureCounts[result.Service]

		// Get max retries for this service
		maxRetries := 1
		for _, svc := range w.cfg.Services {
			if svc.Name == result.Service && svc.MaxRetries > 0 {
				maxRetries = svc.MaxRetries
				break
			}
		}

		if failCount >= maxRetries {
			w.logger.Printf("[%s] %s: %s (failure %d/%d) - attempting recovery", result.Service, result.Status, result.Message, failCount, maxRetries)
			w.attemptRecovery(result.Service)
		} else {
			w.logger.Printf("[%s] %s: %s (failure %d/%d)", result.Service, result.Status, result.Message, failCount, maxRetries)
		}
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

	// Increment restart count
	w.restartCounts[serviceName]++
	restartCount := w.restartCounts[serviceName]

	// Get max restarts for this service (service-specific or global default)
	maxRestarts := w.cfg.Reboot.MaxRestarts
	for _, svc := range w.cfg.Services {
		if svc.Name == serviceName && svc.MaxRestarts > 0 {
			maxRestarts = svc.MaxRestarts
			break
		}
	}

	// Check if we've exceeded max restarts
	if w.cfg.Reboot.Enabled && restartCount > maxRestarts {
		w.logger.Printf("Service %s exceeded max restarts (%d), considering reboot", serviceName, maxRestarts)
		w.attemptReboot(serviceName)
		return
	}

	// Get restart command for this service
	restartCmd := targetPlugin.GetConfig().RestartCmd
	if restartCmd == "" {
		w.logger.Printf("Service %s has no restart command configured, considering reboot", serviceName)
		if w.cfg.Reboot.Enabled {
			w.attemptReboot(serviceName)
		}
		return
	}

	// Try to restart the service
	w.logger.Printf("Attempting to restart service: %s (attempt %d/%d) using: %s", serviceName, restartCount, maxRestarts, restartCmd)
	if w.dryRun {
		w.logger.Printf("[DRY RUN] Would restart %s using: %s", serviceName, restartCmd)
		return
	}
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
		// Reset restart and failure counts on successful recovery
		w.restartCounts[serviceName] = 0
		w.failureCounts[serviceName] = 0
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

	if w.dryRun {
		w.logger.Printf("[DRY RUN] Would reboot system")
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
	dryRun := flag.Bool("dry_run", false, "Dry run: log actions without executing")
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
	watchdog := NewWatchdog(cfg, *interval, *dryRun)

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
