package plugins

import (
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"worfdog/config"
)

func TestMySQLDSNRoundTripsSpecialCharacters(t *testing.T) {
	cfg := config.ServiceConfig{
		Host: "db.example", Port: 3307, Username: "wd@ops", Password: "p@ss/w:rd?x(1)", Database: "information_schema", Timeout: 5,
	}

	got, err := mysql.ParseDSN(mysqlDSN(cfg))
	if err != nil {
		t.Fatalf("DSN does not parse: %v", err)
	}
	if got.User != cfg.Username || got.Passwd != cfg.Password || got.Addr != "db.example:3307" ||
		got.DBName != cfg.Database || got.Timeout != 5*time.Second {
		t.Fatalf("DSN did not round-trip: %+v", got)
	}
}
