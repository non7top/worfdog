package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"worfdog/config"
	"worfdog/plugins"
	"worfdog/reboot"
)

// Version is set at build time via ldflags
var Version = "dev"

// service pairs a monitored plugin with its config and recovery counters
type service struct {
	plugin   plugins.Plugin
	cfg      config.ServiceConfig
	failures int // consecutive failed checks
	restarts int // restart attempts since the last recovery
}

// maxRetries is the number of consecutive failed checks before recovery starts
func (s *service) maxRetries() int {
	if s.cfg.MaxRetries > 0 {
		return s.cfg.MaxRetries
	}
	return 1
}

// Watchdog is the main service that monitors and manages services
type Watchdog struct {
	cfg           *config.Config
	services      []*service
	rebootTracker *reboot.Tracker
	interval      time.Duration
	dryRun        bool // if true, only log actions without executing
	logger        *log.Logger
}

// NewWatchdog creates a new watchdog instance
func NewWatchdog(cfg *config.Config, interval time.Duration, dryRun bool) *Watchdog {
	w := &Watchdog{
		cfg:      cfg,
		interval: interval,
		dryRun:   dryRun,
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
		case "mysql":
			p = plugins.NewMySQLPlugin(svcCfg)
		default:
			w.logger.Printf("WARNING: Unknown service type '%s' for %s, skipping", svcCfg.Type, svcCfg.Name)
			continue
		}

		w.services = append(w.services, &service{plugin: p, cfg: svcCfg})
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

	w.logger.Printf("Starting watchdog with %d plugins, check interval: %v", len(w.services), w.interval)
	w.logger.Println(w.rebootTracker.Status())

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run initial check
	w.checkAll()

	for range ticker.C {
		w.checkAll()
	}
}

// checkAll runs health checks on all services concurrently, then handles the results in order
func (w *Watchdog) checkAll() {
	results := make([]plugins.CheckResult, len(w.services))

	var wg sync.WaitGroup
	for i, svc := range w.services {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = svc.plugin.Check()
		}()
	}
	wg.Wait()

	for i, svc := range w.services {
		w.handleResult(svc, results[i])
	}
}

// handleResult processes a check result and takes appropriate action
func (w *Watchdog) handleResult(svc *service, result plugins.CheckResult) {
	switch result.Status {
	case plugins.StatusOK, plugins.StatusWarning:
		w.logger.Printf("[%s] %s: %s", result.Service, result.Status, result.Message)
		svc.failures = 0
	case plugins.StatusCritical:
		svc.failures++
		maxRetries := svc.maxRetries()

		if svc.failures >= maxRetries {
			w.logger.Printf("[%s] %s: %s (failure %d/%d) - attempting recovery", result.Service, result.Status, result.Message, svc.failures, maxRetries)
			w.attemptRecovery(svc)
		} else {
			w.logger.Printf("[%s] %s: %s (failure %d/%d)", result.Service, result.Status, result.Message, svc.failures, maxRetries)
		}
	case plugins.StatusUnknown:
		w.logger.Printf("[%s] %s: %s", result.Service, result.Status, result.Message)
	}
}

// maxRestarts is the service-specific restart limit, or the global default
func (w *Watchdog) maxRestarts(svc *service) int {
	if svc.cfg.MaxRestarts > 0 {
		return svc.cfg.MaxRestarts
	}
	return w.cfg.Reboot.MaxRestarts
}

// attemptRecovery tries to recover a failed service
func (w *Watchdog) attemptRecovery(svc *service) {
	name := svc.cfg.Name
	svc.restarts++
	maxRestarts := w.maxRestarts(svc)

	// Check if we've exceeded max restarts
	if w.cfg.Reboot.Enabled && svc.restarts > maxRestarts {
		w.logger.Printf("Service %s exceeded max restarts (%d), considering reboot", name, maxRestarts)
		if w.attemptReboot(name) {
			return
		}
		w.logger.Printf("Reboot not possible, continuing to restart %s", name)
	}

	restartCmd := svc.plugin.GetConfig().RestartCmd
	if restartCmd == "" {
		w.logger.Printf("Service %s has no restart command configured, considering reboot", name)
		if w.cfg.Reboot.Enabled {
			w.attemptReboot(name)
		}
		return
	}

	w.logger.Printf("Attempting to restart service: %s (attempt %d/%d) using: %s", name, svc.restarts, maxRestarts, restartCmd)
	if w.dryRun {
		w.logger.Printf("[DRY RUN] Would restart %s using: %s", name, restartCmd)
		return
	}
	if err := svc.plugin.Restart(); err != nil {
		w.logger.Printf("Failed to restart %s: %v", name, err)
		if w.cfg.Reboot.Enabled {
			w.attemptReboot(name)
		}
		return
	}

	w.logger.Printf("Successfully restarted %s", name)
	w.verifyRecovery(svc)
}

// verifyRecovery re-checks a restarted service and resets its counters on success
func (w *Watchdog) verifyRecovery(svc *service) {
	name := svc.cfg.Name

	time.Sleep(5 * time.Second)
	result := svc.plugin.Check()
	if result.Status == plugins.StatusOK {
		w.logger.Printf("Service %s recovered successfully", name)
		svc.restarts = 0
		svc.failures = 0
		return
	}

	w.logger.Printf("Service %s still unhealthy after restart: %s", name, result.Message)
	if w.cfg.Reboot.Enabled {
		w.attemptReboot(name)
	}
}

// attemptReboot tries to reboot the system if allowed and reports whether a reboot was initiated
func (w *Watchdog) attemptReboot(serviceName string) bool {
	w.logger.Printf("Considering system reboot due to persistent failure of %s", serviceName)

	if allowed, reason := w.rebootTracker.CanReboot(); !allowed {
		w.logger.Printf("Reboot blocked: %s", reason)
		return false
	}

	if w.dryRun {
		w.logger.Printf("[DRY RUN] Would reboot system")
		return true
	}

	w.logger.Printf("Initiating system reboot...")
	if err := w.rebootTracker.Reboot(); err != nil {
		w.logger.Printf("Reboot failed: %v", err)
		return false
	}
	// Note: On success, the system will reboot and this process will terminate
	return true
}

// GetRebootTracker returns the reboot tracker for external access
func (w *Watchdog) GetRebootTracker() *reboot.Tracker {
	return w.rebootTracker
}

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	interval := flag.Duration("interval", 0, "Health check interval (default: from config or 30s)")
	initialDelay := flag.Duration("initial-delay", 0, "Initial delay before first check (default: from config or same as interval)")
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

	// Log configuration warnings
	if cfg.HasWarnings() {
		for _, msg := range cfg.GetWarnings() {
			fmt.Printf("[worfdog] WARNING: %s\n", msg)
		}
	}

	// Use config values if not overridden by flags
	intervalValue := *interval
	if intervalValue == 0 {
		intervalValue = time.Duration(cfg.Worfdog.Interval) * time.Second
	}

	initialDelayValue := *initialDelay
	if initialDelayValue == 0 {
		initialDelayValue = time.Duration(cfg.Worfdog.InitialDelay) * time.Second
	}

	dryRunValue := *dryRun || cfg.Worfdog.DryRun

	// Create watchdog
	watchdog := NewWatchdog(cfg, intervalValue, dryRunValue)

	// Handle status request
	if *showStatus {
		fmt.Println("Worfdog Status")
		fmt.Println("==============")
		fmt.Printf("Plugins: %d\n", len(watchdog.services))
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

	// Handle initial delay
	if initialDelayValue > 0 {
		watchdog.logger.Printf("Waiting %v before first check...", initialDelayValue)
		time.Sleep(initialDelayValue)
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
