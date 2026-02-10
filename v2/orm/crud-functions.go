package orm

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/MelloB1989/karma/database"
)

// GetByFieldIn to return a QueryResult for chaining
func (o *ORM) GetByFieldIn(fieldName string, values ...any) *QueryResult {
	// Normalize values into a slice
	valuesSlice, err := o.normalizeValues(values...)
	if err != nil {
		return &QueryResult{nil, fmt.Errorf("IN clause error: %w", err), "", nil, nil, o}
	}

	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query with IN clause
	placeholders := generatePlaceholders(len(valuesSlice))
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " IN (" + placeholders + ")"

	// Execute the query and return the result
	return o.QueryRaw(query, valuesSlice...)
}

func (o *ORM) GetAll() *QueryResult {
	query := "SELECT * FROM " + o.tableName
	return o.QueryRaw(query)
}

func (o *ORM) GetByPrimaryKey(value any) *QueryResult {
	// Get the primary key column name
	pkColumn := o.getPrimaryKeyField()

	// Construct the query
	query := "SELECT * FROM " + o.tableName + " WHERE " + pkColumn + " = $1"

	// Execute the query and return the result
	return o.QueryRaw(query, value)
}

// GetByFieldLike returns records where the field matches the LIKE pattern
func (o *ORM) GetByFieldLike(fieldName string, pattern string) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query with LIKE clause
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " LIKE $1"

	// Execute the query and return the result
	return o.QueryRaw(query, pattern)
}

// GetByFieldEquals returns records where the field equals the value
func (o *ORM) GetByFieldEquals(fieldName string, value any) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " = $1"

	// Execute the query and return the result
	return o.QueryRaw(query, value)
}

// GetByFieldsEquals returns records matching multiple field equality conditions
func (o *ORM) GetByFieldsEquals(fieldValueMap map[string]any) *QueryResult {
	if len(fieldValueMap) == 0 {
		return o.GetAll()
	}

	// Build WHERE conditions and collect values
	var conditions string
	values := make([]any, 0, len(fieldValueMap))
	paramCount := 1

	for fieldName, value := range fieldValueMap {
		// Get the column name
		columnName, err := o.resolveColumn(fieldName)
		if err != nil {
			return &QueryResult{nil, err, "", nil, nil, o}
		}

		if paramCount > 1 {
			conditions += " AND "
		}
		conditions += columnName + " = $" + fmt.Sprintf("%d", paramCount)
		values = append(values, value)
		paramCount++
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + conditions

	// Execute the query and return the result
	return o.QueryRaw(query, values...)
}

// GetByFieldGreaterThan returns records where the field is greater than the value
func (o *ORM) GetByFieldGreaterThan(fieldName string, value any) *QueryResult {
	return o.GetByFieldCompare(fieldName, ">", value)
}

// GetByFieldLessThan returns records where the field is less than the value
func (o *ORM) GetByFieldLessThan(fieldName string, value any) *QueryResult {
	return o.GetByFieldCompare(fieldName, "<", value)
}

// GetByFieldGreaterThanEquals returns records where the field is greater than or equal to the value
func (o *ORM) GetByFieldGreaterThanEquals(fieldName string, value any) *QueryResult {
	return o.GetByFieldCompare(fieldName, ">=", value)
}

// GetByFieldLessThanEquals returns records where the field is less than or equal to the value
func (o *ORM) GetByFieldLessThanEquals(fieldName string, value any) *QueryResult {
	return o.GetByFieldCompare(fieldName, "<=", value)
}

// GetByFieldCompare returns records using a custom comparison operator
func (o *ORM) GetByFieldCompare(fieldName string, operator string, value any) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Validate operator
	validOperators := map[string]bool{">": true, "<": true, ">=": true, "<=": true, "=": true, "!=": true, "<>": true}
	if !validOperators[operator] {
		return &QueryResult{nil, fmt.Errorf("invalid operator: %s", operator), "", nil, nil, o}
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " " + operator + " $1"

	// Execute the query and return the result
	return o.QueryRaw(query, value)
}

// GetCount returns the count of records matching the given condition
func (o *ORM) GetCount(filters map[string]any) (int, error) {
	// Check if any filters are provided
	if len(filters) == 0 {
		return 0, fmt.Errorf("no filters provided")
	}

	// Prepare slices to hold WHERE clauses and their corresponding values
	var whereClauses []string
	var args []any
	placeholder := 1 // PostgreSQL placeholders start at $1

	// Iterate over the filters to build the WHERE clause
	for fieldName, value := range filters {
		columnName, ok := o.fieldMap[fieldName]
		if !ok {
			return 0, fmt.Errorf("field %s not found in struct", fieldName)
		}

		// Append the condition with the appropriate placeholder (quote column name for case sensitivity)
		whereClauses = append(whereClauses, fmt.Sprintf(`"%s" = $%d`, columnName, placeholder))
		args = append(args, value)
		placeholder++
	}

	// Join all conditions with AND
	whereStatement := strings.Join(whereClauses, " AND ")

	// Get the shared database connection
	db, err := o.getDB()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}

	// Construct query for count
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", o.tableName, whereStatement)

	// Execute the query
	var count int
	err = db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		log.Println("Failed to get count by field comparison:", err)
		return 0, err
	}

	return count, nil
}

// GetTotalRowCount returns the total number of rows in the table
func (o *ORM) GetTotalRowCount() (int, error) {
	// Get the shared database connection
	db, err := o.getDB()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}

	// Construct simple count query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", o.tableName)

	// Execute the query
	var count int
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		log.Println("Failed to get total row count:", err)
		return 0, err
	}

	return count, nil
}

// GetByFieldNotEquals returns records where the field does not equal the value
func (o *ORM) GetByFieldNotEquals(fieldName string, value any) *QueryResult {
	return o.GetByFieldCompare(fieldName, "!=", value)
}

// GetByFieldIsNull returns records where the field is NULL
func (o *ORM) GetByFieldIsNull(fieldName string) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " IS NULL"

	// Execute the query and return the result
	return o.QueryRaw(query)
}

// GetByFieldIsNotNull returns records where the field is NOT NULL
func (o *ORM) GetByFieldIsNotNull(fieldName string) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " IS NOT NULL"

	// Execute the query and return the result
	return o.QueryRaw(query)
}

// GetByFieldBetween returns records where the field is between two values
func (o *ORM) GetByFieldBetween(fieldName string, start any, end any) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " WHERE " + columnName + " BETWEEN $1 AND $2"

	// Execute the query and return the result
	return o.QueryRaw(query, start, end)
}

// Limit limits the number of results returned
func (qr *QueryResult) Limit(limit int) *QueryResult {
	if qr.err != nil {
		return qr
	}

	// Assuming qr.Rows is a sql.Rows type
	// This would need to be implemented based on your database driver
	// This is a placeholder implementation
	return &QueryResult{qr.rows, fmt.Errorf("Limit not implemented yet"), "", nil, nil, qr.orm}
}

type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// OrderBy orders the results
func (o *ORM) OrderBy(fieldName string, direction OrderDirection) *QueryResult {
	// Get the column name
	columnName, err := o.resolveColumn(fieldName)
	if err != nil {
		return &QueryResult{nil, err, "", nil, nil, o}
	}

	// Construct query
	query := "SELECT * FROM " + o.tableName + " ORDER BY " + columnName + " " + string(direction)

	// Execute the query and return the result
	return o.QueryRaw(query)
}

// Insert inserts a new row into the table.
func (o *ORM) Insert(entity any) error {
	if o.tx != nil {
		return database.InsertTrxStruct(o.tx, o.tableName, entity)
	} else {
		db, err := o.getDB()
		if err != nil {
			log.Printf("Database connection error: %v", err)
			return err
		}
		return database.InsertStruct(db, o.tableName, entity)
	}
}

// Update updates an existing row in the table.
func (o *ORM) Update(entity any, primaryKeyValue string) error {
	primaryField := o.getPrimaryKeyField()
	if primaryField == "" {
		return errors.New("primary key not defined in struct")
	}

	if o.tx != nil {
		return database.UpdateTrxStruct(o.tx, o.tableName, entity, primaryField, primaryKeyValue)
	} else {
		db, err := o.getDB()
		if err != nil {
			log.Printf("Database connection error: %v", err)
			return err
		}
		return database.UpdateStruct(db, o.tableName, entity, primaryField, primaryKeyValue)
	}
}

func (o *ORM) DeleteByFieldEquals(fieldName string, value any) (int64, error) {
	// Check if the field exists in the struct
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %s not found in struct", fieldName)
	}

	// Get the shared database connection
	db, err := o.getDB()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}

	// Construct DELETE query (quote column name for case sensitivity)
	query := fmt.Sprintf(`DELETE FROM %s WHERE "%s" = $1`, o.tableName, columnName)

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

	// Get the shared database connection
	db, err := o.getDB()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}

	// Construct DELETE query (quote column name for case sensitivity)
	query := fmt.Sprintf(`DELETE FROM %s WHERE "%s" %s $1`, o.tableName, columnName, operator)

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

	// Get the shared database connection
	db, err := o.getDB()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}

	// Construct DELETE query with IN clause
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf(`DELETE FROM %s WHERE "%s" IN (%s)`, o.tableName, columnName, strings.Join(placeholders, ", "))

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
	// Get the shared database connection
	db, err := o.getDB()
	if err != nil {
		log.Println("DB connection error:", err)
		return 0, err
	}

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
