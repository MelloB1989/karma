package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/database"
	"github.com/MelloB1989/karma/utils"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// ORM struct encapsulates the metadata and methods for a table.
type ORM struct {
	tableName   string
	structType  reflect.Type
	fieldMap    map[string]string
	tx          *sqlx.Tx
	db          *sqlx.DB
	CacheOn     *bool
	CacheMethod *string
	CacheKey    *string
	CacheTTL    *time.Duration
	RedisClient *redis.Client
}

type QueryResult struct {
	rows       *sql.Rows
	err        error
	query      string
	args       []any
	cachedData []byte
	orm        any
}

// encodeArgs serializes query arguments to create a unique cache key
func encodeArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}

	encoded, err := json.Marshal(args)
	if err != nil {
		log.Printf("Failed to encode query args: %v", err)
		return ""
	}

	return string(encoded)
}

// generateCacheKey creates a unique key for Redis based on the query and arguments
func (o *ORM) generateCacheKey(query string, args []any) string {
	prefix := ""
	if o.CacheKey != nil {
		prefix = *o.CacheKey + ":"
	}

	// Create a unique key by hashing the query and its arguments
	argsStr := encodeArgs(args)
	queryKey := query + argsStr
	hashedKey := utils.Sha256Sum(queryKey)

	return prefix + hashedKey
}

type Options func(*ORM)

// Load initializes the ORM with the given struct.
func Load(entity any, opts ...Options) *ORM {
	if entity == nil {
		log.Printf("Error: entity cannot be nil")
		return nil
	}

	entityType := reflect.TypeOf(entity)
	if entityType.Kind() != reflect.Ptr {
		log.Printf("Error: entity must be a pointer to a struct")
		return nil
	}

	t := entityType.Elem() // Get the type of the struct
	if t.Kind() != reflect.Struct {
		log.Printf("Error: entity must be a pointer to a struct")
		return nil
	}

	tableName := ""

	// Get the table name from the struct tag
	if field, ok := t.FieldByName("TableName"); ok {
		tableName = field.Tag.Get("karma_table")
		if tableName == "" {
			log.Printf("Warning: TableName field found but karma_table tag is empty")
		}
	} else {
		log.Printf("Warning: No TableName field found with karma_table tag")
	}

	// Build the field mapping
	fieldMap := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// Handle cases where json tag includes options like `json:"name,omitempty"`
			parts := strings.Split(jsonTag, ",")
			fieldMap[field.Name] = parts[0]
		} else {
			fieldMap[field.Name] = field.Name
		}
	}

	// db, err := database.PostgresConn()
	// if err != nil {
	// 	log.Printf("Database connection error: %v", err)
	// 	return nil
	// }

	orm := &ORM{
		tableName:  tableName,
		structType: t,
		fieldMap:   fieldMap,
		db:         nil,
		tx:         nil,
	}

	// Apply options
	for _, opt := range opts {
		opt(orm)
	}
	return orm
}

func WithCacheOn(cacheOn bool) Options {
	return func(o *ORM) {
		opt, err := redis.ParseURL(config.DefaultConfig().RedisURL)
		if err != nil {
			log.Println("Error parsing Redis URL:", err)
			panic(err)
		}
		client := redis.NewClient(opt)
		o.CacheOn = &cacheOn
		o.RedisClient = client
	}
}

func WithCacheMethod(cacheMethod string) Options {
	return func(o *ORM) {
		o.CacheMethod = &cacheMethod
	}
}

func WithCacheKey(cacheKey string) Options {
	return func(o *ORM) {
		o.CacheKey = &cacheKey
	}
}

func WithCacheTTL(cacheTTL time.Duration) Options {
	return func(o *ORM) {
		o.CacheTTL = &cacheTTL
	}
}

func WithRedisClient(redisClient *redis.Client) Options {
	return func(o *ORM) {
		o.RedisClient = redisClient
	}
}

// Scan maps the query result to the provided destination pointer
func (qr *QueryResult) Scan(dest any) error {
	// If there was an error during query execution, return it
	if qr.err != nil {
		log.Println("Query error:", qr.err)
		return qr.err
	}

	// If we have cached data, unmarshal it into the destination
	if qr.cachedData != nil {
		err := json.Unmarshal(qr.cachedData, dest)
		if err != nil {
			log.Printf("Failed to unmarshal cached data: %v", err)
			// Fall back to querying the database if unmarshaling fails
			if qr.rows != nil {
				err := database.ParseRows(qr.rows, dest)
				qr.rows.Close()
				return err
			}
			return err
		}
		return nil
	}

	// If we don't have cached data but do have rows, scan them
	if qr.rows != nil {
		defer qr.rows.Close()

		// Keep a reference to the ORM that created this QueryResult
		// This needs to be added to your QueryResult struct during creation
		orm, ok := qr.orm.(*ORM)

		err := database.ParseRows(qr.rows, dest)
		if err != nil {
			log.Println("Failed to scan rows:", err)
			return err
		}

		// If caching is enabled, cache the results
		if ok && orm != nil && orm.CacheOn != nil && *orm.CacheOn && orm.RedisClient != nil {
			// Serialize the results
			data, err := json.Marshal(dest)
			if err != nil {
				log.Printf("Failed to marshal result for caching: %v", err)
				return nil // Don't return an error as we successfully scanned the results
			}

			// Store in cache
			cacheKey := orm.generateCacheKey(qr.query, qr.args)
			ttl := 5 * time.Minute // Default TTL
			if orm.CacheTTL != nil {
				ttl = *orm.CacheTTL
			}

			ctx := context.Background()
			err = orm.RedisClient.Set(ctx, cacheKey, data, ttl).Err()
			if err != nil {
				log.Printf("Failed to cache query results: %v", err)
			} else {
				log.Printf("Cached query results with key: %s", cacheKey)
			}
		}

		return nil
	}

	return fmt.Errorf("no data available to scan")
}

func (o *ORM) QueryRaw(query string, args ...any) *QueryResult {
	// If caching is disabled, go straight to the database
	if o.CacheOn == nil || !*o.CacheOn || o.RedisClient == nil {
		result := o.executeQuery(query, args...)
		result.orm = o // Set the reference to the ORM
		return result
	}

	// Generate cache key
	cacheKey := o.generateCacheKey(query, args)

	// Try to get from cache first
	ctx := context.Background()
	cachedData, err := o.RedisClient.Get(ctx, cacheKey).Bytes()

	if err == nil {
		// Cache hit - return cached data
		log.Printf("Cache hit for query: %s", query)
		return &QueryResult{
			rows:       nil,
			err:        nil,
			query:      query,
			args:       args,
			cachedData: cachedData,
			orm:        o,
		}
	} else if err != redis.Nil {
		// Log any Redis errors that aren't just "key not found"
		log.Printf("Redis error when getting key %s: %v", cacheKey, err)
	}

	// Cache miss - execute the query
	result := o.executeQuery(query, args...)
	result.orm = o // Set the reference to the ORM

	return result
}

// executeQuery executes the actual database query
func (o *ORM) executeQuery(query string, args ...any) *QueryResult {
	var rows *sql.Rows
	var err error

	// Use transaction if available, otherwise use the database connection
	if o.tx != nil {
		rows, err = o.tx.Query(query, args...)
	} else {
		if o.db == nil {
			db, err := database.PostgresConn()
			if err != nil {
				log.Printf("Database connection error: %v", err)
				return &QueryResult{nil, err, query, args, nil, o}
			}
			o.db = db
		}
		rows, err = o.db.Query(query, args...)
	}

	if err != nil {
		return &QueryResult{nil, err, query, args, nil, o}
	}

	return &QueryResult{
		rows:       rows,
		err:        nil,
		query:      query,
		args:       args,
		cachedData: nil,
		orm:        o,
	}
}

func (o *ORM) Close() {
	if o.tx != nil {
		err := o.tx.Commit()
		if err != nil {
			log.Println("Failed to commit transaction:", err)
		}
	} else if o.db != nil {
		err := o.db.Close()
		if err != nil {
			log.Println("Failed to close database connection:", err)
		}
	}
}

// normalizeValues converts various input formats into a flat slice of values
func (o *ORM) normalizeValues(values ...any) ([]any, error) {
	var valuesSlice []any

	if len(values) == 0 {
		return nil, fmt.Errorf("no values provided")
	}

	if len(values) == 1 {
		// Check if it's a slice
		val := reflect.ValueOf(values[0])
		if val.Kind() == reflect.Slice {
			// Convert the slice to []any
			valuesSlice = make([]any, val.Len())
			for i := range valuesSlice {
				valuesSlice[i] = val.Index(i).Interface()
			}
		} else {
			// Single value
			valuesSlice = values
		}
	} else {
		// Multiple values passed directly
		valuesSlice = values
	}

	// Check if we ended up with an empty slice
	if len(valuesSlice) == 0 {
		return nil, fmt.Errorf("no values provided after normalization")
	}

	return valuesSlice, nil
}

// resolveColumn gets the DB column name for a struct field name
func (o *ORM) resolveColumn(fieldName string) (string, error) {
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s not found in struct", fieldName)
	}
	return columnName, nil
}

// generatePlaceholders creates SQL placeholders ($1, $2, etc.)
func generatePlaceholders(count int) string {
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}
	return strings.Join(placeholders, ", ")
}

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

// GetQuery returns the SQL query that produced this result
func (qr *QueryResult) GetQuery() string {
	return qr.query
}

// GetArgs returns the arguments used in the query
func (qr *QueryResult) GetArgs() []any {
	return qr.args
}
