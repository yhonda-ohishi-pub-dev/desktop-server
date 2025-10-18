package server

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
)

type DatabaseConnection struct {
	DB     *sql.DB
	Driver string
}

func NewDatabaseConnection() (*DatabaseConnection, error) {
	// Get database configuration from environment variables
	driver := os.Getenv("DB_DRIVER") // "sqlserver" or "mysql"
	if driver == "" {
		driver = "sqlserver" // default
	}

	var dsn string
	switch driver {
	case "sqlserver":
		server := os.Getenv("DB_SERVER")
		if server == "" {
			server = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "1433"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "sa"
		}
		password := os.Getenv("DB_PASSWORD")
		database := os.Getenv("DB_NAME")
		if database == "" {
			database = "master"
		}

		dsn = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s",
			server, user, password, port, database)

	case "mysql":
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "3306"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "root"
		}
		password := os.Getenv("DB_PASSWORD")
		database := os.Getenv("DB_NAME")
		if database == "" {
			database = "mysql"
		}

		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			user, password, host, port, database)

	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseConnection{
		DB:     db,
		Driver: driver,
	}, nil
}

func (dc *DatabaseConnection) Close() error {
	if dc.DB != nil {
		return dc.DB.Close()
	}
	return nil
}

func (dc *DatabaseConnection) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return dc.DB.Query(query, args...)
}

func (dc *DatabaseConnection) Exec(query string, args ...interface{}) (sql.Result, error) {
	return dc.DB.Exec(query, args...)
}

func (dc *DatabaseConnection) GetTables() ([]string, error) {
	var query string
	switch dc.Driver {
	case "sqlserver":
		query = "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE'"
	case "mysql":
		query = "SHOW TABLES"
	default:
		return nil, fmt.Errorf("unsupported driver: %s", dc.Driver)
	}

	rows, err := dc.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}
