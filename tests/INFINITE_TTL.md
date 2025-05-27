# Infinite TTL Cache Feature

## Overview

The Karma ORM now supports infinite Time-To-Live (TTL) caching for both in-memory and Redis cache stores. When infinite TTL is enabled, cached query results will never expire automatically and must be manually invalidated or cleared.

## Quick Start

### Basic Usage

```go
// Using the convenience function
userORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("both"), // Use both memory and Redis
    orm.WithInfiniteCacheTTL(),  // Cache forever
)

// Using the TTL constant explicitly  
userORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheTTL(orm.InfiniteTTL), // Same effect as above
)
```

### Configuration Options

```go
// Memory cache only with infinite TTL
memoryORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("memory"),
    orm.WithInfiniteCacheTTL(),
)

// Redis cache only with infinite TTL
redisORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("redis"),
    orm.WithInfiniteCacheTTL(),
)

// Both caches with infinite TTL and custom cache key
bothORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("both"),
    orm.WithCacheKey("users"),
    orm.WithInfiniteCacheTTL(),
)
```

## API Reference

### Constants

- `orm.InfiniteTTL`: A special duration constant representing infinite cache TTL

### Functions

- `orm.WithInfiniteCacheTTL()`: Option function to configure infinite TTL
- `orm.WithCacheTTL(orm.InfiniteTTL)`: Alternative way to set infinite TTL
- `orm.IsInfiniteTTL(ttl time.Duration) bool`: Check if a duration represents infinite TTL

### Methods

- `orm.HasInfiniteTTL() bool`: Check if ORM is configured for infinite TTL
- `orm.InvalidateCache(query string, args ...any) error`: Remove specific query from cache
- `orm.InvalidateCacheByPrefix(prefix string) error`: Remove all cache entries with prefix
- `orm.ClearCache(clearRedis bool) error`: Clear all cache entries

## Manual Cache Management

Since infinite TTL items never expire automatically, you must manage cache invalidation manually:

### Invalidate Specific Queries

```go
// Invalidate a specific query result
err := userORM.InvalidateCache("SELECT * FROM users WHERE active = $1", true)
if err != nil {
    log.Printf("Failed to invalidate cache: %v", err)
}
```

### Invalidate by Prefix

```go
// Invalidate all cache entries with "users" prefix
err := userORM.InvalidateCacheByPrefix("users")
if err != nil {
    log.Printf("Failed to invalidate cache by prefix: %v", err)
}
```

### Clear All Caches

```go
// Clear memory cache only
err := userORM.ClearCache(false)

// Clear both memory and Redis caches
err := userORM.ClearCache(true)
```

## Data Update Pattern

When updating data that affects cached results, always invalidate related caches:

```go
// Update user data
_, err := db.Exec("UPDATE users SET name = $1 WHERE id = $2", "New Name", userID)
if err != nil {
    return err
}

// Invalidate related caches
queries := []struct {
    query string
    args  []interface{}
}{
    {"SELECT * FROM users WHERE id = $1", []interface{}{userID}},
    {"SELECT * FROM users WHERE active = $1", []interface{}{true}},
    {"SELECT * FROM users ORDER BY created_at DESC", nil},
}

for _, q := range queries {
    if q.args != nil {
        userORM.InvalidateCache(q.query, q.args...)
    } else {
        userORM.InvalidateCache(q.query)
    }
}
```

## Best Practices

### When to Use Infinite TTL

✅ **Good Use Cases:**
- User profiles and settings
- System configurations
- Reference data (countries, categories)
- Lookup tables
- Authentication tokens with custom expiration logic

❌ **Avoid for:**
- Frequently changing data
- Large datasets that could cause memory issues
- Time-sensitive information
- Analytics or reporting data

### Implementation Guidelines

1. **Always implement cache invalidation**: Set up proper invalidation logic when data changes
2. **Use cache prefixes**: Group related data for easier bulk invalidation
3. **Monitor memory usage**: Infinite TTL can accumulate data over time
4. **Be selective**: Use infinite TTL sparingly for truly static data
5. **Consider cache size limits**: Implement cleanup strategies for large datasets

### Memory Management

```go
// Check if using infinite TTL
if userORM.HasInfiniteTTL() {
    log.Println("Using infinite TTL - ensure proper invalidation")
}

// Periodic cleanup for specific prefixes
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        // Clean up old user session caches
        userORM.InvalidateCacheByPrefix("user_sessions")
    }
}()
```

## Implementation Details

### Memory Cache

- Infinite TTL entries are stored with zero time (`time.Time{}`)
- No automatic expiration checking for these entries
- Thread-safe with read/write mutexes

### Redis Cache  

- Infinite TTL translates to Redis TTL of 0 (no expiration)
- Uses standard Redis SET command with 0 expiration
- Supports all Redis cache features (clustering, persistence, etc.)

### Performance

- Infinite TTL items have slightly better performance (no expiration checks)
- Memory usage grows over time without manual invalidation
- Redis memory usage follows Redis configuration and policies

## Migration Guide

### From Regular TTL to Infinite TTL

```go
// Before
userORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheTTL(24 * time.Hour),
)

// After  
userORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithInfiniteCacheTTL(), // Add invalidation logic!
)
```

### From No Caching to Infinite TTL

```go
// Before
userORM := orm.Load(&User{})

// After
userORM := orm.Load(&User{},
    orm.WithCacheOn(true),
    orm.WithCacheMethod("both"),
    orm.WithCacheKey("users"),
    orm.WithInfiniteCacheTTL(),
)
```

## Troubleshooting

### Common Issues

**Cache not invalidating:**
- Check that you're using the exact same query and arguments
- Verify cache keys are generated consistently
- Ensure Redis connection is working

**Memory usage growing:**
- Implement regular cache cleanup
- Use more specific cache prefixes
- Consider switching to Redis-only caching

**Cache misses:**
- Verify infinite TTL is properly configured with `HasInfiniteTTL()`
- Check cache method configuration
- Monitor Redis connectivity

### Debugging

```go
// Check TTL configuration
if !userORM.HasInfiniteTTL() {
    log.Println("Warning: Expected infinite TTL but not configured")
}

// Verify cache method
fmt.Printf("Cache method: %s\n", *userORM.CacheMethod)
fmt.Printf("Cache key prefix: %s\n", *userORM.CacheKey)
```

## Examples

See the complete example in `examples/infinite_ttl_example.go` for a comprehensive demonstration of infinite TTL usage patterns.