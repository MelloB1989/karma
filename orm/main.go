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

func (o *ORM) GetByFieldsEquals(filters map[string]interface{}) (any, error) {
	fmt.Println(reflect.TypeOf(o.structType))

	// Check if any filters are provided
	if len(filters) == 0 {
		return nil, fmt.Errorf("no filters provided")
	}

	// Prepare slices to hold WHERE clauses and their corresponding values
	var whereClauses []string
	var args []interface{}
	placeholder := 1 // PostgreSQL placeholders start at $1

	// Iterate over the filters to build the WHERE clause
	for fieldName, value := range filters {
		columnName, ok := o.fieldMap[fieldName]
		if !ok {
			return nil, fmt.Errorf("field %s not found in struct", fieldName)
		}
		// Append the condition with the appropriate placeholder
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", columnName, placeholder))
		args = append(args, value)
		placeholder++
	}

	// Join all conditions with AND
	whereStatement := strings.Join(whereClauses, " AND ")

	// Construct the final query
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s", o.tableName, whereStatement)

	// Connect to the database
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Execute the query with the collected arguments
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("Failed to get rows by field comparison:", err)
		return nil, err
	}
	defer rows.Close()

	// Dynamically create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType))
	resultsPtr := reflect.New(sliceType) // *([]*structType)

	// Parse the rows into the results slice
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

func (o *ORM) QueryRaw(sqlQuery string, args ...interface{}) (interface{}, error) {
	// Establish database connection
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return nil, err
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		log.Println("Query execution error:", err)
		return nil, err
	}
	defer rows.Close()

	// Retrieve column names from the result
	columns, err := rows.Columns()
	if err != nil {
		log.Println("Failed to retrieve columns:", err)
		return nil, err
	}

	// Reverse fieldMap to map columns to struct fields
	columnToField := make(map[string]string) // column name -> field name
	for field, column := range o.fieldMap {
		columnToField[column] = field
	}

	// Prepare a slice to hold the results
	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType)) // []*StructType
	results := reflect.MakeSlice(sliceType, 0, 0)

	// Iterate over the rows
	for rows.Next() {
		// Create a new instance of the struct
		structPtr := reflect.New(o.structType) // *StructType
		structVal := structPtr.Elem()          // StructType

		// Prepare a slice for Scan destination pointers
		scanDest := make([]interface{}, len(columns))
		for i, col := range columns {
			if fieldName, ok := columnToField[col]; ok {
				field := structVal.FieldByName(fieldName)
				if !field.IsValid() {
					// Field not found; use a dummy variable
					var dummy interface{}
					scanDest[i] = &dummy
				} else {
					scanDest[i] = field.Addr().Interface()
				}
			} else {
				// Column does not map to any struct field; use a dummy variable
				var dummy interface{}
				scanDest[i] = &dummy
			}
		}

		// Scan the row into the struct fields
		if err := rows.Scan(scanDest...); err != nil {
			log.Println("Failed to scan row:", err)
			return nil, err
		}

		// Append the struct pointer to the results slice
		results = reflect.Append(results, structPtr)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Println("Rows iteration error:", err)
		return nil, err
	}

	return results.Interface(), nil
}

func (o *ORM) DeleteByFieldEquals(fieldName string, value any) (int64, error) {
	// Check if the field exists in the struct
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %s not found in struct", fieldName)
	}

	// Establish database connection
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}
	defer db.Close()

	// Construct DELETE query
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", o.tableName, columnName)

	// Execute the query
	result, err := db.Exec(query, value)
	if err != nil {
		log.Println("Failed to execute DELETE:", err)
		return 0, err
	}

	// Get the number of rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Failed to retrieve RowsAffected:", err)
		return 0, err
	}

	return rowsAffected, nil
}

/*
rowsDeleted, err := ormInstance.DeleteByFieldEquals("username", "johndoe")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted %d rows\n", rowsDeleted)
*/

func (o *ORM) DeleteByFieldCompare(fieldName string, value any, operator string) (int64, error) {
	// Check if the field exists in the struct
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %s not found in struct", fieldName)
	}

	// Sanitize the operator to prevent SQL injection
	allowedOperators := map[string]bool{
		"=":    true,
		">":    true,
		"<":    true,
		">=":   true,
		"<=":   true,
		"LIKE": true,
	}
	if !allowedOperators[operator] {
		return 0, fmt.Errorf("unsupported operator: %s", operator)
	}

	// Establish database connection
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}
	defer db.Close()

	// Construct DELETE query
	query := fmt.Sprintf("DELETE FROM %s WHERE %s %s $1", o.tableName, columnName, operator)

	// Execute the query
	result, err := db.Exec(query, value)
	if err != nil {
		log.Println("Failed to execute DELETE:", err)
		return 0, err
	}

	// Get the number of rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Failed to retrieve RowsAffected:", err)
		return 0, err
	}

	return rowsAffected, nil
}

/*
rowsDeleted, err := ormInstance.DeleteByFieldCompare("age", 30, ">")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted %d rows where age > 30\n", rowsDeleted)
*/

func (o *ORM) DeleteByFieldIn(fieldName string, values []any) (int64, error) {
	// Check if the field exists in the struct
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %s not found in struct", fieldName)
	}

	if len(values) == 0 {
		return 0, fmt.Errorf("values slice is empty")
	}

	// Establish database connection
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}
	defer db.Close()

	// Construct DELETE query with IN clause
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", o.tableName, columnName, strings.Join(placeholders, ", "))

	// Convert []any to []interface{}
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}

	// Execute the query
	result, err := db.Exec(query, args...)
	if err != nil {
		log.Println("Failed to execute DELETE with IN:", err)
		return 0, err
	}

	// Get the number of rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Failed to retrieve RowsAffected:", err)
		return 0, err
	}

	return rowsAffected, nil
}

/*
userIDs := []any{1, 2, 3, 4, 5}
rowsDeleted, err := ormInstance.DeleteByFieldIn("id", userIDs)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted %d rows with IDs in %v\n", rowsDeleted, userIDs)
*/

func (o *ORM) DeleteByPrimaryKey(pkValue any) (int64, error) {
	primaryKey := o.getPrimaryKeyField()
	if primaryKey == "" {
		return 0, fmt.Errorf("primary key not found in struct tags")
	}
	return o.DeleteByFieldEquals(primaryKey, pkValue)
}

/*
rowsDeleted, err := ormInstance.DeleteByPrimaryKey(10)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted %d row with primary key 10\n", rowsDeleted)
*/

func (o *ORM) DeleteAll() (int64, error) {
	// Establish database connection
	db, err := database.PostgresConn()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}
	defer db.Close()

	// Construct DELETE ALL query
	query := fmt.Sprintf("DELETE FROM %s", o.tableName)

	// Execute the query
	result, err := db.Exec(query)
	if err != nil {
		log.Println("Failed to execute DELETE ALL:", err)
		return 0, err
	}

	// Get the number of rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Failed to retrieve RowsAffected:", err)
		return 0, err
	}

	return rowsAffected, nil
}

/*
rowsDeleted, err := ormInstance.DeleteAll()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Deleted all %d rows from the table\n", rowsDeleted)
*/

// TableData represents the data for a single table in the join result
type TableData struct {
	TableName string
	Data      interface{}
}

// JoinResult represents the full result of a join operation
type JoinResult struct {
	Tables map[string]interface{} // Maps table names to their respective data
}

// JoinCondition represents a simple join between two tables with added table name tracking
type JoinCondition struct {
	Target      interface{} // The target struct to join with
	OnField     string      // The field to join on (from source table)
	TargetField string      // The field to join on (from target table)
	TableName   string      // The name of the target table (will be populated automatically)
}

// JoinBuilder helps build and execute join queries with table tracking
type JoinBuilder struct {
	sourceORM    *ORM
	joins        []JoinCondition
	whereField   string
	whereValue   interface{}
	resultStruct interface{}
}

// Join sets up a join with another table
func (o *ORM) Join(joinCond JoinCondition) *JoinBuilder {
	// Get the table name from the Target struct
	targetType := reflect.TypeOf(joinCond.Target).Elem()
	if field, ok := targetType.FieldByName("TableName"); ok {
		joinCond.TableName = field.Tag.Get("karma_table")
	}

	return &JoinBuilder{
		sourceORM: o,
		joins:     []JoinCondition{joinCond},
	}
}

// Into specifies the result struct type to map the joined data into
func (jb *JoinBuilder) Into(result interface{}) *JoinBuilder {
	jb.resultStruct = result
	return jb
}

// Where adds a where clause to the join query
func (jb *JoinBuilder) Where(field string, value interface{}) *JoinBuilder {
	jb.whereField = field
	jb.whereValue = value
	return jb
}

// Execute runs the join query and returns the results organized by table
func (jb *JoinBuilder) Execute() ([]interface{}, error) {
	if jb.resultStruct == nil {
		return nil, fmt.Errorf("result struct not specified. Use .Into() to specify result type")
	}

	db, err := database.PostgresConn()
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return nil, fmt.Errorf("database connection error: %v", err)
	}
	defer db.Close()

	// Build the query
	query, err := jb.buildJoinQuery()
	if err != nil {
		log.Printf("Failed to build join query: %v", err)
		return nil, fmt.Errorf("query building error: %v", err)
	}

	// Execute query
	var rows *sql.Rows
	if jb.whereValue != nil {
		rows, err = db.Query(query, jb.whereValue)
	} else {
		rows, err = db.Query(query)
	}
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return nil, fmt.Errorf("query execution error: %v", err)
	}
	defer rows.Close()

	// Get column names from the result
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Failed to get column names: %v", err)
		return nil, fmt.Errorf("failed to get column names: %v", err)
	}

	var results []interface{}

	for rows.Next() {
		// Create scan destinations
		values := make([]interface{}, len(columns))
		for i := range values {
			values[i] = new(interface{})
		}

		// Scan the row into the values slice
		if err := rows.Scan(values...); err != nil {
			log.Printf("Failed to scan row: %v", err)
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Create a new instance of the result struct
		resultValue := reflect.New(reflect.TypeOf(jb.resultStruct).Elem())
		result := resultValue.Interface()

		// Create maps for both tables
		sourceData := make(map[string]interface{})
		targetData := make(map[string]interface{})

		// Organize values into their respective maps
		for i, col := range columns {
			value := *(values[i].(*interface{}))
			if value == nil {
				continue
			}

			parts := strings.Split(col, "_")
			if len(parts) < 2 {
				continue
			}

			tableName := parts[0]
			fieldName := strings.Join(parts[1:], "_")

			switch tableName {
			case jb.sourceORM.tableName:
				sourceData[fieldName] = value
			case jb.joins[0].TableName:
				targetData[fieldName] = value
			}
		}

		// Set the maps in the result struct
		resultVal := resultValue.Elem()

		sourceField := resultVal.FieldByName(strings.Title(jb.sourceORM.tableName))
		if sourceField.IsValid() && sourceField.CanSet() {
			sourceField.Set(reflect.ValueOf(sourceData))
		}

		targetField := resultVal.FieldByName(strings.Title(jb.joins[0].TableName))
		if targetField.IsValid() && targetField.CanSet() {
			targetField.Set(reflect.ValueOf(targetData))
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v", err)
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return results, nil
}

// Helper function to build the join query
func (jb *JoinBuilder) buildJoinQuery() (string, error) {
	// Get all columns from source table
	sourceColumns := getTableColumns(jb.sourceORM.tableName, jb.sourceORM.structType)

	// Get all columns from target table
	targetColumns := getTableColumns(jb.joins[0].TableName, reflect.TypeOf(jb.joins[0].Target).Elem())

	// Combine all columns
	allColumns := append(sourceColumns, targetColumns...)

	// Construct base query
	query := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(allColumns, ", "),
		jb.sourceORM.tableName)

	// Add JOIN clause
	query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.%s",
		jb.joins[0].TableName,
		jb.sourceORM.tableName, jb.joins[0].OnField,
		jb.joins[0].TableName, jb.joins[0].TargetField)

	// Add WHERE clause if specified
	if jb.whereField != "" {
		query += fmt.Sprintf(" WHERE %s.%s = $1", jb.sourceORM.tableName, jb.whereField)
	}

	return query, nil
}

// Helper function to get table columns with proper aliases
func getTableColumns(tableName string, structType reflect.Type) []string {
	var columns []string
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" &&
			field.Name != "TableName" {
			columns = append(columns,
				fmt.Sprintf("%s.%s as %s_%s",
					tableName, jsonTag, tableName, jsonTag))
		}
	}
	return columns
}
