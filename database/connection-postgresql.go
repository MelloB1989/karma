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
	ssl := env.DatabaseSSLMode
	var driverName string
	var driverSource string

	// Parse the URL
	parsedURL, err := url.Parse(env.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Extract user info
	userInfo := parsedURL.User
	username := userInfo.Username()
	password, _ := userInfo.Password()

	// Extract host and port
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "5432" // Default PostgreSQL port
	}

	// Extract database name
	pathSegments := strings.Split(parsedURL.Path, "/")
	databaseName := pathSegments[len(pathSegments)-1]

	// Extract SSL mode
	queryParams := parsedURL.Query()
	sslMode := queryParams.Get("sslmode")
	if sslMode == "require" {
		sslMode = "true"
	} else {
		sslMode = "false"
	}
	driverName = "postgres"
	if ssl == "true" {
		driverSource = fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=require", username, databaseName, password, host, port)
	} else {
		driverSource = fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=disable", username, databaseName, password, host, port)
	}

	db, err := sqlx.Connect(driverName, driverSource)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
		return nil, err
	} else {
		log.Println("Successfully Connected")
		return db, nil
	}
}
