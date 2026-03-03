package plugins

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"worfdog/config"
)

// MySQLPlugin monitors MySQL database connectivity
type MySQLPlugin struct {
	cfg config.ServiceConfig
}

// NewMySQLPlugin creates a new MySQL monitoring plugin
func NewMySQLPlugin(cfg config.ServiceConfig) *MySQLPlugin {
	return &MySQLPlugin{
		cfg: cfg,
	}
}

func (p *MySQLPlugin) Name() string {
	return p.cfg.Name
}

func (p *MySQLPlugin) GetConfig() config.ServiceConfig {
	return p.cfg
}

func (p *MySQLPlugin) Check() CheckResult {
	if p.cfg.Host == "" {
		return CheckResult{
			Status:  StatusUnknown,
			Message: "No host configured",
			Service: p.cfg.Name,
		}
	}

	if p.cfg.Username == "" {
		return CheckResult{
			Status:  StatusUnknown,
			Message: "No username configured",
			Service: p.cfg.Name,
		}
	}

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%ds",
		p.cfg.Username,
		p.cfg.Password,
		p.cfg.Host,
		p.cfg.Port,
		p.cfg.Database,
		p.cfg.Timeout,
	)

	// Open connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return CheckResult{
			Status:  StatusCritical,
			Message: fmt.Sprintf("Failed to open connection: %v", err),
			Service: p.cfg.Name,
		}
	}
	defer db.Close()

	// Set connection timeout
	db.SetConnMaxLifetime(time.Duration(p.cfg.Timeout) * time.Second)

	// Ping the database
	if err := db.Ping(); err != nil {
		return CheckResult{
			Status:  StatusCritical,
			Message: fmt.Sprintf("Connection failed: %v", err),
			Service: p.cfg.Name,
		}
	}

	return CheckResult{
		Status:  StatusOK,
		Message: fmt.Sprintf("Connected to %s:%d", p.cfg.Host, p.cfg.Port),
		Service: p.cfg.Name,
	}
}

func (p *MySQLPlugin) Restart() error {
	if p.cfg.RestartCmd != "" {
		return executeCommand(p.cfg.RestartCmd)
	}
	return fmt.Errorf("no restart command configured for %s", p.cfg.Name)
}
