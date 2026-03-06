//go:build oracle

package repository

import (
	"fmt"
	"os"
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
		return NewOracle(dsn)
	default:
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "./jugan.db"
		}
		return New(dbPath)
	}
}
