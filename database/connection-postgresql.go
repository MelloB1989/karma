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

	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		log.Fatal(err)
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
	pathSegments := strings.Split(parsedURL.Path, "/")
	databaseName := pathSegments[len(pathSegments)-1]
	sslMode := parsedURL.Query().Get("sslmode")

	connStr := fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=%s",
		username, databaseName, password, host, port, sslMode)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Set connection pool settings
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
			db.SetConnMaxLifetime(time.Duration(*opt.ConnMaxLifetime) * time.Second)
		}
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully Connected")
	return db, nil
}
