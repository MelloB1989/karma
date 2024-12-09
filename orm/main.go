package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

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

// Helper function to get the field name from a field pointer
func GetFieldName(structPtr any, fieldPtr any) (string, error) {
	sValue := reflect.ValueOf(structPtr)
	if sValue.Kind() != reflect.Ptr || sValue.Elem().Kind() != reflect.Struct {
		return "", errors.New("structPtr must be a pointer to a struct")
	}
	sValue = sValue.Elem()

	fValue := reflect.ValueOf(fieldPtr)
	if fValue.Kind() != reflect.Ptr {
		return "", errors.New("fieldPtr must be a pointer")
	}
	fValue = fValue.Elem()

	// Get the base address of the struct
	sPtr := unsafe.Pointer(sValue.UnsafeAddr())

	// Get the address of the field
	fPtr := unsafe.Pointer(fValue.UnsafeAddr())

	// Calculate offset
	offset := uintptr(fPtr) - uintptr(sPtr)

	// Iterate over struct fields to find matching offset
	sType := sValue.Type()
	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		if field.Offset == offset {
			// Field found; return the JSON tag or field name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				return jsonTag, nil
			}
			return field.Name, nil
		}
	}

	return "", errors.New("field not found in struct")
}

func (o *ORM) GetByFieldCompare(structPtr any, fieldPtr any, value any, operator string) (any, error) {
	// Use reflection to get the field name
	fmt.Println(o.fieldMap)
	fieldName, err := GetFieldName(structPtr, fieldPtr)
	if err != nil {
		return nil, err
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
	query := "SELECT * FROM " + o.tableName + " WHERE " + fieldName + " " + operator + " $1"

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

func (o *ORM) GetByFieldIn(structPtr any, fieldPtr any, values []any) (any, error) {
	// Use reflection to get the field name
	fieldName, err := GetFieldName(structPtr, fieldPtr)
	if err != nil {
		return nil, err
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
	query := "SELECT * FROM " + o.tableName + " WHERE " + fieldName + " IN (" + strings.Join(placeholders, ", ") + ")"

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

func (o *ORM) GetCount(structPtr any, fieldPtr any, value any, operator string) (int, error) {
	// Use reflection to get the field name
	fieldName, err := GetFieldName(structPtr, fieldPtr)
	if err != nil {
		return 0, err
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
	query := "SELECT COUNT(*) FROM " + o.tableName + " WHERE " + fieldName + " " + operator + " $1"

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
