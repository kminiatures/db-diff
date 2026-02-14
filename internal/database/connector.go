package database

import (
	"fmt"
	"os"

	"github.com/koba/db-diff/internal/schema"
)

// Config holds database connection configuration
type Config struct {
	Type     string // "mysql" or "postgres"
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// Database interface defines operations for database connections
type Database interface {
	Connect() error
	Close() error
	GetAllTables() ([]string, error)
	GetTableSchema(tableName string) (*schema.TableSchema, error)
	GetTableData(tableName string, limit int) ([]schema.Row, error)
}

// NewDatabase creates a new database connection based on type
func NewDatabase(config Config) (Database, error) {
	switch config.Type {
	case "mysql", "MySQL":
		return NewMySQL(config), nil
	case "postgres", "Postgres", "PostgreSQL":
		return NewPostgres(config), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// LoadConfigFromEnv loads database configuration from environment variables
func LoadConfigFromEnv() (Config, error) {
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		return Config{}, fmt.Errorf("DB_TYPE environment variable is required")
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	database := os.Getenv("DB_NAME")
	if database == "" {
		return Config{}, fmt.Errorf("DB_NAME environment variable is required")
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	port := os.Getenv("DB_PORT")
	if port == "" {
		if dbType == "mysql" || dbType == "MySQL" {
			port = "3306"
		} else if dbType == "postgres" || dbType == "Postgres" || dbType == "PostgreSQL" {
			port = "5432"
		}
	}

	return Config{
		Type:     dbType,
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
	}, nil
}
