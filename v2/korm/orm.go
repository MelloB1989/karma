package korm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/MelloB1989/karma/database"
)

// Execute executes the query and returns a QueryResult
func (q *QueryBuilder) Execute() *QueryResult {
	if q.rawQuery != "" {
		return q.orm.QueryRaw(q.rawQuery, q.rawArgs...)
	}

	query, args := q.buildQuery()
	return q.orm.QueryRaw(query, args...)
}

// Select initiates a SELECT query
func (o *ORM) Select(fields ...string) *QueryBuilder {
	selectFields := fields
	if len(fields) == 0 {
		selectFields = []string{"*"}
	}

	return &QueryBuilder{
		orm:          o,
		operation:    "SELECT",
		selectFields: selectFields,
		limit:        -1,
		offset:       -1,
	}
}

// Count initiates a SELECT COUNT(*) query
func (o *ORM) Count() *QueryBuilder {
	return &QueryBuilder{
		orm:       o,
		operation: "SELECT",
		isCount:   true,
		limit:     -1,
		offset:    -1,
	}
}

// Delete initiates a DELETE query
func (o *ORM) Delete() *QueryBuilder {
	return &QueryBuilder{
		orm:       o,
		operation: "DELETE",
		limit:     -1,
		offset:    -1,
	}
}

// Raw sets a raw SQL query with arguments
func (o *ORM) Raw(query string, args ...any) *QueryBuilder {
	return &QueryBuilder{
		orm:      o,
		rawQuery: query,
		rawArgs:  args,
	}
}

// Where adds a condition to the query
func (q *QueryBuilder) Where(field string, operator Operator, value any) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	q.conditions = append(q.conditions, Condition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})

	return q
}

// WhereIn adds an IN condition to the query
func (q *QueryBuilder) WhereIn(field string, values ...any) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	normalizedValues, err := q.orm.normalizeValues(values...)
	if err == nil {
		q.conditions = append(q.conditions, Condition{
			Field:    field,
			Operator: In,
			Values:   normalizedValues,
		})
	}

	return q
}

// WhereBetween adds a BETWEEN condition to the query
func (q *QueryBuilder) WhereBetween(field string, start, end any) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	q.conditions = append(q.conditions, Condition{
		Field:    field,
		Operator: Between,
		Values:   []any{start, end},
	})

	return q
}

// WhereNull adds an IS NULL condition to the query
func (q *QueryBuilder) WhereNull(field string) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	q.conditions = append(q.conditions, Condition{
		Field:    field,
		Operator: IsNull,
	})

	return q
}

// WhereNotNull adds an IS NOT NULL condition to the query
func (q *QueryBuilder) WhereNotNull(field string) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	q.conditions = append(q.conditions, Condition{
		Field:    field,
		Operator: IsNotNull,
	})

	return q
}

// OrderBy adds an ORDER BY clause to the query
func (q *QueryBuilder) OrderBy(field string, direction OrderDirection) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	q.orders = append(q.orders, Order{
		Field:     field,
		Direction: direction,
	})

	return q
}

// Join adds a JOIN clause to the query
func (q *QueryBuilder) Join(tableName string, joinType JoinType, conditions ...Condition) *QueryBuilder {
	q.joins = append(q.joins, Join{
		TableName:  tableName,
		Type:       joinType,
		Conditions: conditions,
	})

	return q
}

// GroupBy adds a GROUP BY clause to the query
func (q *QueryBuilder) GroupBy(fields ...string) *QueryBuilder {
	resolvedFields := make([]string, 0, len(fields))

	for _, field := range fields {
		columnName, err := q.orm.resolveColumn(field)
		if err == nil {
			resolvedFields = append(resolvedFields, columnName)
		} else {
			resolvedFields = append(resolvedFields, field)
		}
	}

	q.groupByFields = append(q.groupByFields, resolvedFields...)
	return q
}

// Having adds a HAVING clause to the query
func (q *QueryBuilder) Having(field string, operator Operator, value any) *QueryBuilder {
	columnName, err := q.orm.resolveColumn(field)
	if err == nil {
		field = columnName
	}

	q.havingConds = append(q.havingConds, Condition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})

	return q
}

// Limit adds a LIMIT clause to the query
func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
	q.limit = limit
	return q
}

// Offset adds an OFFSET clause to the query
func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
	q.offset = offset
	return q
}

// buildQuery constructs the SQL query string and argument list
func (q *QueryBuilder) buildQuery() (string, []any) {
	var query strings.Builder
	var args []any
	argIndex := 1

	// Build the operation part
	if q.operation == "SELECT" {
		query.WriteString("SELECT ")
		if q.isCount {
			query.WriteString("COUNT(*)")
		} else {
			query.WriteString(strings.Join(q.selectFields, ", "))
		}
		query.WriteString(" FROM ")
		query.WriteString(q.orm.tableName)
	} else if q.operation == "DELETE" {
		query.WriteString("DELETE FROM ")
		query.WriteString(q.orm.tableName)
	}

	// Add JOIN clauses
	for _, join := range q.joins {
		query.WriteString(" ")
		query.WriteString(string(join.Type))
		query.WriteString(" ")
		query.WriteString(join.TableName)
		query.WriteString(" ON ")

		for i, cond := range join.Conditions {
			if i > 0 {
				query.WriteString(" AND ")
			}

			query.WriteString(cond.Field)
			query.WriteString(" ")
			query.WriteString(string(cond.Operator))
			query.WriteString(" ")

			if cond.Operator == IsNull || cond.Operator == IsNotNull {
				// No placeholder needed
			} else {
				query.WriteString(fmt.Sprintf("$%d", argIndex))
				args = append(args, cond.Value)
				argIndex++
			}
		}
	}

	// Add WHERE clause
	if len(q.conditions) > 0 {
		query.WriteString(" WHERE ")

		for i, cond := range q.conditions {
			if i > 0 {
				query.WriteString(" AND ")
			}

			query.WriteString(cond.Field)
			query.WriteString(" ")

			switch cond.Operator {
			case IsNull, IsNotNull:
				query.WriteString(string(cond.Operator))
			case In:
				query.WriteString(string(cond.Operator))
				query.WriteString(" (")
				placeholders := make([]string, len(cond.Values))
				for j := range cond.Values {
					placeholders[j] = fmt.Sprintf("$%d", argIndex)
					args = append(args, cond.Values[j])
					argIndex++
				}
				query.WriteString(strings.Join(placeholders, ", "))
				query.WriteString(")")
			case Between:
				query.WriteString(string(cond.Operator))
				query.WriteString(fmt.Sprintf(" $%d AND $%d", argIndex, argIndex+1))
				args = append(args, cond.Values[0], cond.Values[1])
				argIndex += 2
			default:
				query.WriteString(string(cond.Operator))
				query.WriteString(fmt.Sprintf(" $%d", argIndex))
				args = append(args, cond.Value)
				argIndex++
			}
		}
	}

	// Add GROUP BY clause
	if len(q.groupByFields) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(q.groupByFields, ", "))
	}

	// Add HAVING clause
	if len(q.havingConds) > 0 {
		query.WriteString(" HAVING ")

		for i, cond := range q.havingConds {
			if i > 0 {
				query.WriteString(" AND ")
			}

			query.WriteString(cond.Field)
			query.WriteString(" ")
			query.WriteString(string(cond.Operator))
			query.WriteString(" ")

			if cond.Operator == IsNull || cond.Operator == IsNotNull {
				// No placeholder needed
			} else {
				query.WriteString(fmt.Sprintf("$%d", argIndex))
				args = append(args, cond.Value)
				argIndex++
			}
		}
	}

	// Add ORDER BY clause
	if len(q.orders) > 0 {
		query.WriteString(" ORDER BY ")

		orderClauses := make([]string, len(q.orders))
		for i, order := range q.orders {
			orderClauses[i] = fmt.Sprintf("%s %s", order.Field, order.Direction)
		}

		query.WriteString(strings.Join(orderClauses, ", "))
	}

	// Add LIMIT clause
	if q.limit >= 0 {
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
		args = append(args, q.limit)
		argIndex++
	}

	// Add OFFSET clause
	if q.offset >= 0 {
		query.WriteString(fmt.Sprintf(" OFFSET $%d", argIndex))
		args = append(args, q.offset)
		argIndex++
	}

	return query.String(), args
}

// First executes the query and returns the first result
func (q *QueryBuilder) First() (*QueryResult, error) {
	q.limit = 1
	result := q.Execute()
	if result.err != nil {
		return nil, result.err
	}
	return result, nil
}

// Get executes the query and returns all results
func (q *QueryBuilder) Get() *QueryResult {
	return q.Execute()
}

// ScanAll scans all rows into a slice of the given struct type
func (qr *QueryResult) ScanAll(dest any) error {
	if qr.err != nil {
		return qr.err
	}
	defer qr.rows.Close()

	// Check if destination is a pointer to a slice
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return errors.New("destination must be a pointer to a slice")
	}

	// Get the type of the slice elements
	sliceType := destValue.Elem().Type()
	elemType := sliceType.Elem()

	// Create a slice to hold the results
	results := reflect.MakeSlice(sliceType, 0, 0)

	// Scan each row
	for qr.rows.Next() {
		// Create a new instance of the element type
		elemPtr := reflect.New(elemType)

		// Scan the row into the element
		err := database.ParseRows(qr.rows, elemPtr.Interface())
		if err != nil {
			return err
		}

		// Append the element to the results slice
		results = reflect.Append(results, elemPtr.Elem())
	}

	// Check for errors from iterating over rows
	if err := qr.rows.Err(); err != nil {
		return err
	}

	// Set the destination to the results
	destValue.Elem().Set(results)
	return nil
}

// Value returns the value of a single field from the first row
func (qr *QueryResult) Value() (any, error) {
	if qr.err != nil {
		return nil, qr.err
	}
	defer qr.rows.Close()

	// Check if there's a row to scan
	if !qr.rows.Next() {
		return nil, sql.ErrNoRows
	}

	// Scan the first column
	var value any
	err := qr.rows.Scan(&value)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// Insert inserts a new row into the table
func (o *ORM) Insert(entity any) error {
	if o.tx != nil {
		return database.InsertTrxStruct(o.tx, o.tableName, entity)
	} else {
		return database.InsertStruct(o.db, o.tableName, entity)
	}
}

// Update updates an existing row in the table
func (o *ORM) Update(entity any, primaryKeyValue any) error {
	primaryField := o.getPrimaryKeyField()
	if primaryField == "" {
		return errors.New("primary key not defined in struct")
	}

	if o.tx != nil {
		return database.UpdateTrxStruct(o.tx, o.tableName, entity, primaryField, primaryKeyValue)
	} else {
		return database.UpdateStruct(o.db, o.tableName, entity, primaryField, primaryKeyValue)
	}
}

// FindByPK finds a record by its primary key
func (o *ORM) FindByPK(value any) *QueryBuilder {
	pkField := o.getPrimaryKeyField()
	if pkField == "" {
		return &QueryBuilder{
			orm:      o,
			rawQuery: "",
			rawArgs:  nil,
		}
	}

	return o.Select().Where(pkField, Equals, value)
}

// Close closes the database connection
func (o *ORM) Close() error {
	if o.db != nil {
		return o.db.Close()
	}
	return nil
}
