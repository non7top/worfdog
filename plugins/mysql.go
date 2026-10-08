package plugins

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
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

// mysqlDSN builds the DSN through mysql.Config so credentials with special characters are escaped.
func mysqlDSN(cfg config.ServiceConfig) string {
	c := mysql.NewConfig()
	c.User = cfg.Username
	c.Passwd = cfg.Password
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	c.DBName = cfg.Database
	c.Timeout = time.Duration(cfg.Timeout) * time.Second
	return c.FormatDSN()
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

	// Open connection
	db, err := sql.Open("mysql", mysqlDSN(p.cfg))
	if err != nil {
		return CheckResult{
			Status:  StatusCritical,
			Message: fmt.Sprintf("Failed to open connection: %v", err),
			Service: p.cfg.Name,
		}
	}
	defer func() { _ = db.Close() }()

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
		return executeCommand(p.cfg.RestartCmd, restartTimeout)
	}
	return fmt.Errorf("no restart command configured for %s", p.cfg.Name)
}
