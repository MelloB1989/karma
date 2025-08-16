package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func FetchColumnNames(db *sqlx.DB, tableName string) ([]string, error) {
	query := "SELECT column_name FROM information_schema.columns WHERE table_name = $1"
	rows, err := db.Queryx(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, nil
}

func ParseRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("destination must be a pointer to a slice")
	}
	sliceValue := destValue.Elem()
	elemType := sliceValue.Type().Elem()

	// Ensure elemType is a struct
	isPtr := false
	if elemType.Kind() == reflect.Ptr {
		isPtr = true
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return errors.New("destination slice must contain struct elements")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// Build a mapping from column names to struct fields
	columnToFieldMap := make(map[string]reflect.StructField)
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Skip fields with json:"-" (like TableName)
		if jsonTag == "-" {
			continue
		}

		var columnName string
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options from json tag
			parts := strings.Split(jsonTag, ",")
			columnName = parts[0]
		} else {
			columnName = camelToSnake(field.Name)
		}

		columnToFieldMap[columnName] = field

		// Also map db tag if it exists for special handling
		if dbTag != "" && dbTag != "-" {
			columnToFieldMap[dbTag] = field
		}
	}

	for rows.Next() {
		columnValues := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return err
		}

		// Create a new instance of the struct (or a pointer to the struct)
		elem := reflect.New(elemType).Elem()

		for i, column := range columns {
			fieldInfo, ok := columnToFieldMap[column]
			if !ok {
				// Try snakeToCamel conversion
				fieldInfo, ok = columnToFieldMap[snakeToCamel(column)]
				if !ok {
					continue // Skip columns without corresponding struct fields
				}
			}

			field := elem.FieldByName(fieldInfo.Name)
			if !field.IsValid() || !field.CanSet() {
				continue
			}

			// Check if columnValues[i] is nil
			if columnValues[i] == nil {
				// Handle nil values - set zero value for the field type
				if field.CanSet() {
					field.Set(reflect.Zero(field.Type()))
				}
				continue
			}

			val := reflect.ValueOf(columnValues[i])

			// Add check for zero value before calling val.Type()
			if !val.IsValid() {
				log.Printf("Invalid value for field %s, skipping", fieldInfo.Name)
				continue
			}

			// Handle fields with 'db' tag
			if fieldInfo.Tag.Get("db") != "" {
				// Obtain the data as []byte
				var data []byte
				switch v := val.Interface().(type) {
				case []byte:
					data = v
				case string:
					data = []byte(v)
				default:
					log.Printf("Unsupported type for field with 'db' tag: %s, got type: %v", fieldInfo.Name, val.Type())
					continue
				}

				// Unmarshal the JSON into the field
				if err := json.Unmarshal(data, field.Addr().Interface()); err != nil {
					// If unmarshal fails, try to handle the case where we have a string that needs to be converted
					var jsonStr string
					if err := json.Unmarshal(data, &jsonStr); err == nil {
						// If the field is a float32
						if field.Kind() == reflect.Float32 {
							if f, err := stringToFloat32(jsonStr); err == nil {
								field.SetFloat(float64(f))
								continue
							}
						}
						// TODO: Add more type conversions here
					}
					log.Printf("Failed to unmarshal JSON for field %s: %v (data: %s)", fieldInfo.Name, err, string(data))
					continue
				}
			} else {
				// Standard processing for other fields
				if val.Kind() == reflect.Ptr && !val.IsNil() {
					val = val.Elem()
				}
				if val.Kind() == reflect.Interface && !val.IsNil() {
					val = val.Elem()
				}

				// Additional check after dereferencing
				if !val.IsValid() {
					log.Printf("Invalid value after dereferencing for field %s, skipping", fieldInfo.Name)
					continue
				}

				if val.Type().ConvertibleTo(field.Type()) {
					field.Set(val.Convert(field.Type()))
				} else if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 && field.Kind() == reflect.String {
					// Convert []byte to string
					field.SetString(string(val.Interface().([]byte)))
				} else {
					// Safe check before calling val.Type()
					if val.IsValid() {
						log.Printf("Cannot set field %s with value of type %v", fieldInfo.Name, val.Type())
					} else {
						log.Printf("Cannot set field %s with invalid value", fieldInfo.Name)
					}
				}
			}
		}

		// Append the new struct (or pointer) to the slice
		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, elem.Addr()))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, elem))
		}
	}

	return nil
}

func stringToFloat32(s string) (float32, error) {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}
	return float32(f), nil
}

func snakeToCamel(s string) string {
	caser := cases.Title(language.English)
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = caser.String(parts[i])
	}
	return strings.Join(parts, "")
}

func camelToSnake(s string) string {
	var snake string
	for i, c := range s {
		if unicode.IsUpper(c) {
			if i > 0 {
				snake += "_"
			}
			snake += string(unicode.ToLower(c))
		} else {
			snake += string(c)
		}
	}
	return snake
}

// InsertStruct inserts a struct's fields into the specified table.
func InsertStruct(db *sqlx.DB, tableName string, data any) error {
	var err error
	if db == nil {
		db, err = PostgresConn()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	}
	// Input validation
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("data must be a pointer to a struct, but got %s", val.Kind())
	}

	// Ensure that the pointer is not nil
	if val.IsNil() {
		return fmt.Errorf("data pointer is nil")
	}

	// Dereference the pointer
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("data must point to a struct, but points to %s", elem.Kind())
	}

	typ := elem.Type()

	var columns []string
	var values []any

	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)

		// Retrieve tags
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Skip fields with json:"-" (like TableName)
		if jsonTag == "-" {
			continue
		}

		// Determine the column name - prioritize json tag
		var column string
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options from json tag
			parts := strings.Split(jsonTag, ",")
			column = parts[0]
		} else {
			// Skip fields without json tags
			continue
		}

		fieldValue := elem.Field(i)

		// Handle pointer fields: get the actual value or nil
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				values = append(values, nil)
				columns = append(columns, column)
				continue
			}
			// Dereference pointer
			fieldValue = fieldValue.Elem()
		}

		// Handle interface{} types by getting the underlying value
		if fieldValue.Kind() == reflect.Interface && !fieldValue.IsNil() {
			fieldValue = fieldValue.Elem()
		}

		// Handle special types: Slice, Map, Struct (or if dbTag exists for JSON serialization)
		if fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Map || fieldValue.Kind() == reflect.Struct || dbTag != "" {
			// Marshal the field to JSON
			jsonBytes, err := json.Marshal(fieldValue.Interface())
			if err != nil {
				log.Printf("Failed to marshal JSON for field '%s': %v\n", field.Name, err)
				return fmt.Errorf("failed to marshal JSON for field '%s': %w", field.Name, err)
			}
			values = append(values, string(jsonBytes))
		} else {
			// Handle other types directly
			values = append(values, fieldValue.Interface())
		}

		columns = append(columns, column)
	}

	// Ensure there are columns to insert
	if len(columns) == 0 {
		return fmt.Errorf("no columns to insert for table '%s'", tableName)
	}

	// Construct the SQL query with placeholders
	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s)`,
		tableName,
		strings.Join(columns, ", "),
		placeholders(len(values)),
	)

	// Execute the query
	_, err = db.Exec(query, values...)
	if err != nil {
		log.Printf("Failed to insert data into table '%s': %v\n", tableName, err)
		return fmt.Errorf("failed to insert data into table '%s': %w", tableName, err)
	}

	return nil
}

func InsertTrxStruct(db *sqlx.Tx, tableName string, data any) error {
	// Input validation
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("data must be a pointer to a struct, but got %s", val.Kind())
	}

	// Ensure that the pointer is not nil
	if val.IsNil() {
		return fmt.Errorf("data pointer is nil")
	}

	// Dereference the pointer
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("data must point to a struct, but points to %s", elem.Kind())
	}

	typ := elem.Type()

	var columns []string
	var values []any

	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)

		// Retrieve tags
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Skip fields with json:"-" (like TableName)
		if jsonTag == "-" {
			continue
		}

		// Determine the column name - prioritize json tag
		var column string
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options from json tag
			parts := strings.Split(jsonTag, ",")
			column = parts[0]
		} else {
			// Skip fields without json tags
			continue
		}

		fieldValue := elem.Field(i)

		// Handle pointer fields: get the actual value or nil
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				values = append(values, nil)
				columns = append(columns, column)
				continue
			}
			// Dereference pointer
			fieldValue = fieldValue.Elem()
		}

		// Handle interface{} types by getting the underlying value
		if fieldValue.Kind() == reflect.Interface && !fieldValue.IsNil() {
			fieldValue = fieldValue.Elem()
		}

		// Handle special types: Slice, Map, Struct (or if dbTag exists for JSON serialization)
		if fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Map || fieldValue.Kind() == reflect.Struct || dbTag != "" {
			// Marshal the field to JSON
			jsonBytes, err := json.Marshal(fieldValue.Interface())
			if err != nil {
				log.Printf("Failed to marshal JSON for field '%s': %v\n", field.Name, err)
				return fmt.Errorf("failed to marshal JSON for field '%s': %w", field.Name, err)
			}
			values = append(values, string(jsonBytes))
		} else {
			// Handle other types directly
			values = append(values, fieldValue.Interface())
		}

		columns = append(columns, column)
	}

	// Ensure there are columns to insert
	if len(columns) == 0 {
		return fmt.Errorf("no columns to insert for table '%s'", tableName)
	}

	// Construct the SQL query with placeholders
	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s)`,
		tableName,
		strings.Join(columns, ", "),
		placeholders(len(values)),
	)

	// Execute the query
	_, err := db.Exec(query, values...)
	if err != nil {
		log.Printf("Failed to insert data into table '%s': %v\n", tableName, err)
		return fmt.Errorf("failed to insert data into table '%s': %w", tableName, err)
	}

	return nil
}

// UpdateStruct updates fields in the specified table for a given struct based on a condition.
func UpdateStruct(db *sqlx.DB, tableName string, data any, conditionField string, conditionValue any) error {
	var err error
	if db == nil {
		db, err = PostgresConn()
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	}
	// Input validation
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("data must be a pointer to a struct, but got %s", val.Kind())
	}

	// Ensure that the pointer is not nil
	if val.IsNil() {
		return fmt.Errorf("data pointer is nil")
	}

	// Dereference the pointer
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("data must point to a struct, but points to %s", elem.Kind())
	}

	var columns []string
	var values []any
	typ := elem.Type()
	placeholderIdx := 1 // Start placeholder index at 1
	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Skip fields with json:"-" (like TableName)
		if jsonTag == "-" {
			continue
		}

		// Determine the column name - prioritize json tag
		var column string
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options from json tag
			parts := strings.Split(jsonTag, ",")
			column = parts[0]
		} else {
			// Skip fields without json tags
			continue
		}
		// Skip the condition field to avoid updating it
		if column == conditionField {
			continue
		}
		fieldValue := elem.Field(i)

		// Handle pointer fields: get the actual value or nil
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				values = append(values, nil)
				columns = append(columns, fmt.Sprintf("%s = $%d", camelToSnake(column), placeholderIdx))
				placeholderIdx++
				continue
			}
			// Dereference pointer
			fieldValue = fieldValue.Elem()
		}

		// Handle interface{} types by getting the underlying value
		if fieldValue.Kind() == reflect.Interface && !fieldValue.IsNil() {
			fieldValue = fieldValue.Elem()
		}

		// Handle slice, map, struct, or fields marked with db tag for JSON serialization
		if fieldValue.Kind() == reflect.Slice ||
			fieldValue.Kind() == reflect.Map ||
			fieldValue.Kind() == reflect.Struct ||
			dbTag != "" {
			jsonValue, err := json.Marshal(fieldValue.Interface())
			if err != nil {
				log.Printf("Failed to marshal JSON field '%s': %v\n", column, err)
				return fmt.Errorf("failed to marshal JSON field '%s': %w", column, err)
			}
			values = append(values, string(jsonValue)) // Add serialized JSON as string
		} else {
			// Use the actual value for other types
			values = append(values, fieldValue.Interface())
		}
		// Add the column update statement with the placeholder
		columns = append(columns, fmt.Sprintf("%s = $%d", camelToSnake(column), placeholderIdx))
		placeholderIdx++
	}
	// Add the condition field and its value as the last placeholder
	values = append(values, conditionValue)
	query := fmt.Sprintf(
		`UPDATE %s SET %s WHERE %s = $%d`,
		tableName,
		strings.Join(columns, ", "),
		camelToSnake(conditionField), // Convert condition field to snake_case if necessary
		placeholderIdx,
	)

	// Execute the query
	_, err = db.Exec(query, values...)
	if err != nil {
		log.Printf("Failed to update record in table: %v. Query: %s, Values: %v", err, query, values)
		return err
	}
	return nil
}

func UpdateTrxStruct(db *sqlx.Tx, tableName string, data any, conditionField string, conditionValue any) error {
	// Input validation
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("data must be a pointer to a struct, but got %s", val.Kind())
	}

	// Ensure that the pointer is not nil
	if val.IsNil() {
		return fmt.Errorf("data pointer is nil")
	}

	// Dereference the pointer
	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("data must point to a struct, but points to %s", elem.Kind())
	}

	var columns []string
	var values []any
	typ := elem.Type()
	placeholderIdx := 1 // Start placeholder index at 1
	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		// Skip fields with json:"-" (like TableName)
		if jsonTag == "-" {
			continue
		}

		// Determine the column name - prioritize json tag
		var column string
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options from json tag
			parts := strings.Split(jsonTag, ",")
			column = parts[0]
		} else {
			// Skip fields without json tags
			continue
		}
		// Skip the condition field to avoid updating it
		if column == conditionField {
			continue
		}
		fieldValue := elem.Field(i)

		// Handle pointer fields: get the actual value or nil
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				values = append(values, nil)
				columns = append(columns, fmt.Sprintf("%s = $%d", camelToSnake(column), placeholderIdx))
				placeholderIdx++
				continue
			}
			// Dereference pointer
			fieldValue = fieldValue.Elem()
		}

		// Handle interface{} types by getting the underlying value
		if fieldValue.Kind() == reflect.Interface && !fieldValue.IsNil() {
			fieldValue = fieldValue.Elem()
		}

		// Handle slice, map, struct, or fields marked with db tag for JSON serialization
		if fieldValue.Kind() == reflect.Slice ||
			fieldValue.Kind() == reflect.Map ||
			fieldValue.Kind() == reflect.Struct ||
			dbTag != "" {
			jsonValue, err := json.Marshal(fieldValue.Interface())
			if err != nil {
				log.Printf("Failed to marshal JSON field '%s': %v\n", column, err)
				return fmt.Errorf("failed to marshal JSON field '%s': %w", column, err)
			}
			values = append(values, string(jsonValue)) // Add serialized JSON as string
		} else {
			// Use the actual value for other types
			values = append(values, fieldValue.Interface())
		}
		// Add the column update statement with the placeholder
		columns = append(columns, fmt.Sprintf("%s = $%d", camelToSnake(column), placeholderIdx))
		placeholderIdx++
	}
	// Add the condition field and its value as the last placeholder
	values = append(values, conditionValue)
	query := fmt.Sprintf(
		`UPDATE %s SET %s WHERE %s = $%d`,
		tableName,
		strings.Join(columns, ", "),
		camelToSnake(conditionField), // Convert condition field to snake_case if necessary
		placeholderIdx,
	)

	// Execute the query
	_, err := db.Exec(query, values...)
	if err != nil {
		log.Printf("Failed to update record in table: %v. Query: %s, Values: %v", err, query, values)
		return err
	}
	return nil
}

// placeholders generates a string of placeholders for SQL based on the number of fields.
func placeholders(n int) string {
	ph := make([]string, n)
	for i := range ph {
		ph[i] = "$" + strconv.Itoa(i+1)
	}
	return strings.Join(ph, ", ")
}
