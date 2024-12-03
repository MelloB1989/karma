package orm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/MelloB1989/karma/database"
	"github.com/jmoiron/sqlx"
)

type ORM struct {
	DB     *sqlx.DB
	Schema *Schema
}

// getAll: Fetch all records
func (o *ORM) GetAll(result interface{}) error {
	// Validate that result is a pointer to a slice
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result must be a pointer to a slice")
	}

	// Get the element type of the slice
	elemType := resultValue.Elem().Type().Elem()

	// Create a new instance of the element type to access the embedded Model
	instance := reflect.New(elemType).Interface()

	// Check if the struct embeds the Model and access TableName
	modelValue := reflect.ValueOf(instance).Elem().FieldByName("Model")
	if !modelValue.IsValid() {
		return fmt.Errorf("struct must embed the Model struct")
	}

	tableName := modelValue.FieldByName("TableName").String()
	if tableName == "" {
		return fmt.Errorf("table name is required in the Model struct")
	}

	// Build and execute the query
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	fmt.Println("Executing Query:", query)

	rows, err := o.DB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Use ParseRows to populate the result slice
	if err := database.ParseRows(rows, result); err != nil {
		return fmt.Errorf("failed to parse rows: %w", err)
	}

	return nil
}

// insert: Insert a record
func (o *ORM) Insert(model interface{}) error {
	fields := []string{}
	values := []interface{}{}
	placeholders := []string{}

	v := reflect.ValueOf(model).Elem()
	for _, field := range o.Schema.Fields {
		fieldValue := v.FieldByName(field.Name).Interface()
		if field.IsEmbedded {
			// Serialize embedded fields
			embeddedJSON, err := json.Marshal(fieldValue)
			if err != nil {
				return err
			}
			fieldValue = string(embeddedJSON)
		}
		fields = append(fields, field.Tag)
		values = append(values, fieldValue)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(placeholders)+1))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		o.Schema.TableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := o.DB.Exec(query, values...)
	return err
}
