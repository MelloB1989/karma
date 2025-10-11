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
	"sync"
	"unicode"

	"github.com/jmoiron/sqlx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TypeRegistry maps interface{} field names to their expected concrete types
var TypeRegistry = make(map[string]reflect.Type)
var typeRegistryMutex sync.RWMutex

func RegisterType(fieldName string, sampleValue interface{}) {
	typeRegistryMutex.Lock()
	defer typeRegistryMutex.Unlock()
	TypeRegistry[fieldName] = reflect.TypeOf(sampleValue)
}

func inferTypeFromJSON(data []byte, fieldName string) (interface{}, error) {
	if registeredType, exists := TypeRegistry[fieldName]; exists {
		newValue := reflect.New(registeredType).Interface()
		if err := json.Unmarshal(data, newValue); err == nil {
			return reflect.ValueOf(newValue).Elem().Interface(), nil
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err == nil {
		return result, nil
	}

	var genericResult interface{}
	if err := json.Unmarshal(data, &genericResult); err == nil {
		return genericResult, nil
	}

	return nil, errors.New("failed to unmarshal JSON data")
}

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
	if columns == nil {
		columns = []string{}
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

	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return errors.New("destination slice must contain struct elements")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	columnToFieldMap := buildColumnFieldMap(elemType)

	for rows.Next() {
		columnValues := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return err
		}

		elem := reflect.New(elemType).Elem()

		for i, column := range columns {
			fieldInfo := findFieldInfo(column, columnToFieldMap)
			if fieldInfo == nil {
				continue
			}

			field := elem.FieldByName(fieldInfo.Name)
			if !field.IsValid() || !field.CanSet() {
				continue
			}

			if err := setFieldValue(field, *fieldInfo, columnValues[i]); err != nil {
				log.Printf("Failed to set field %s: %v", fieldInfo.Name, err)
			}
		}

		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, elem.Addr()))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, elem))
		}
	}

	return nil
}

func buildColumnFieldMap(elemType reflect.Type) map[string]reflect.StructField {
	columnToFieldMap := make(map[string]reflect.StructField)

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		jsonTag := field.Tag.Get("json")
		dbTag := field.Tag.Get("db")

		if jsonTag == "-" {
			continue
		}

		if jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			jsonColumnName := parts[0]
			columnToFieldMap[jsonColumnName] = field
			columnToFieldMap[camelToSnake(jsonColumnName)] = field
		}

		columnToFieldMap[field.Name] = field
		columnToFieldMap[camelToSnake(field.Name)] = field

		if dbTag != "" && dbTag != "-" {
			columnToFieldMap[dbTag] = field
		}
	}

	return columnToFieldMap
}

func findFieldInfo(column string, columnToFieldMap map[string]reflect.StructField) *reflect.StructField {
	if fieldInfo, ok := columnToFieldMap[column]; ok {
		return &fieldInfo
	}

	conversions := []string{
		snakeToCamel(column),
		snakeToPascal(column),
		strings.ToLower(column),
	}

	for _, converted := range conversions {
		if fieldInfo, ok := columnToFieldMap[converted]; ok {
			return &fieldInfo
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, fieldInfo reflect.StructField, value interface{}) error {
	if value == nil {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return errors.New("invalid value")
	}

	if fieldInfo.Tag.Get("db") != "" {
		return setJSONField(field, fieldInfo, val)
	}

	return setStandardField(field, val)
}

func setJSONField(field reflect.Value, fieldInfo reflect.StructField, val reflect.Value) error {
	var data []byte
	switch v := val.Interface().(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for JSON field: %v", val.Type())
	}

	if field.Kind() == reflect.Interface {
		result, err := inferTypeFromJSON(data, fieldInfo.Name)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(result))
		return nil
	}

	if err := json.Unmarshal(data, field.Addr().Interface()); err != nil {
		if field.Kind() == reflect.Float32 {
			var jsonStr string
			if err := json.Unmarshal(data, &jsonStr); err == nil {
				if f, err := strconv.ParseFloat(jsonStr, 32); err == nil {
					field.SetFloat(f)
					return nil
				}
			}
		}
		return err
	}

	return nil
}

func setStandardField(field reflect.Value, val reflect.Value) error {
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}
	if val.Kind() == reflect.Interface && !val.IsNil() {
		val = val.Elem()
	}

	if !val.IsValid() {
		return errors.New("invalid value after dereferencing")
	}

	if field.Kind() == reflect.Ptr {
		if val.Type().ConvertibleTo(field.Type().Elem()) {
			newVal := reflect.New(field.Type().Elem())
			newVal.Elem().Set(val.Convert(field.Type().Elem()))
			field.Set(newVal)
			return nil
		}
		return fmt.Errorf("cannot convert %v to %v", val.Type(), field.Type())
	}

	if val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
		return nil
	}

	if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 && field.Kind() == reflect.String {
		field.SetString(string(val.Interface().([]byte)))
		return nil
	}

	return fmt.Errorf("cannot set field with value of type %v", val.Type())
}

func snakeToCamel(s string) string {
	if s == "" {
		return s
	}

	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}

	result := parts[0]
	caser := cases.Title(language.English)
	for i := 1; i < len(parts); i++ {
		result += caser.String(parts[i])
	}
	return result
}

func snakeToPascal(s string) string {
	if s == "" {
		return s
	}

	parts := strings.Split(s, "_")
	caser := cases.Title(language.English)
	var result string
	for _, part := range parts {
		result += caser.String(part)
	}
	return result
}

func camelToSnake(s string) string {
	var snake strings.Builder
	for i, c := range s {
		if unicode.IsUpper(c) {
			if i > 0 {
				snake.WriteRune('_')
			}
			snake.WriteRune(unicode.ToLower(c))
		} else {
			snake.WriteRune(c)
		}
	}
	return snake.String()
}

type dbExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func InsertStruct(db *sqlx.DB, tableName string, data any) error {
	var err error
	if db == nil {
		db, err = PostgresConn()
		if err != nil {
			return err
		}
		defer db.Close()
	}
	return insertStruct(db, tableName, data)
}

func InsertTrxStruct(db *sqlx.Tx, tableName string, data any) error {
	return insertStruct(db, tableName, data)
}

func insertStruct(executor dbExecutor, tableName string, data any) error {
	columns, values, err := extractFieldsForInsert(data)
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return fmt.Errorf("no columns to insert for table '%s'", tableName)
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s)`,
		tableName,
		strings.Join(columns, ", "),
		placeholders(len(values)),
	)

	if _, err := executor.Exec(query, values...); err != nil {
		return fmt.Errorf("failed to insert data into table '%s': %w", tableName, err)
	}

	return nil
}

func UpdateStruct(db *sqlx.DB, tableName string, data any, conditionField string, conditionValue any) error {
	var err error
	if db == nil {
		db, err = PostgresConn()
		if err != nil {
			return err
		}
		defer db.Close()
	}
	return updateStruct(db, tableName, data, conditionField, conditionValue)
}

func UpdateTrxStruct(db *sqlx.Tx, tableName string, data any, conditionField string, conditionValue any) error {
	return updateStruct(db, tableName, data, conditionField, conditionValue)
}

func updateStruct(executor dbExecutor, tableName string, data any, conditionField string, conditionValue any) error {
	columns, values, err := extractFieldsForUpdate(data, conditionField)
	if err != nil {
		return err
	}

	values = append(values, conditionValue)
	query := fmt.Sprintf(
		`UPDATE %s SET %s WHERE %s = $%d`,
		tableName,
		strings.Join(columns, ", "),
		camelToSnake(conditionField),
		len(values),
	)

	if _, err := executor.Exec(query, values...); err != nil {
		log.Printf("Failed to update record in table: %v. Query: %s, Values: %v", err, query, values)
		return err
	}

	return nil
}

func extractFieldsForInsert(data any) ([]string, []any, error) {
	elem, err := validateAndDereference(data)
	if err != nil {
		return nil, nil, err
	}

	typ := elem.Type()
	var columns []string
	var values []any

	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)
		column, skip := getColumnName(field)
		if skip {
			continue
		}

		fieldValue := elem.Field(i)
		value := extractFieldValue(field, fieldValue)

		columns = append(columns, column)
		values = append(values, value)
	}

	return columns, values, nil
}

func extractFieldsForUpdate(data any, conditionField string) ([]string, []any, error) {
	elem, err := validateAndDereference(data)
	if err != nil {
		return nil, nil, err
	}

	typ := elem.Type()
	var columns []string
	var values []any
	placeholderIdx := 1

	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)
		column, skip := getColumnName(field)
		if skip || column == conditionField {
			continue
		}

		fieldValue := elem.Field(i)
		value := extractFieldValue(field, fieldValue)

		columns = append(columns, fmt.Sprintf("%s = $%d", camelToSnake(column), placeholderIdx))
		values = append(values, value)
		placeholderIdx++
	}

	return columns, values, nil
}

func validateAndDereference(data any) (reflect.Value, error) {
	if data == nil {
		return reflect.Value{}, errors.New("data cannot be nil")
	}

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("data must be a pointer to a struct, but got %s", val.Kind())
	}

	if val.IsNil() {
		return reflect.Value{}, errors.New("data pointer is nil")
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("data must point to a struct, but points to %s", elem.Kind())
	}

	return elem, nil
}

func getColumnName(field reflect.StructField) (string, bool) {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		return "", true
	}

	if jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		return parts[0], false
	}

	return "", true
}

func extractFieldValue(field reflect.StructField, fieldValue reflect.Value) any {
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return nil
		}
		fieldValue = fieldValue.Elem()
	}

	if fieldValue.Kind() == reflect.Interface && !fieldValue.IsNil() {
		fieldValue = fieldValue.Elem()
	}

	dbTag := field.Tag.Get("db")
	needsJSON := fieldValue.Kind() == reflect.Slice ||
		fieldValue.Kind() == reflect.Map ||
		fieldValue.Kind() == reflect.Struct ||
		dbTag != ""

	if needsJSON {
		jsonBytes, err := json.Marshal(fieldValue.Interface())
		if err != nil {
			log.Printf("Failed to marshal JSON for field '%s': %v\n", field.Name, err)
			return nil
		}
		return string(jsonBytes)
	}

	return fieldValue.Interface()
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(n * 3)

	for i := 1; i <= n; i++ {
		if i > 1 {
			builder.WriteString(", ")
		}
		builder.WriteRune('$')
		builder.WriteString(strconv.Itoa(i))
	}

	return builder.String()
}
