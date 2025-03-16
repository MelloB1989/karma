package korm

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

// Operator represents comparison operators for query conditions
type Operator string

const (
	Equals              Operator = "="
	NotEquals           Operator = "!="
	GreaterThan         Operator = ">"
	LessThan            Operator = "<"
	GreaterThanOrEquals Operator = ">="
	LessThanOrEquals    Operator = "<="
	Like                Operator = "LIKE"
	IsNull              Operator = "IS NULL"
	IsNotNull           Operator = "IS NOT NULL"
	In                  Operator = "IN"
	Between             Operator = "BETWEEN"
)

// OrderDirection defines the sorting direction
type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// JoinType defines the type of SQL JOIN
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	FullJoin  JoinType = "FULL JOIN"
)

// Condition represents a WHERE clause condition
type Condition struct {
	Field    string
	Operator Operator
	Value    any
	Values   []any // For IN and BETWEEN operators
}

// Order represents an ORDER BY clause
type Order struct {
	Field     string
	Direction OrderDirection
}

// Join represents a JOIN clause
type Join struct {
	TableName  string
	Type       JoinType
	Conditions []Condition
}

// QueryBuilder builds SQL queries incrementally
type QueryBuilder struct {
	orm           *ORM
	operation     string
	selectFields  []string
	conditions    []Condition
	orders        []Order
	joins         []Join
	groupByFields []string
	havingConds   []Condition
	limit         int
	offset        int
	rawQuery      string
	rawArgs       []any
	isCount       bool
}

// ORM struct encapsulates the metadata and methods for a table
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
	t := reflect.TypeOf(entity).Elem() // Get the type of the struct
	tableName := ""

	// Get the table name from the struct tag
	if field, ok := t.FieldByName("TableName"); ok {
		tableName = field.Tag.Get("karma_table")
	}

	// Build the field mapping
	fieldMap := make(map[string]string)
	for i := range make([]int, t.NumField()) {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			fieldMap[field.Name] = jsonTag
		} else {
			fieldMap[field.Name] = field.Name
		}
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	// defer db.Close()

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
	if qr.err != nil {
		return qr.err
	}
	defer qr.rows.Close()

	// Check if there's a row to scan
	if !qr.rows.Next() {
		return sql.ErrNoRows
	}

	// Scan the row into the destination
	err := database.ParseRows(qr.rows, dest)
	if err != nil {
		return err
	}

	// Check for errors from iterating over rows
	if err := qr.rows.Err(); err != nil {
		return err
	}

	return nil
}

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
