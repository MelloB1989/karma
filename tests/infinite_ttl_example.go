package tests

// import (
// 	"fmt"
// 	"log"
// 	"time"

// 	"github.com/MelloB1989/karma/v2/orm"
// )

// // User represents a user entity with infinite cache TTL example
// type User struct {
// 	TableName string `karma_table:"users"`
// 	ID        int    `json:"id" karma:"primary"`
// 	Name      string `json:"name"`
// 	Email     string `json:"email"`
// 	CreatedAt string `json:"created_at"`
// }

// func main() {
// 	// Example 1: Using WithInfiniteCacheTTL() - most convenient way
// 	userORM1 := orm.Load(&User{},
// 		orm.WithCacheOn(true),
// 		orm.WithCacheMethod("both"), // Use both memory and Redis
// 		orm.WithCacheKey("users"),
// 		orm.WithInfiniteCacheTTL(), // Cache forever
// 	)
// 	defer userORM1.Close()

// 	// Example 2: Using WithCacheTTL(InfiniteTTL) - explicit way
// 	userORM2 := orm.Load(&User{},
// 		orm.WithCacheOn(true),
// 		orm.WithCacheMethod("memory"), // Only memory cache
// 		orm.WithCacheKey("users_memory"),
// 		orm.WithCacheTTL(orm.InfiniteTTL), // Same effect as WithInfiniteCacheTTL()
// 	)
// 	defer userORM2.Close()

// 	// Example 3: Mixed TTL - some queries with infinite, others with normal TTL
// 	userORM3 := orm.Load(&User{},
// 		orm.WithCacheOn(true),
// 		orm.WithCacheMethod("redis"),
// 		orm.WithCacheKey("users_mixed"),
// 		orm.WithCacheTTL(5*time.Minute), // Default TTL
// 	)
// 	defer userORM3.Close()

// 	// Check which ORMs are configured for infinite TTL
// 	fmt.Println("=== TTL Configuration Check ===")
// 	fmt.Printf("userORM1 has infinite TTL: %t\n", userORM1.HasInfiniteTTL())
// 	fmt.Printf("userORM2 has infinite TTL: %t\n", userORM2.HasInfiniteTTL())
// 	fmt.Printf("userORM3 has infinite TTL: %t\n", userORM3.HasInfiniteTTL())

// 	// Query with infinite TTL (userORM1)
// 	fmt.Println("\n=== Infinite TTL Example ===")
// 	var users1 []User
// 	result1 := userORM1.QueryRaw("SELECT * FROM users WHERE active = $1", true)
// 	err := result1.Scan(&users1)
// 	if err != nil {
// 		log.Printf("Query error: %v", err)
// 	} else {
// 		fmt.Printf("Found %d users (cached with infinite TTL)\n", len(users1))
// 	}

// 	// Query with memory-only infinite TTL (userORM2)
// 	var users2 []User
// 	result2 := userORM2.QueryRaw("SELECT * FROM users WHERE created_at > $1", "2024-01-01")
// 	err = result2.Scan(&users2)
// 	if err != nil {
// 		log.Printf("Query error: %v", err)
// 	} else {
// 		fmt.Printf("Found %d recent users (memory cache with infinite TTL)\n", len(users2))
// 	}

// 	// Demonstrate manual cache invalidation
// 	fmt.Println("\n=== Manual Cache Invalidation ===")

// 	// Invalidate specific query cache
// 	err = userORM1.InvalidateCache("SELECT * FROM users WHERE active = $1", true)
// 	if err != nil {
// 		log.Printf("Failed to invalidate specific cache: %v", err)
// 	} else {
// 		fmt.Println("Successfully invalidated specific query cache")
// 	}

// 	// Invalidate all caches with "users" prefix
// 	err = userORM1.InvalidateCacheByPrefix("users")
// 	if err != nil {
// 		log.Printf("Failed to invalidate cache by prefix: %v", err)
// 	} else {
// 		fmt.Println("Successfully invalidated all caches with 'users' prefix")
// 	}

// 	// Clear all caches (memory and Redis)
// 	err = userORM1.ClearCache(true)
// 	if err != nil {
// 		log.Printf("Failed to clear cache: %v", err)
// 	} else {
// 		fmt.Println("Successfully cleared all caches")
// 	}

// 	fmt.Println("\n=== Best Practices for Infinite TTL ===")
// 	fmt.Println("1. Use infinite TTL for relatively static data (user profiles, configurations)")
// 	fmt.Println("2. Always implement manual cache invalidation when data changes")
// 	fmt.Println("3. Monitor memory usage when using infinite TTL with memory cache")
// 	fmt.Println("4. Consider using cache prefixes to group related data for easier invalidation")
// 	fmt.Println("5. Use infinite TTL sparingly - most data should have reasonable expiration times")
// 	fmt.Println("6. Use HasInfiniteTTL() method to check if ORM is configured for infinite caching")

// 	// Example of updating data and invalidating cache
// 	fmt.Println("\n=== Data Update and Cache Invalidation Pattern ===")

// 	// Simulate updating user data
// 	updateQuery := "UPDATE users SET name = $1 WHERE id = $2"
// 	// In a real application, you would execute this update
// 	fmt.Printf("Simulating update: %s\n", updateQuery)

// 	// After updating, invalidate related caches
// 	queries_to_invalidate := []struct {
// 		query string
// 		args  []interface{}
// 	}{
// 		{"SELECT * FROM users WHERE active = $1", []interface{}{true}},
// 		{"SELECT * FROM users WHERE id = $1", []interface{}{123}},
// 		{"SELECT * FROM users WHERE created_at > $1", []interface{}{"2024-01-01"}},
// 	}

// 	for _, q := range queries_to_invalidate {
// 		err = userORM1.InvalidateCache(q.query, q.args...)
// 		if err != nil {
// 			log.Printf("Failed to invalidate cache for query '%s': %v", q.query, err)
// 		} else {
// 			fmt.Printf("Invalidated cache for: %s\n", q.query)
// 		}
// 	}
// }
