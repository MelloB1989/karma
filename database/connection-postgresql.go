package database

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresConnOptions struct {
	MaxOpenConns      *int
	MaxIdleConns      *int
	ConnMaxLifetime   *time.Duration
	ConnMaxIdleTime   *time.Duration
	DatabaseUrlPrefix string
}

func PostgresConn(options ...PostgresConnOptions) (*sqlx.DB, error) {
	env := config.DefaultConfig()

	// Choose URL based on environment
	var dbURL string
	prefix := ""
	if len(options) > 0 && options[0].DatabaseUrlPrefix != "" {
		prefix = "_" + options[0].DatabaseUrlPrefix
	}

	if env.Environment == "" {
		dbURL = config.GetEnvRaw(strings.TrimPrefix(prefix, "_") + "_DATABASE_URL")
		if dbURL == "" {
			dbURL = env.DatabaseURL
		}
	} else {
		dbURL = config.GetEnvRaw(env.Environment + prefix + "_DATABASE_URL")
	}

	if dbURL == "" {
		log.Fatal("DATABASE_URL is empty or not set")
	}

	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse database URL: %v", err)
	}

	// Extract credentials and connection components
	userInfo := parsedURL.User
	username := userInfo.Username()
	password, _ := userInfo.Password()
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "5432"
	}

	// The database name is the last segment in the path
	pathSegments := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/")
	databaseName := ""
	if len(pathSegments) > 0 && pathSegments[len(pathSegments)-1] != "" {
		databaseName = pathSegments[len(pathSegments)-1]
	}

	if databaseName == "" {
		log.Fatal("Database name is empty in the URL")
	}

	sslMode := parsedURL.Query().Get("sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}

	connStr := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=%s",
		username, databaseName, password, host, port, sslMode)

	// Log connection attempt (without password for security)
	log.Printf("Attempting to connect to PostgreSQL: host=%s port=%s dbname=%s user=%s sslmode=%s",
		host, port, databaseName, username, sslMode)

	var db *sqlx.DB
	maxRetries := 3
	retryDelay := 2 * time.Second

	// Retry logic
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Connection attempt %d/%d...", attempt, maxRetries)

		db, err = sqlx.Connect("postgres", connStr)
		if err == nil {
			// Connection successful, try to ping
			if err = db.Ping(); err == nil {
				log.Println("Successfully connected to PostgreSQL")
				break
			}
			// Ping failed, close the connection and retry
			db.Close()
			log.Printf("Ping failed on attempt %d: %v", attempt, err)
		} else {
			log.Printf("Connection failed on attempt %d: %v", attempt, err)
		}

		if attempt < maxRetries {
			log.Printf("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	// If all retries failed, panic
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to PostgreSQL after %d attempts. Last error: %v\nConnection string (sanitized): user=%s dbname=%s host=%s port=%s sslmode=%s",
			maxRetries, err, username, databaseName, host, port, sslMode)
	}

	// Set connection pool settings with defaults
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0) // No limit on connection lifetime

	if len(options) > 0 {
		opt := options[0]
		if opt.MaxOpenConns != nil {
			db.SetMaxOpenConns(*opt.MaxOpenConns)
		}
		if opt.MaxIdleConns != nil {
			db.SetMaxIdleConns(*opt.MaxIdleConns)
		}
		if opt.ConnMaxLifetime != nil {
			db.SetConnMaxLifetime(*opt.ConnMaxLifetime)
		}
		if opt.ConnMaxIdleTime != nil {
			db.SetConnMaxIdleTime(*opt.ConnMaxIdleTime)
		}
	}

	log.Printf("Connection pool configured: MaxOpenConns=%d, MaxIdleConns=%d, ConnMaxLifetime=%v, ConnMaxIdleTime=%v",
		db.Stats().MaxOpenConnections, db.Stats().Idle, db.Stats().MaxLifetimeClosed, db.Stats().MaxIdleTimeClosed)

	return db, nil
}
