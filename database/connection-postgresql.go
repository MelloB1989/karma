package database

import (
	"fmt"
	"log"

	"github.com/MelloB1989/karma/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func PostgresConn() (*sqlx.DB, error) {

	env := config.DefaultConfig()
	ssl := env.DatabaseSSLMode
	var driverName string
	var driverSource string

	driverName = "postgres"
	if ssl == "true" {
		driverSource = fmt.Sprintf("user=%s dbname=%s password=%s host=%s port=%s sslmode=require", env.DatabaseUser, env.DatabaseName, env.DatabasePassword, env.DatabaseHost, env.DatabasePort)
	} else {
		driverSource = fmt.Sprintf("user=%s dbname=%s sslmode=disable password=%s host=%s port=%s", env.DatabaseUser, env.DatabaseName, env.DatabasePassword, env.DatabaseHost, env.DatabasePort)
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
