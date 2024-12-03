package orm

import (
	"reflect"
	"strings"
)

type FieldInfo struct {
	Name       string
	Tag        string
	IsPrimary  bool
	IsUnique   bool
	IsForeign  bool
	ForeignKey string
	IsEmbedded bool
}

type Schema struct {
	TableName  string
	Fields     []FieldInfo
	PrimaryKey string
}

type Model struct {
	TableName  string
	PrimaryKey string
}

var schemaRegistry = make(map[string]*Schema)

func Load(model interface{}) *Schema {
	// Ensure model is a pointer
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Retrieve the embedded Model field
	meta := v.FieldByName("Model").Interface().(Model)

	if meta.TableName == "" {
		panic("Table name is required in Model")
	}
	if meta.PrimaryKey == "" {
		panic("Primary key is required in Model")
	}

	schema := &Schema{
		TableName:  meta.TableName,
		PrimaryKey: meta.PrimaryKey,
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "Model" {
			continue
		}

		fieldInfo := FieldInfo{
			Name: field.Name,
			Tag:  field.Tag.Get("json"),
		}

		// Parse custom tags
		tag := field.Tag.Get("karma")
		if strings.Contains(tag, "primary_key") {
			fieldInfo.IsPrimary = true
		}
		if strings.Contains(tag, "unique") {
			fieldInfo.IsUnique = true
		}
		if strings.Contains(tag, "foreign") {
			fieldInfo.IsForeign = true
			foreignKey := strings.Split(strings.Split(tag, ":")[1], ";")[0]
			fieldInfo.ForeignKey = foreignKey
		}
		if strings.Contains(tag, "embedded_json") {
			fieldInfo.IsEmbedded = true
		}

		schema.Fields = append(schema.Fields, fieldInfo)
	}

	schemaRegistry[schema.TableName] = schema
	return schema
}
