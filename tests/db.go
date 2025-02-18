package tests

import "github.com/MelloB1989/karma/database"

func TestDBConnection() {
	database.PostgresConn()
}
