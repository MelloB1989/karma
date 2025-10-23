# Karma ORM - LLM Documentation

Concise reference for Go ORM with caching, CRUD operations, joins, and transactions.

## Initialization

Package: "github.com/MelloB1989/karma/v2/orm"

```go
// Load ORM with struct
orm := orm.Load(&YourStruct{}, options...)

// Options:
orm.WithDatabasePrefix(prefix string)           // Set DB connection prefix
orm.WithCacheOn(true)                           // Enable caching (Redis + memory)
orm.WithCacheMethod("redis|memory|both")        // Cache storage method
orm.WithCacheKey(key string)                    // Cache key prefix
orm.WithCacheTTL(duration time.Duration)        // Cache expiration
orm.WithInfiniteCacheTTL()                      // Never expire cache
orm.WithRedisClient(client *redis.Client)       // Custom Redis client
```

### Struct Definition
```go
type User struct {
    TableName string `karma_table:"users"` // REQUIRED: table name tag
    ID        int    `json:"id" karma:"primary_key"`
    Username  string `json:"username"`
    Email     string `json:"email"`
    Age       int    `json:"age"`
}
```

## CRUD Operations

### Create (Insert)
```go
orm.Insert(entity any) error
// Example:
user := &User{Username: "john", Email: "john@example.com", Age: 30}
err := userORM.Insert(user)
```

### Read (Query)

**Get All Records:**
```go
GetAll() *QueryResult
// Example: userORM.GetAll().Scan(&users)
```

**Get by Primary Key:**
```go
GetByPrimaryKey(value any) *QueryResult
// Example: userORM.GetByPrimaryKey(1).Scan(&user)
```

**Get by Field Equals:**
```go
GetByFieldEquals(fieldName string, value any) *QueryResult
// Example: userORM.GetByFieldEquals("Username", "john").Scan(&users)
```

**Get by Multiple Fields:**
```go
GetByFieldsEquals(fieldValueMap map[string]any) *QueryResult
// Example: userORM.GetByFieldsEquals(map[string]any{"Username": "john", "Age": 30}).Scan(&users)
```

**Get by Field IN:**
```go
GetByFieldIn(fieldName string, values ...any) *QueryResult
// Example: userORM.GetByFieldIn("ID", 1, 2, 3).Scan(&users)
```

**Get by Field LIKE:**
```go
GetByFieldLike(fieldName string, pattern string) *QueryResult
// Example: userORM.GetByFieldLike("Email", "%@gmail.com").Scan(&users)
```

**Comparison Operators:**
```go
GetByFieldGreaterThan(fieldName string, value any) *QueryResult
GetByFieldLessThan(fieldName string, value any) *QueryResult
GetByFieldGreaterThanEquals(fieldName string, value any) *QueryResult
GetByFieldLessThanEquals(fieldName string, value any) *QueryResult
GetByFieldCompare(fieldName string, operator string, value any) *QueryResult
// Valid operators: ">", "<", ">=", "<=", "=", "!=", "<>"
// Example: userORM.GetByFieldGreaterThan("Age", 25).Scan(&users)
```

**NULL Checks:**
```go
GetByFieldIsNull(fieldName string) *QueryResult
GetByFieldIsNotNull(fieldName string) *QueryResult
// Example: userORM.GetByFieldIsNull("Email").Scan(&users)
```

**BETWEEN:**
```go
GetByFieldBetween(fieldName string, start any, end any) *QueryResult
// Example: userORM.GetByFieldBetween("Age", 20, 40).Scan(&users)
```

**NOT Equals:**
```go
GetByFieldNotEquals(fieldName string, value any) *QueryResult
// Example: userORM.GetByFieldNotEquals("Status", "deleted").Scan(&users)
```

**Order By:**
```go
OrderBy(fieldName string, direction OrderDirection) *QueryResult
// Directions: orm.OrderAsc, orm.OrderDesc
// Example: userORM.OrderBy("Age", orm.OrderDesc).Scan(&users)
```

**Count Records:**
```go
GetCount(filters map[string]any) (int, error)
GetTotalRowCount() (int, error)
// Example: count, err := userORM.GetCount(map[string]any{"Age": 30})
```

### Update
```go
Update(entity any, primaryKeyValue string) error
// Example:
user.Email = "newemail@example.com"
err := userORM.Update(user, "1")
```

### Delete

**Delete by Field:**
```go
DeleteByFieldEquals(fieldName string, value any) (int64, error)
// Example: rowsDeleted, err := userORM.DeleteByFieldEquals("Username", "john")
```

**Delete by Comparison:**
```go
DeleteByFieldCompare(fieldName string, value any, operator string) (int64, error)
// Operators: "=", ">", "<", ">=", "<=", "LIKE"
// Example: rowsDeleted, err := userORM.DeleteByFieldCompare("Age", 30, ">")
```

**Delete by IN:**
```go
DeleteByFieldIn(fieldName string, values []any) (int64, error)
// Example: rowsDeleted, err := userORM.DeleteByFieldIn("ID", []any{1, 2, 3})
```

**Delete by Primary Key:**
```go
DeleteByPrimaryKey(pkValue any) (int64, error)
// Example: rowsDeleted, err := userORM.DeleteByPrimaryKey(10)
```

**Delete All:**
```go
DeleteAll() (int64, error)
// Example: rowsDeleted, err := userORM.DeleteAll()
```

## Raw Queries

```go
QueryRaw(query string, args ...any) *QueryResult
ExecuteRaw(query string, args ...any) (sql.Result, error)

// Example:
userORM.QueryRaw("SELECT * FROM users WHERE age > $1", 25).Scan(&users)
result, err := userORM.ExecuteRaw("UPDATE users SET active = $1 WHERE id = $2", true, 1)
```

## Scanning Results

```go
var user User
var users []User

// Single record
err := userORM.GetByPrimaryKey(1).Scan(&user)

// Multiple records
err := userORM.GetAll().Scan(&users)

// Check errors
if err != nil {
    // Handle error
}
```

## Joins

### Join Types
- `InnerJoin` - INNER JOIN
- `LeftJoin` - LEFT JOIN
- `RightJoin` - RIGHT JOIN
- `FullJoin` - FULL JOIN

### Basic Join
```go
userORM.Join(orm.InnerJoin, "orders", "users.id = orders.user_id").Execute().Scan(&results)

// Shortcuts:
userORM.InnerJoin("orders", "users.id = orders.user_id").Execute().Scan(&results)
userORM.LeftJoin("orders", "users.id = orders.user_id").Execute().Scan(&results)
```

### Simple Join (same field name)
```go
SimpleJoin(joinType JoinType, table string, field string) *JoinBuilder
// Example: userORM.SimpleJoin(orm.InnerJoin, "profiles", "user_id").Execute().Scan(&results)
```

### Join with Different Field Names
```go
JoinOnFields(joinType JoinType, table string, baseField string, joinField string) *JoinBuilder
// Example: userORM.JoinOnFields(orm.LeftJoin, "orders", "id", "user_id").Execute().Scan(&results)
```

### Join Builder Methods
```go
jb := userORM.InnerJoin("orders", "users.id = orders.user_id")

// Chain additional joins
jb.AddJoin(orm.LeftJoin, "products", "orders.product_id = products.id")

// Add WHERE clause
jb.Where("users.age > $1", 25)

// Select specific columns
jb.Select("users.username", "orders.total", "products.name")

// Order results
jb.OrderBy("orders.created_at DESC")

// Pagination
jb.Limit(10).Offset(20)

// Execute
jb.Execute().Scan(&results)
```

### Complete Join Example
```go
type UserOrder struct {
    Username  string
    OrderID   int
    Total     float64
}

var results []UserOrder
err := userORM.
    InnerJoin("orders", "users.id = orders.user_id").
    Select("users.username", "orders.id AS order_id", "orders.total").
    Where("orders.total > $1", 100).
    OrderBy("orders.total DESC").
    Limit(10).
    Execute().
    Scan(&results)
```

## Transactions

### Manual Transaction
```go
tx, err := orm.Begin()
if err != nil {
    return err
}

// Use tx.ORM() for operations
err = tx.ORM().Insert(user)
if err != nil {
    tx.Rollback()
    return err
}

err = tx.ORM().Update(order, "123")
if err != nil {
    tx.Rollback()
    return err
}

// Commit
err = tx.Commit()
```

### Transaction with Context
```go
ctx := context.Background()
tx, err := orm.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
// ... use tx.ORM() ...
tx.Commit() or tx.Rollback()
```

### Automatic Transaction (Recommended)
```go
err := userORM.WithTransaction(func(txOrm *ORM) error {
    if err := txOrm.Insert(user); err != nil {
        return err // auto-rollback
    }
    if err := txOrm.Update(profile, "1"); err != nil {
        return err // auto-rollback
    }
    return nil // auto-commit
})
```

## Caching

### Cache Configuration
```go
// Enable caching with default settings (Redis + Memory)
orm := orm.Load(&User{}, orm.WithCacheOn(true))

// Memory-only cache
orm := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("memory"))

// Redis-only cache
orm := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("redis"))

// Custom TTL (5 minutes)
orm := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheTTL(5 * time.Minute))

// Infinite TTL (never expire)
orm := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithInfiniteCacheTTL())

// Custom cache key prefix
orm := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheKey("myapp:users"))
```

### Cache Invalidation

**Invalidate Specific Query:**
```go
err := orm.InvalidateCache(query, args...)
// Example:
query := "SELECT * FROM users WHERE id = $1"
orm.InvalidateCache(query, 1)
```

**Invalidate by Prefix:**
```go
err := orm.InvalidateCacheByPrefix(prefix string)
// Example: orm.InvalidateCacheByPrefix("myapp:users")
```

**Clear All Cache:**
```go
err := orm.ClearCache(clearRedis bool)
// clearRedis=true: clears both memory and Redis
// clearRedis=false: clears memory only
// Example: orm.ClearCache(true)
```

### Cache Constants
```go
const InfiniteTTL = time.Duration(-1) // Never expire

// Check if TTL is infinite
IsInfiniteTTL(ttl time.Duration) bool
orm.HasInfiniteTTL() bool
```

## Query Result Methods

```go
type QueryResult struct {
    // Methods:
    Scan(dest any) error          // Map results to destination
    GetQuery() string             // Get executed query
    GetArgs() []any               // Get query arguments
}

// Example:
qr := userORM.GetAll()
query := qr.GetQuery()    // "SELECT * FROM users"
args := qr.GetArgs()      // []
err := qr.Scan(&users)
```

## Common Patterns

### Find User by Email
```go
var user User
err := userORM.GetByFieldEquals("Email", "john@example.com").Scan(&user)
```

### Get Users with Age Range
```go
var users []User
err := userORM.GetByFieldBetween("Age", 18, 65).Scan(&users)
```

### Paginated Query
```go
var users []User
err := userORM.
    OrderBy("CreatedAt", orm.OrderDesc).
    QueryRaw("SELECT * FROM users LIMIT $1 OFFSET $2", 10, 20).
    Scan(&users)
```

### Cached Query with Prefix
```go
orm := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheKey("users:active"),
    orm.WithCacheTTL(10 * time.Minute))

var users []User
err := orm.GetByFieldEquals("Active", true).Scan(&users)
// Cached as "users:active:<hash>"
```

### Bulk Delete
```go
ids := []any{1, 2, 3, 4, 5}
deleted, err := userORM.DeleteByFieldIn("ID", ids)
fmt.Printf("Deleted %d records\n", deleted)
```

## Error Handling

All query methods return `*QueryResult`. Check errors after `Scan()`:

```go
result := userORM.GetByPrimaryKey(1)
var user User
if err := result.Scan(&user); err != nil {
    // Handle error
}
```

For direct operations, check returned error:

```go
if err := userORM.Insert(user); err != nil {
    // Handle error
}

deleted, err := userORM.DeleteByPrimaryKey(1)
if err != nil {
    // Handle error
}
```

## Notes

- Field names in methods refer to struct field names (e.g., "Username"), not column names
- Use `karma:"primary_key"` tag to mark primary key field
- `karma_table` tag is REQUIRED on TableName field
- Caching automatically uses SHA-256 hashed keys for query/args combination
- All queries support PostgreSQL placeholder syntax ($1, $2, etc.)
- Transaction operations use the same CRUD methods via `tx.ORM()`
