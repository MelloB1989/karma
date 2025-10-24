package orm

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MelloB1989/karma/database"
	"github.com/MelloB1989/karma/utils"
	"github.com/jmoiron/sqlx"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
)

var (
	ctx         = context.Background()
	json        = jsoniter.ConfigFastest
	memoryCache = newMemoryCache()
)

const (
	InfiniteTTL = time.Duration(-1)
)

// MemoryCache provides in-memory caching capabilities
type MemoryCache struct {
	data  map[string][]byte
	ttl   map[string]time.Time
	mutex sync.RWMutex
}

// newMemoryCache initializes a new in-memory cache
func newMemoryCache() *MemoryCache {
	return &MemoryCache{
		data:  make(map[string][]byte),
		ttl:   make(map[string]time.Time),
		mutex: sync.RWMutex{},
	}
}

// Get retrieves an item from the memory cache
func (m *MemoryCache) Get(key string) ([]byte, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	expireTime, exists := m.ttl[key]
	if !exists {
		return nil, false
	}

	// Check if this is an infinite TTL entry (zero time means infinite)
	if !expireTime.IsZero() && time.Now().After(expireTime) {
		// Use a goroutine to clean up expired key to avoid blocking
		go func(key string) {
			m.mutex.Lock()
			delete(m.data, key)
			delete(m.ttl, key)
			m.mutex.Unlock()
		}(key)
		return nil, false
	}

	data, exists := m.data[key]
	return data, exists
}

// Set stores an item in the memory cache with the specified TTL.
// If ttl is InfiniteTTL or negative, the item will never expire automatically.
func (m *MemoryCache) Set(key string, data []byte, ttl time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data[key] = data

	// Handle infinite TTL by storing zero time
	if IsInfiniteTTL(ttl) {
		m.ttl[key] = time.Time{} // Zero time indicates infinite TTL
	} else {
		m.ttl[key] = time.Now().Add(ttl)
	}
}

// Delete removes an item from the memory cache
func (m *MemoryCache) Delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	delete(m.ttl, key)
}

// IsInfiniteTTL checks if the given duration represents an infinite TTL
func IsInfiniteTTL(ttl time.Duration) bool {
	return ttl == InfiniteTTL || ttl < 0
}

// HasInfiniteTTL returns true if the ORM is configured to use infinite cache TTL
func (o *ORM) HasInfiniteTTL() bool {
	return o.CacheTTL != nil && IsInfiniteTTL(*o.CacheTTL)
}

// ORM struct encapsulates the metadata and methods for a table.
type ORM struct {
	tableName      string
	structType     reflect.Type
	fieldMap       map[string]string
	tx             *sqlx.Tx
	db             *sqlx.DB
	CacheOn        *bool
	CacheMethod    *string // "redis", "memory", or "both"
	CacheKey       *string
	CacheTTL       *time.Duration
	RedisClient    *redis.Client
	serializeMux   sync.Mutex
	databasePrefix string
}

type QueryResult struct {
	rows       *sql.Rows
	err        error
	query      string
	args       []any
	cachedData []byte
	orm        any
}

// encodeArgs serializes query arguments to create a unique cache key using the faster jsoniter
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

// generateCacheKey creates a unique key for caching based on the query and arguments
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

	orm := &ORM{
		tableName:  fmt.Sprintf(`"%s"`, tableName), // Quote table name for case sensitivity
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

func WithDatabasePrefix(prefix string) Options {
	return func(o *ORM) {
		o.databasePrefix = prefix
	}
}

func WithCacheOn(cacheOn bool) Options {
	return func(o *ORM) {
		client := utils.RedisConnect()
		o.CacheOn = &cacheOn
		o.RedisClient = client

		// Default to both memory and Redis caching
		cacheMethod := "both"
		o.CacheMethod = &cacheMethod
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

// WithCacheTTL sets the cache time-to-live duration for cached query results.
// Use InfiniteTTL or any negative duration for items that never expire.
func WithCacheTTL(cacheTTL time.Duration) Options {
	return func(o *ORM) {
		o.CacheTTL = &cacheTTL
	}
}

// WithInfiniteCacheTTL configures the ORM to cache query results indefinitely.
// Cached items will never expire automatically and must be manually invalidated
// using InvalidateCache, InvalidateCacheByPrefix, or ClearCache methods.
func WithInfiniteCacheTTL() Options {
	return func(o *ORM) {
		infiniteTTL := InfiniteTTL
		o.CacheTTL = &infiniteTTL
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
		orm, ok := qr.orm.(*ORM)

		err := database.ParseRows(qr.rows, dest)
		if err != nil {
			log.Println("Failed to scan rows:", err)
			return err
		}

		// If caching is enabled, cache the results
		if ok && orm != nil && orm.CacheOn != nil && *orm.CacheOn {
			// Use a goroutine to handle caching in the background to not block
			go func(orm *ORM, dest any, query string, args []any) {
				// Prevent concurrent access to serialization
				orm.serializeMux.Lock()
				defer orm.serializeMux.Unlock()

				// Serialize the results with the faster jsoniter
				data, err := json.Marshal(dest)
				if err != nil {
					log.Printf("Failed to marshal result for caching: %v", err)
					return
				}

				cacheKey := orm.generateCacheKey(query, args)
				ttl := 5 * time.Minute // Default TTL
				if orm.CacheTTL != nil {
					ttl = *orm.CacheTTL
				}

				cacheMethod := "both"
				if orm.CacheMethod != nil {
					cacheMethod = *orm.CacheMethod
				}

				// Cache in memory if requested
				if cacheMethod == "memory" || cacheMethod == "both" {
					memoryCache.Set(cacheKey, data, ttl)
					if IsInfiniteTTL(ttl) {
						log.Printf("Cached query results in memory with infinite TTL and key: %s", cacheKey)
					} else {
						log.Printf("Cached query results in memory with key: %s (TTL: %s)", cacheKey, ttl)
					}
				}

				// Cache in Redis if requested
				if (cacheMethod == "redis" || cacheMethod == "both") && orm.RedisClient != nil {
					ctx := context.Background()
					var redisTTL time.Duration

					// Handle infinite TTL for Redis (0 means no expiration)
					if IsInfiniteTTL(ttl) {
						redisTTL = 0
					} else {
						redisTTL = ttl
					}

					err = orm.RedisClient.Set(ctx, cacheKey, data, redisTTL).Err()
					if err != nil {
						log.Printf("Failed to cache query results in Redis: %v", err)
					} else {
						if redisTTL == 0 {
							log.Printf("Cached query results in Redis with infinite TTL and key: %s", cacheKey)
						} else {
							log.Printf("Cached query results in Redis with key: %s (TTL: %s)", cacheKey, redisTTL)
						}
					}
				}
			}(orm, dest, qr.query, qr.args)
		}

		return nil
	}

	return fmt.Errorf("no data available to scan")
}

func (o *ORM) QueryRaw(query string, args ...any) *QueryResult {
	// If caching is disabled, go straight to the database
	if o.CacheOn == nil || !*o.CacheOn {
		result := o.executeQuery(query, args...)
		result.orm = o // Set the reference to the ORM
		return result
	}

	// Generate cache key
	cacheKey := o.generateCacheKey(query, args)
	cacheMethod := "both"
	if o.CacheMethod != nil {
		cacheMethod = *o.CacheMethod
	}

	// Try to get from memory cache first (fastest)
	if cacheMethod == "memory" || cacheMethod == "both" {
		if cachedData, found := memoryCache.Get(cacheKey); found {
			log.Printf("Memory cache hit for query: %s", query)
			return &QueryResult{
				rows:       nil,
				err:        nil,
				query:      query,
				args:       args,
				cachedData: cachedData,
				orm:        o,
			}
		}
	}

	// Try to get from Redis if using Redis caching
	if (cacheMethod == "redis" || cacheMethod == "both") && o.RedisClient != nil {
		// Create a context with timeout for Redis operations
		redisCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		cachedData, err := o.RedisClient.Get(redisCtx, cacheKey).Bytes()

		if err == nil {
			// Cache hit - return cached data
			log.Printf("Redis cache hit for query: %s", query)

			// Store in memory cache as well for faster subsequent access if using both
			if cacheMethod == "both" {
				ttl := 5 * time.Minute
				if o.CacheTTL != nil {
					ttl = *o.CacheTTL
				}
				memoryCache.Set(cacheKey, cachedData, ttl)
			}

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
			var db *sqlx.DB
			var err error
			if o.databasePrefix != "" {
				db, err = database.PostgresConn(database.PostgresConnOptions{
					DatabaseUrlPrefix: o.databasePrefix,
				})
			} else {
				db, err = database.PostgresConn()
			}
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
	} else if o.RedisClient != nil {
		err := o.RedisClient.Close()
		if err != nil {
			log.Println("Failed to close Redis connection:", err)
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
// Returns the column name wrapped in double quotes to preserve case sensitivity in PostgreSQL
func (o *ORM) resolveColumn(fieldName string) (string, error) {
	columnName, ok := o.fieldMap[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s not found in struct", fieldName)
	}
	return fmt.Sprintf(`"%s"`, columnName), nil
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

// Invalidate removes an item from both memory and Redis caches
func (o *ORM) InvalidateCache(query string, args ...any) error {
	cacheKey := o.generateCacheKey(query, args)

	// Remove from memory cache
	memoryCache.Delete(cacheKey)

	// Remove from Redis if using Redis
	if o.RedisClient != nil {
		err := o.RedisClient.Del(ctx, cacheKey).Err()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to invalidate Redis cache: %v", err)
		}
	}

	return nil
}

func (o *ORM) InvalidateCacheByPrefix(prefix string) error {
	// Clear memory cache keys with the given prefix
	prefixPattern := prefix + ":"
	memoryCache.mutex.Lock()
	for key := range memoryCache.data {
		if strings.HasPrefix(key, prefixPattern) {
			delete(memoryCache.data, key)
			delete(memoryCache.ttl, key)
		}
	}
	memoryCache.mutex.Unlock()

	if o.RedisClient == nil {
		o.RedisClient = utils.RedisConnect()
	}

	// Create a pattern to match keys with the given prefix
	pattern := prefix + ":*"
	var cursor uint64
	var keys []string
	var err error

	for {
		keys, cursor, err = o.RedisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan Redis keys: %v", err)
		}

		if len(keys) > 0 {
			err = o.RedisClient.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("failed to delete Redis keys: %v", err)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// ClearCache removes all items from the memory cache and optionally Redis
func (o *ORM) ClearCache(clearRedis bool) error {
	// Reset memory cache with a new one
	memoryCache = newMemoryCache()

	// Clear Redis cache if requested and available
	if clearRedis && o.RedisClient != nil {
		// This is a potentially dangerous operation as it clears ALL Redis keys
		// Consider using a prefix pattern for more targeted clearing
		if o.CacheKey != nil {
			pattern := *o.CacheKey + ":*"
			var cursor uint64
			var keys []string
			var err error

			for {
				keys, cursor, err = o.RedisClient.Scan(ctx, cursor, pattern, 100).Result()
				if err != nil {
					return fmt.Errorf("failed to scan Redis keys: %v", err)
				}

				if len(keys) > 0 {
					err = o.RedisClient.Del(ctx, keys...).Err()
					if err != nil {
						return fmt.Errorf("failed to delete Redis keys: %v", err)
					}
				}

				if cursor == 0 {
					break
				}
			}
		}
	}

	return nil
}
