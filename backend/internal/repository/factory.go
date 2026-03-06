//go:build !oracle

package repository

import (
	"os"
)

// NewFromEnv creates a repository based on DB_TYPE environment variable.
// This is the default build (SQLite only). Build with -tags oracle for Oracle support.
func NewFromEnv() (IRepository, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./jugan.db"
	}
	return New(dbPath)
}
