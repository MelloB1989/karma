package database

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/MelloB1989/karma/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func PostgresConn() (*sqlx.DB, error) {
	env := config.DefaultConfig()

	// Choose URL based on environment
	var dbURL string
	if env.Environment == "" {
		dbURL = env.DatabaseURL
	} else {
		dbURL, _ = config.GetEnv(env.Environment + "_DATABASE_URL")
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

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Successfully Connected")
	return db, nil
}
