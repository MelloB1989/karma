package orm

import (
	"database/sql"
	"errors"
	"log"
	"reflect"
	"strings"

	"github.com/MelloB1989/karma/database"
)

// ORM struct encapsulates the metadata and methods for a table.
type ORM struct {
	tableName  string
	structType reflect.Type
}

// Load initializes the ORM with the given struct.
func Load(entity any) *ORM {
	t := reflect.TypeOf(entity).Elem() // Get the type of the struct
	tableName := ""

	// Get the table name from the struct tag
	if field, ok := t.FieldByName("TableName"); ok {
		tableName = field.Tag.Get("karma_table")
	}

	return &ORM{
		tableName:  tableName,
		structType: t,
	}
}

// GetAll fetches all rows from the table.
func (o *ORM) GetAll() (any, error) {
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM " + o.tableName)
	if err != nil {
		log.Println("Failed to get all rows:", err)
		return nil, err
	}

	// Dynamically create a slice to hold results using PointerTo
	results := reflect.New(reflect.SliceOf(reflect.PointerTo(o.structType))).Interface()

	if err := database.ParseRows(rows, results); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return reflect.ValueOf(results).Elem().Interface(), nil
}

// GetByPrimaryKey fetches a row by its primary key.
func (o *ORM) GetByPrimaryKey(key string) (any, error) {
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	primaryField := o.getPrimaryKeyField()
	if primaryField == "" {
		return nil, errors.New("primary key not defined in struct")
	}

	query := "SELECT * FROM " + o.tableName + " WHERE " + primaryField + " = $1"
	rows, err := db.Query(query, key)
	if err != nil {
		log.Println("Failed to get row by primary key:", err)
		return nil, err
	}

	// Create a slice to hold the result
	results := reflect.New(reflect.SliceOf(reflect.PointerTo(o.structType))).Interface()

	if err := database.ParseRows(rows, results); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	slice := reflect.ValueOf(results).Elem()
	if slice.Len() == 0 {
		return nil, sql.ErrNoRows
	}

	return slice.Index(0).Interface(), nil
}

// Insert inserts a new row into the table.
func (o *ORM) Insert(entity any) error {
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return err
	}
	defer db.Close()

	return database.InsertStruct(db, o.tableName, entity)
}

// Update updates an existing row in the table.
func (o *ORM) Update(entity any, primaryKeyValue string) error {
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return err
	}
	defer db.Close()

	primaryField := o.getPrimaryKeyField()
	if primaryField == "" {
		return errors.New("primary key not defined in struct")
	}

	return database.UpdateStruct(db, o.tableName, entity, primaryField, primaryKeyValue)
}

// Helper function to get the primary key field from struct tags.
func (o *ORM) getPrimaryKeyField() string {
	for i := 0; i < o.structType.NumField(); i++ {
		field := o.structType.Field(i)
		tag := field.Tag.Get("karma")
		if strings.Contains(tag, "primary") {
			return field.Tag.Get("json") // Assuming the json tag matches the column name
		}
	}
	return ""
}
