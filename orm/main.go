package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/MelloB1989/karma/database"
	"github.com/MelloB1989/karma/utils"
)

// ORM struct encapsulates the metadata and methods for a table.
type ORM struct {
	tableName  string
	structType reflect.Type
	fieldMap   map[string]string
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
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			fieldMap[field.Name] = jsonTag
		} else {
			fieldMap[field.Name] = field.Name
		}
	}

	return &ORM{
		tableName:  tableName,
		structType: t,
		fieldMap:   fieldMap,
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

func AssertAndReturnSlice(targetType reflect.Type, value interface{}, e error) ([]interface{}, error) {
	// Check if value is a slice
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice {
		log.Println("Expected a slice but got:", v.Kind())
		return nil, errors.New("value is not a slice")
	}

	// Create a new slice of the target type
	result := reflect.MakeSlice(reflect.SliceOf(targetType), v.Len(), v.Cap())
	reflect.Copy(result, v)

	// Return the value as a slice of interfaces
	var resultSlice []interface{}
	for i := 0; i < result.Len(); i++ {
		resultSlice = append(resultSlice, result.Index(i).Interface())
	}

	return resultSlice, nil
}

func (o *ORM) GetByFieldLike(fieldName string, value any) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + "LIKE" + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldGreaterThanEquals(fieldName string, value any) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + ">=" + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldLessThanEquals(fieldName string, value any) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + "<=" + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldGreaterThan(fieldName string, value any) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + ">" + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldLessThan(fieldName string, value any) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + "<" + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldEquals(fieldName string, value any) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + "=" + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldCompare(fieldName string, value any, operator string) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Sanitize the operator to avoid SQL injection
	allowedOperators := []string{"=", ">", "<", ">=", "<=", "LIKE"}
	if !utils.Contains(allowedOperators, operator) {
		return nil, errors.New("unsupported operator")
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + operator + " $1"

	rows, err := db.Query(query, value)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetByFieldIn(fieldName string, values []any) (any, error) {
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return nil, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Construct query with IN clause
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " IN (" + strings.Join(placeholders, ", ") + ")"

	rows, err := db.Query(query, values...)
	if err != nil {
		log.Println("Failed to get rows by field IN:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	if err := database.ParseRows(rows, resultsPtr.Interface()); err != nil {
		log.Println("Failed to parse rows:", err)
		return nil, err
	}

	// Return the slice directly
	return resultsPtr.Elem().Interface(), nil
}

func (o *ORM) GetCount(fieldName string, value any, operator string) (int, error) {
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %s not found in struct", fieldName)
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}
	defer db.Close()

	// Sanitize the operator to avoid SQL injection
	allowedOperators := []string{"=", ">", "<", ">=", "<=", "LIKE"}
	if !utils.Contains(allowedOperators, operator) {
		return 0, errors.New("unsupported operator")
	}

	// Construct query for count
	query := "SELECT COUNT(*) FROM " + o.tableName + " WHERE " + columnName + " " + operator + " $1"

	var count int
	err = db.QueryRow(query, value).Scan(&count)
	if err != nil {
		log.Println("Failed to get count by field comparison:", err)
		return 0, err
	}

	return count, nil
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
