package orm

// import (
// 	"fmt"
// 	"log"
// 	"reflect"

// 	"github.com/MelloB1989/karma/database"
// )

// // ORM struct encapsulates the metadata and methods for a table.
// type ORM struct {
// 	tableName  string
// 	structType reflect.Type
// 	fieldMap   map[string]string
// }

// // Load initializes the ORM with the given struct.
// func Load(entity any) *ORM {
// 	t := reflect.TypeOf(entity).Elem() // Get the type of the struct
// 	tableName := ""

// 	// Get the table name from the struct tag
// 	if field, ok := t.FieldByName("TableName"); ok {
// 		tableName = field.Tag.Get("karma_table")
// 	}

// 	// Build the field mapping
// 	fieldMap := make(map[string]string)
// 	for i := 0; i < t.NumField(); i++ {
// 		field := t.Field(i)
// 		jsonTag := field.Tag.Get("json")
// 		if jsonTag != "" {
// 			fieldMap[field.Name] = jsonTag
// 		} else {
// 			fieldMap[field.Name] = field.Name
// 		}
// 	}

// 	return &ORM{
// 		tableName:  tableName,
// 		structType: t,
// 		fieldMap:   fieldMap,
// 	}
// }

// // QueryResult holds the result of a query operation and any error that occurred
// type QueryResult struct {
// 	result any
// 	err    error
// }

// // Scan maps the query result to the provided destination pointer
// func (qr *QueryResult) Scan(dest any) error {
// 	if qr.err != nil {
// 		return qr.err
// 	}

// 	// Verify that dest is a pointer
// 	destValue := reflect.ValueOf(dest)
// 	if destValue.Kind() != reflect.Ptr {
// 		return fmt.Errorf("destination must be a pointer")
// 	}

// 	// Get the element the pointer points to
// 	destElem := destValue.Elem()

// 	// Check if queryResult is nil
// 	if qr.result == nil {
// 		return fmt.Errorf("query result is nil")
// 	}

// 	resultValue := reflect.ValueOf(qr.result)

// 	// Check if the query returned a slice of struct pointers
// 	if resultValue.Kind() == reflect.Slice {
// 		// Handle slice destination (multiple results)
// 		if destElem.Kind() == reflect.Slice {
// 			// Clear the destination slice
// 			destElem.Set(reflect.MakeSlice(destElem.Type(), 0, resultValue.Len()))

// 			// Get the element type of the destination slice
// 			destElemType := destElem.Type().Elem()
// 			isDestPtr := destElemType.Kind() == reflect.Ptr

// 			// If dest slice element is not a pointer, get the underlying type
// 			var destStructType reflect.Type
// 			if isDestPtr {
// 				destStructType = destElemType.Elem()
// 			} else {
// 				destStructType = destElemType
// 			}

// 			// Iterate through the result slice
// 			for i := 0; i < resultValue.Len(); i++ {
// 				// Get the source struct pointer
// 				srcPtr := resultValue.Index(i)

// 				var newItem reflect.Value
// 				if isDestPtr {
// 					// Create a new pointer to struct
// 					newItem = reflect.New(destStructType)
// 					// Copy fields from source to destination
// 					copyStructFields(srcPtr.Elem(), newItem.Elem())
// 				} else {
// 					// Create a new struct value
// 					newItem = reflect.New(destStructType).Elem()
// 					// Copy fields from source to destination
// 					copyStructFields(srcPtr.Elem(), newItem)
// 				}

// 				// Append the new item to the destination slice
// 				destElem.Set(reflect.Append(destElem, newItem))
// 			}
// 			return nil
// 		} else if destElem.Kind() == reflect.Struct {
// 			// Handle single struct destination with multiple results
// 			// Just use the first result if there's any
// 			if resultValue.Len() > 0 {
// 				srcPtr := resultValue.Index(0)
// 				copyStructFields(srcPtr.Elem(), destElem)
// 				return nil
// 			}
// 			return fmt.Errorf("no results to scan into struct")
// 		} else {
// 			return fmt.Errorf("destination must be a pointer to slice or struct, got %v", destElem.Kind())
// 		}
// 	} else if resultValue.Kind() == reflect.Ptr && resultValue.Elem().Kind() == reflect.Struct {
// 		// Handle single struct result into single struct destination
// 		if destElem.Kind() == reflect.Struct {
// 			copyStructFields(resultValue.Elem(), destElem)
// 			return nil
// 		} else {
// 			return fmt.Errorf("destination must be a pointer to struct to scan single result")
// 		}
// 	}

// 	return fmt.Errorf("unsupported query result type: %v", resultValue.Type())
// }

// // QueryRaw to return a QueryResult for chaining
// func (o *ORM) QueryRaw(sqlQuery string, args ...any) *QueryResult {
// 	// Establish database connection
// 	db, err := database.PostgresConn()
// 	if err != nil {
// 		log.Println("DB connection error:", err)
// 		return &QueryResult{nil, err}
// 	}
// 	defer db.Close()

// 	// Execute the query
// 	rows, err := db.Query(sqlQuery, args...)
// 	if err != nil {
// 		log.Println("Query execution error:", err)
// 		return &QueryResult{nil, err}
// 	}
// 	defer rows.Close()

// 	// Retrieve column names from the result
// 	columns, err := rows.Columns()
// 	if err != nil {
// 		log.Println("Failed to retrieve columns:", err)
// 		return &QueryResult{nil, err}
// 	}

// 	// Reverse fieldMap to map columns to struct fields
// 	columnToField := make(map[string]string) // column name -> field name
// 	for field, column := range o.fieldMap {
// 		columnToField[column] = field
// 	}

// 	// Prepare a slice to hold the results
// 	sliceType := reflect.SliceOf(reflect.PointerTo(o.structType)) // []*StructType
// 	results := reflect.MakeSlice(sliceType, 0, 0)

// 	// Iterate over the rows
// 	for rows.Next() {
// 		// Create a new instance of the struct
// 		structPtr := reflect.New(o.structType) // *StructType
// 		structVal := structPtr.Elem()          // StructType

// 		// Prepare a slice for Scan destination pointers
// 		scanDest := make([]any, len(columns))
// 		for i, col := range columns {
// 			if fieldName, ok := columnToField[col]; ok {
// 				field := structVal.FieldByName(fieldName)
// 				if !field.IsValid() {
// 					// Field not found; use a dummy variable
// 					var dummy any
// 					scanDest[i] = &dummy
// 				} else {
// 					scanDest[i] = field.Addr().Interface()
// 				}
// 			} else {
// 				// Column does not map to any struct field; use a dummy variable
// 				var dummy any
// 				scanDest[i] = &dummy
// 			}
// 		}

// 		// Scan the row into the struct fields
// 		if err := rows.Scan(scanDest...); err != nil {
// 			log.Println("Failed to scan row:", err)
// 			return &QueryResult{nil, err}
// 		}

// 		// Append the struct pointer to the results slice
// 		results = reflect.Append(results, structPtr)
// 	}

// 	// Check for errors from iterating over rows
// 	if err := rows.Err(); err != nil {
// 		log.Println("Rows iteration error:", err)
// 		return &QueryResult{nil, err}
// 	}

// 	return &QueryResult{results.Interface(), nil}
// }

// // copyStructFields copies field values from src to dest where field names match
// func copyStructFields(src, dest reflect.Value) {
// 	// Ensure both are structs
// 	if src.Kind() != reflect.Struct || dest.Kind() != reflect.Struct {
// 		return
// 	}

// 	destType := dest.Type()
// 	for i := range make([]int, destType.NumField()) {
// 		destField := destType.Field(i)

// 		// Skip unexported fields
// 		if destField.PkgPath != "" {
// 			continue
// 		}

// 		// Find matching field in source struct
// 		srcField := src.FieldByName(destField.Name)
// 		if !srcField.IsValid() {
// 			continue
// 		}

// 		// Check if types are compatible for assignment
// 		destFieldValue := dest.Field(i)
// 		if srcField.Type().AssignableTo(destFieldValue.Type()) && destFieldValue.CanSet() {
// 			destFieldValue.Set(srcField)
// 		}
// 	}
// }
