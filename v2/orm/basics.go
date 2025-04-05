package orm

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/MelloB1989/karma/database"
	"github.com/jmoiron/sqlx"
)

// ORM struct encapsulates the metadata and methods for a table.
type ORM struct {
	tableName  string
	structType reflect.Type
	fieldMap   map[string]string
	tx         *sqlx.Tx
	db         *sqlx.DB
}

// QueryResult holds the result of a query operation and any error that occurred
type QueryResult struct {
	rows  *sql.Rows
	err   error
	query string
	args  []any
}

// Load initializes the ORM with the given struct.
func Load(entity any) *ORM {
	if entity == nil {
		log.Printf("Error: entity cannot be nil")
		return nil
	}

	entityType := reflect.TypeOf(entity)
	if entityType.Kind() != reflect.Ptr {
		log.Printf("Error: entity must be a pointer to a struct")
		return nil
	}

	t := entityType.Elem() // Get the type of the struct
	if t.Kind() != reflect.Struct {
		log.Printf("Error: entity must be a pointer to a struct")
		return nil
	}

	tableName := ""

	// Get the table name from the struct tag
	if field, ok := t.FieldByName("TableName"); ok {
		tableName = field.Tag.Get("karma_table")
		if tableName == "" {
			log.Printf("Warning: TableName field found but karma_table tag is empty")
		}
	} else {
		log.Printf("Warning: No TableName field found with karma_table tag")
	}

	// Build the field mapping
	fieldMap := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// Handle cases where json tag includes options like `json:"name,omitempty"`
			parts := strings.Split(jsonTag, ",")
			fieldMap[field.Name] = parts[0]
		} else {
			fieldMap[field.Name] = field.Name
		}
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Printf("Database connection error: %v", err)
		return nil
	}

	return &ORM{
		tableName:  tableName,
		structType: t,
		fieldMap:   fieldMap,
		db:         db,
		tx:         nil,
	}
}

// Scan maps the query result to the provided destination pointer
func (qr *QueryResult) Scan(dest any) error {
	err := database.ParseRows(qr.rows, dest)
	if err != nil {
		log.Println("Failed to scan rows:", err)
		return err
	}
	return nil
}

// QueryRaw to return a QueryResult for chaining
// func (o *ORM) QueryRaw(sqlQuery string, args ...any) *QueryResult {
// 	// Establish database connection
// 	db, err := database.PostgresConn()
// 	if err != nil {
// 		log.Println("DB connection error:", err)
// 		return &QueryResult{nil, err, sqlQuery, args}
// 	}
// 	defer db.Close()

// // Execute the query
// rows, err := db.Query(sqlQuery, args...)
//
//	if err != nil {
//		log.Println("Query execution error:", err)
//		return &QueryResult{nil, err, sqlQuery, args}
//	}
//
//		return &QueryResult{rows, nil, sqlQuery, args}
//	}
func (o *ORM) QueryRaw(query string, args ...any) *QueryResult {
	var rows *sql.Rows
	var err error

	// Use transaction if available, otherwise use the database connection
	if o.tx != nil {
		rows, err = o.tx.Query(query, args...)
	} else {
		rows, err = o.db.Query(query, args...)
	}

	if err != nil {
		return &QueryResult{nil, err, query, args}
	}

	return &QueryResult{
		rows:  rows,
		err:   err,
		query: query,
		args:  args,
	}
}

// normalizeValues converts various input formats into a flat slice of values
func (o *ORM) normalizeValues(values ...any) ([]any, error) {
	var valuesSlice []any

	if len(values) == 0 {
		return nil, fmt.Errorf("no values provided")
	}

	if len(values) == 1 {
		// Check if it's a slice
		val := reflect.ValueOf(values[0])
		if val.Kind() == reflect.Slice {
			// Convert the slice to []any
			valuesSlice = make([]any, val.Len())
			for i := range valuesSlice {
				valuesSlice[i] = val.Index(i).Interface()
			}
		} else {
			// Single value
			valuesSlice = values
		}
	} else {
		// Multiple values passed directly
		valuesSlice = values
	}

	// Check if we ended up with an empty slice
	if len(valuesSlice) == 0 {
		return nil, fmt.Errorf("no values provided after normalization")
	}

	return valuesSlice, nil
}

// resolveColumn gets the DB column name for a struct field name
func (o *ORM) resolveColumn(fieldName string) (string, error) {
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s not found in struct", fieldName)
	}
	return columnName, nil
}

// generatePlaceholders creates SQL placeholders ($1, $2, etc.)
func generatePlaceholders(count int) string {
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}
	return strings.Join(placeholders, ", ")
}

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

// GetQuery returns the SQL query that produced this result
func (qr *QueryResult) GetQuery() string {
	return qr.query
}

// GetArgs returns the arguments used in the query
func (qr *QueryResult) GetArgs() []any {
	return qr.args
}
