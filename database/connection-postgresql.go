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
	var driverName string
	var driverSource string

	// Parse the URL according to the environment
	environment := env.Environment
	var parsedURL *url.URL
	if environment == "" {
		p, err := url.Parse(env.DatabaseURL)
		if err != nil {
			log.Fatal(err)
		}
		parsedURL = p
	} else {
		dburl, _ := config.GetEnv(environment + "_" + "DATABASE_URL")
		p, err := url.Parse(dburl)
		if err != nil {
			log.Fatal(err)
		}
		parsedURL = p
	}
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

	driverName = "postgres"

	driverSource = fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=%s", username, databaseName, password, host, port, sslMode)

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
