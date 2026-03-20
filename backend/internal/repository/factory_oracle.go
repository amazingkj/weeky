//go:build oracle

package repository

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

// NewFromEnv creates a repository based on DB_TYPE environment variable.
// DB_TYPE=oracle uses Oracle, anything else defaults to SQLite.
func NewFromEnv() (IRepository, error) {
	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case "oracle":
		dsn := os.Getenv("ORACLE_DSN")
		if dsn == "" {
			return nil, fmt.Errorf("ORACLE_DSN is required when DB_TYPE=oracle")
		}
		oraURL, err := convertDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_DSN: %w", err)
		}
		return NewOracle(oraURL)
	default:
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "./jugan.db"
		}
		return New(dbPath)
	}
}

// convertDSN parses godror-style DSN (user/password@host:port/service)
// and converts it to go-ora URL format.
// Password may contain special chars like # and @.
func convertDSN(dsn string) (string, error) {
	// user/password@host:port/service
	slashIdx := strings.Index(dsn, "/")
	if slashIdx < 0 {
		return "", fmt.Errorf("missing / in DSN")
	}
	user := dsn[:slashIdx]
	rest := dsn[slashIdx+1:]

	// Last @ separates password from host (password may contain @)
	atIdx := strings.LastIndex(rest, "@")
	if atIdx < 0 {
		return "", fmt.Errorf("missing @ in DSN")
	}
	password := rest[:atIdx]
	hostPart := rest[atIdx+1:]

	// host:port/service
	colonIdx := strings.Index(hostPart, ":")
	if colonIdx < 0 {
		return "", fmt.Errorf("missing : in host part")
	}
	host := hostPart[:colonIdx]
	portService := hostPart[colonIdx+1:]

	slashIdx2 := strings.Index(portService, "/")
	if slashIdx2 < 0 {
		return "", fmt.Errorf("missing / in port/service")
	}
	portStr := portService[:slashIdx2]
	service := portService[slashIdx2+1:]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", fmt.Errorf("invalid port: %w", err)
	}

	return go_ora.BuildUrl(host, port, service, user, password, nil), nil
}
