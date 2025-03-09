package orm

import (
	"fmt"
	"strings"
)

// JoinType represents the type of SQL join
type JoinType string

const (
    InnerJoin JoinType = "INNER JOIN"
    LeftJoin  JoinType = "LEFT JOIN"
    RightJoin JoinType = "RIGHT JOIN"
    FullJoin  JoinType = "FULL JOIN"
)

// Join creates a join query with another table
func (o *ORM) Join(joinType JoinType, table string, on string) *JoinBuilder {
    // Start with the base table
    query := "SELECT * FROM " + o.tableName + " " + string(joinType) + " " + table + " ON " + on

    return &JoinBuilder{
        orm:      o,
        query:    query,
        joinType: joinType,
        tables:   []string{o.tableName, table},
    }
}

// Common shortcuts for different join types
func (o *ORM) InnerJoin(table string, on string) *JoinBuilder {
    return o.Join(InnerJoin, table, on)
}

func (o *ORM) LeftJoin(table string, on string) *JoinBuilder {
    return o.Join(LeftJoin, table, on)
}

func (o *ORM) RightJoin(table string, on string) *JoinBuilder {
    return o.Join(RightJoin, table, on)
}

func (o *ORM) FullJoin(table string, on string) *JoinBuilder {
    return o.Join(FullJoin, table, on)
}

// JoinBuilder helps construct and execute join queries
type JoinBuilder struct {
    orm      *ORM
    query    string
    args     []any
    joinType JoinType
    tables   []string
    where    string
    orderBy  string
    limit    int
    offset   int
}

// SimpleJoin provides a simpler syntax for the most common join case:
// joining two tables with matching field names
func (o *ORM) SimpleJoin(joinType JoinType, table string, field string) *JoinBuilder {
    // Construct the ON condition using the common field
    on := o.tableName + "." + field + " = " + table + "." + field
    return o.Join(joinType, table, on)
}

// JoinOnFields provides a convenient way to join tables when field names differ
func (o *ORM) JoinOnFields(joinType JoinType, table string, baseField string, joinField string) *JoinBuilder {
    // Construct the ON condition with different field names
    on := o.tableName + "." + baseField + " = " + table + "." + joinField
    return o.Join(joinType, table, on)
}

// AddJoin adds another join to the existing join query
func (jb *JoinBuilder) AddJoin(joinType JoinType, table string, on string) *JoinBuilder {
    jb.query += " " + string(joinType) + " " + table + " ON " + on
    jb.tables = append(jb.tables, table)
    return jb
}

// Where adds a WHERE clause to the join query
func (jb *JoinBuilder) Where(condition string, args ...any) *JoinBuilder {
    jb.where = condition
    jb.args = append(jb.args, args...)
    return jb
}

// OrderBy adds an ORDER BY clause to the join query
func (jb *JoinBuilder) OrderBy(orderBy string) *JoinBuilder {
    jb.orderBy = orderBy
    return jb
}

// Limit adds a LIMIT clause to the join query
func (jb *JoinBuilder) Limit(limit int) *JoinBuilder {
    jb.limit = limit
    return jb
}

// Offset adds an OFFSET clause to the join query
func (jb *JoinBuilder) Offset(offset int) *JoinBuilder {
    jb.offset = offset
    return jb
}

// Select specifies the columns to select
func (jb *JoinBuilder) Select(columns ...string) *JoinBuilder {
    // Replace the initial "SELECT *" with the specified columns
    selectClause := "SELECT " + strings.Join(columns, ", ")
    jb.query = strings.Replace(jb.query, "SELECT *", selectClause, 1)
    return jb
}

// Execute builds and executes the final join query
func (jb *JoinBuilder) Execute() *QueryResult {
    query := jb.query

    // Add WHERE clause if specified
    if jb.where != "" {
        query += " WHERE " + jb.where
    }

    // Add ORDER BY clause if specified
    if jb.orderBy != "" {
        query += " ORDER BY " + jb.orderBy
    }

    // Add LIMIT clause if specified
    if jb.limit > 0 {
        query += " LIMIT " + fmt.Sprintf("%d", jb.limit)
    }

    // Add OFFSET clause if specified
    if jb.offset > 0 {
        query += " OFFSET " + fmt.Sprintf("%d", jb.offset)
    }

    // Execute the query
    return jb.orm.QueryRaw(query, jb.args...)
}
