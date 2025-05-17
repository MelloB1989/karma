package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/MelloB1989/karma/v2/orm"
)

type Item struct {
	TableName               struct{} `karma_table:"items"`
	Id                      string   `json:"id"`
	CuisineId               string   `json:"cuisine_id"`
	Category                string   `json:"category"`
	SpiceLevel              string   `json:"spice_level"`
	Ingredients             []string `json:"ingredients" db:"ingredients"`
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	Price                   int      `json:"price"`
	Images                  []string `json:"images" db:"images"`
	Type                    string   `json:"type"`
	Timing                  string   `json:"timing"`
	Tags                    []string `json:"tags" db:"tags"`
	Region                  string   `json:"region"`
	StandardServingQuantity int      `json:"standard_serving_quantity"`
	StandardServingUnit     string   `json:"standard_serving_unit"`
	StandardServingPrice    int      `json:"standard_serving_price"`
	MinHeadCount            int      `json:"min_head_count"`
}

type Packages struct {
	TableName          struct{} `karma_table:"packages"`
	Id                 string   `json:"id"`
	Banner             string   `json:"banner"`
	MinGuests          int      `json:"min_guests"`
	MaxGuests          int      `json:"max_guests"`
	ServiceId          string   `json:"service_id"`
	Name               string   `json:"name"`
	DiscountPercentage int      `json:"discount_percentage"`
	DiscountPerXPerson int      `json:"discount_per_x_person"`
	BasePrice          int      `json:"base_price"`
	Admin              bool     `json:"admin"`
}

type PackageGroups struct {
	TableName        struct{} `karma_table:"package_groups"`
	Id               string   `json:"id"`
	PackageId        string   `json:"package_id"`
	Name             string   `json:"name"`
	Banner           string   `json:"banner"`
	DefaultSelection []string `json:"default_selection"`
	MaxItems         int      `json:"max_items"`
}

type PackageItems struct {
	TableName    struct{} `karma_table:"package_items"`
	Id           string   `json:"id"`
	PackageId    string   `json:"package_id"`
	GroupId      string   `json:"group_id"`
	ItemId       string   `json:"item_id"`
	Quantity     int      `json:"quantity"`
	Price        int      `json:"price"`
	MinHeadCount int      `json:"min_head_count"`
}

type PackageFull struct {
	Package       Packages `json:"package"`
	PackageGroups []struct {
		Group PackageGroups `json:"group"`
		Items []Item        `json:"items"`
	} `json:"package_groups"`
}

func TestORMV2(t *testing.T) {
	// userORM := orm.Load(&User{})
	// var user []User
	// // userORM.QueryRaw("SELECT * FROM users WHERE phone = $1", "+917569236628").Scan(&user)
	// userORM.GetByFieldIn("Phone", "+917569236628").Scan(&user)
	// fmt.Println(user[0].Email)

	// itemsORM := orm.Load(&Item{})
	// var items []Item
	// itemsORM.OrderBy("Price", orm.OrderAsc).Scan(&items)
	// fmt.Println(items[0].Price)
	//
	packagesORM := orm.Load(&Packages{})
	// var result []struct {
	// 	Packages
	// 	PackageGroups
	// }
	res := packagesORM.JoinOnFields(orm.LeftJoin, "packages_groups", "id", "package_id").Execute() //.Scan(&result)
	fmt.Println(res)
}

func TestORMCachingPerformance(t *testing.T) {
	// Define test data structure
	type Users struct {
		TableName    struct{}          `karma_table:"users"`
		Id           string            `json:"id" karma:"primary"`
		Username     string            `json:"username"`
		Email        string            `json:"email"`
		Name         string            `json:"name"`
		Phone        string            `json:"phone"`
		Bio          string            `json:"bio"`
		ProfileImage string            `json:"profile_image"`
		Socials      map[string]string `json:"socials" db:"socials"`
		DateOfBirth  time.Time         `json:"date_of_birth"`
		Gender       string            `json:"gender"`
		CreatedAt    time.Time         `json:"created_at"`
		DeviceId     string            `json:"device_id"`
		PasswordHash string            `json:"password_hash"`
	}

	// Test user ID to query
	testUserId := "user_2uiCd5By9dH12HMlXSzXzLyusAh"

	// Number of iterations for performance testing
	iterations := 10

	// First test: ORM with caching disabled
	t.Run("Without Cache", func(t *testing.T) {
		// Create ORM with caching disabled
		uncachedORM := orm.Load(&Users{}, orm.WithCacheOn(false))
		defer uncachedORM.Close()

		// Clear any existing cache before starting
		uncachedORM.ClearCache(true)

		// Measure performance
		startTime := time.Now()

		for i := 0; i < iterations; i++ {
			var users []Users
			err := uncachedORM.GetByFieldEquals("Id", testUserId).Scan(&users)
			if err != nil {
				t.Fatalf("Error selecting users: %v", err)
			}

			// Verify we got results
			if len(users) == 0 {
				t.Fatalf("No users found with ID: %s", testUserId)
			}

			// Optional: Add a small delay to simulate real-world usage
			time.Sleep(10 * time.Millisecond)
		}

		elapsedTime := time.Since(startTime)
		avgTime := elapsedTime / time.Duration(iterations)

		t.Logf("Without cache - Total time for %d queries: %v (avg: %v per query)",
			iterations, elapsedTime, avgTime)
	})

	// Second test: ORM with caching enabled
	t.Run("With Cache", func(t *testing.T) {
		// Create ORM with caching enabled
		cachedORM := orm.Load(&Users{},
			orm.WithCacheOn(true),
			orm.WithCacheTTL(5*time.Minute),
			orm.WithCacheKey("storizz:users"),
			orm.WithCacheMethod("memory"))
		defer cachedORM.Close()

		// Clear any existing cache before starting
		cachedORM.ClearCache(true)

		// Measure performance
		startTime := time.Now()

		for i := 0; i < iterations; i++ {
			var users []Users
			err := cachedORM.GetByFieldEquals("Id", testUserId).Scan(&users)
			if err != nil {
				t.Fatalf("Error selecting users: %v", err)
			}

			// Verify we got results
			if len(users) == 0 {
				t.Fatalf("No users found with ID: %s", testUserId)
			}

			// Optional: Add a small delay to simulate real-world usage
			time.Sleep(10 * time.Millisecond)
		}

		elapsedTime := time.Since(startTime)
		avgTime := elapsedTime / time.Duration(iterations)

		t.Logf("With cache - Total time for %d queries: %v (avg: %v per query)",
			iterations, elapsedTime, avgTime)
	})

	// Third test: Detailed cache performance breakdown
	t.Run("Cache Performance Breakdown", func(t *testing.T) {
		// Create ORM with caching enabled
		cachedORM := orm.Load(&Users{},
			orm.WithCacheOn(true),
			orm.WithCacheTTL(5*time.Minute),
			orm.WithCacheKey("storizz:users"),
			orm.WithCacheMethod("memory"))
		defer cachedORM.Close()

		// Clear any existing cache before starting
		cachedORM.ClearCache(true)

		// First query (cache miss)
		var firstQueryElapsed time.Duration
		{
			startTime := time.Now()

			var users []Users
			err := cachedORM.GetByFieldEquals("Id", testUserId).Scan(&users)
			if err != nil {
				t.Fatalf("Error selecting users: %v", err)
			}

			firstQueryElapsed = time.Since(startTime)
			t.Logf("First query (cache miss): %v", firstQueryElapsed)
		}

		// Subsequent queries (cache hits)
		var cacheMissTotal, cacheHitTotal time.Duration

		// Reset for accurate measurement
		cacheMissTotal = firstQueryElapsed

		for i := 0; i < iterations; i++ {
			startTime := time.Now()

			var users []Users
			err := cachedORM.GetByFieldEquals("Id", testUserId).Scan(&users)
			if err != nil {
				t.Fatalf("Error selecting users: %v", err)
			}

			queryTime := time.Since(startTime)
			cacheHitTotal += queryTime
		}

		// Calculate average times
		avgCacheMiss := cacheMissTotal
		avgCacheHit := cacheHitTotal / time.Duration(iterations)

		// Calculate performance improvement
		improvement := float64(avgCacheMiss) / float64(avgCacheHit)

		t.Logf("Cache miss avg: %v", avgCacheMiss)
		t.Logf("Cache hit avg: %v", avgCacheHit)
		t.Logf("Performance improvement: %.2fx faster with cache", improvement)
	})
}

// Example manual test with output printing
func TestORMCaching(t *testing.T) {
	type Users struct {
		TableName    struct{}          `karma_table:"users"`
		Id           string            `json:"id" karma:"primary"`
		Username     string            `json:"username"`
		Email        string            `json:"email"`
		Name         string            `json:"name"`
		Phone        string            `json:"phone"`
		Bio          string            `json:"bio"`
		ProfileImage string            `json:"profile_image"`
		Socials      map[string]string `json:"socials" db:"socials"`
		DateOfBirth  time.Time         `json:"date_of_birth"`
		Gender       string            `json:"gender"`
		CreatedAt    time.Time         `json:"created_at"`
		DeviceId     string            `json:"device_id"`
		PasswordHash string            `json:"password_hash"`
	}

	// Test with caching enabled
	fmt.Println("===== Testing with cache enabled =====")
	usersORM := orm.Load(&Users{},
		orm.WithCacheOn(true),
		orm.WithCacheTTL(5*time.Minute),
		orm.WithCacheKey("storizz:users"),
		orm.WithCacheMethod("memory"))
	defer usersORM.Close()

	fmt.Println("First query (cache miss):")
	start := time.Now()
	var users []Users
	err := usersORM.GetByFieldEquals("Id", "user_2uiCd5By9dH12HMlXSzXzLyusAh").Scan(&users)
	if err != nil {
		fmt.Println("Error selecting users:", err)
		return
	}
	fmt.Printf("Query time: %v\n", time.Since(start))
	fmt.Printf("Found %d users\n", len(users))
	fmt.Println("User details:", users[0])
}

/*
// Initialize ORM with in-memory caching only
orm := Load(&MyEntity{},
    WithCacheOn(true),
    WithCacheMethod("memory"),
    WithCacheTTL(10 * time.Minute))

// Or use both Redis and in-memory caching (default)
orm := Load(&MyEntity{},
    WithCacheOn(true),
    WithCacheMethod("both"))

// Clear cached data when needed
orm.InvalidateCache("SELECT * FROM users WHERE id = $1", userId)
orm.ClearCache(true) // Clear both memory and Redis caches
*/

/*
 // Start a transaction
 tx, err := orm.Begin()
 if err != nil {
     log.Fatalf("Failed to begin transaction: %v", err)
 }

 // Get the ORM associated with the transaction
 txOrm := tx.ORM()

 // Use the transaction ORM to perform database operations
 _, err = txOrm.Insert(map[string]any{
     "name": "John Doe",
     "email": "john@example.com",
 })
 if err != nil {
     // Roll back the transaction if an error occurs
     tx.Rollback()
     log.Fatalf("Insert failed: %v", err)
 }

 // Perform another operation
 err = txOrm.Update(123, map[string]any{
     "status": "active",
 })
 if err != nil {
     // Roll back the transaction if an error occurs
     tx.Rollback()
     log.Fatalf("Update failed: %v", err)
 }

 // Commit the transaction if all operations succeed
 err = tx.Commit()
 if err != nil {
     log.Fatalf("Failed to commit transaction: %v", err)
 }

 ### Example 2: Using the WithTransaction helper

 err := orm.WithTransaction(func(txOrm *ORM) error {
     // Insert a new user
     userID, err := txOrm.Insert(map[string]any{
         "name": "Jane Doe",
         "email": "jane@example.com",
     })
     if err != nil {
         return err // Transaction will be rolled back
     }

     // Insert an address for the user
     _, err = txOrm.Table("addresses").Insert(map[string]any{
         "user_id": userID,
         "street": "123 Main St",
         "city": "Anytown",
         "state": "CA",
     })
     if err != nil {
         return err // Transaction will be rolled back
     }

     // All operations successful, transaction will be committed
     return nil
 })

 if err != nil {
     log.Fatalf("Transaction failed: %v", err)
 }

 ### Example 1: Simple inner join
 // Get users with their orders
 result := orm.InnerJoin("orders", "users.id = orders.user_id").Execute()

 ### Example 2: Multiple joins with conditions
 // Get users with their orders and order items
 result := orm.InnerJoin("orders", "users.id = orders.user_id")
             .AddJoin(InnerJoin, "order_items", "orders.id = order_items.order_id")
             .Where("users.status = ?", "active")
             .OrderBy("users.created_at DESC")
             .Limit(10)
             .Execute()

 ### Example 3: Using the simplified join helpers
 // Join tables that have matching field names (e.g., both tables have an 'id' field)
 result := orm.SimpleJoin(LeftJoin, "profiles", "id").Execute()

 // Join tables with different field names
 result := orm.JoinOnFields(LeftJoin, "addresses", "id", "user_id").Execute()

 ### Example 4: Specifying columns to select
 // Select specific columns from joined tables
 result := orm.InnerJoin("orders", "users.id = orders.user_id")
             .Select("users.name", "users.email", "orders.order_date", "orders.total")
             .Execute()
*/

/*
For new:
package main

import (
	"fmt"
	"log"

	"github.com/MelloB1989/karma/orm"
)

// User represents a user in the system
type User struct {
	TableName string `karma_table:"users"`
	ID        int    `json:"id" karma_pk:"true"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	Active    bool   `json:"active"`
}

func main() {
	// Create a new User instance
	user := &User{}

	// Load the ORM
	ormInstance := orm.Load(user)
	defer ormInstance.Close()

	// Example 1: Basic Select
	fmt.Println("Example 1: Basic Select")
	var users []User
	err := ormInstance.Select().Execute().ScanAll(&users)
	if err != nil {
		log.Fatalf("Error selecting users: %v", err)
	}

	// Example 2: Select with conditions
	fmt.Println("Example 2: Select with conditions")
	var activeUsers []User
	err = ormInstance.Select().
		Where("active", orm.Equals, true).
		Where("age", orm.GreaterThan, 18).
		OrderBy("username", orm.OrderAsc).
		Execute().ScanAll(&activeUsers)
	if err != nil {
		log.Fatalf("Error selecting active users: %v", err)
	}

	// Example 3: Complex query with multiple conditions
	fmt.Println("Example 3: Complex query with multiple conditions")
	var filteredUsers []User
	err = ormInstance.Select("id", "username", "email").
		Where("age", orm.GreaterThanOrEquals, 21).
		WhereIn("username", "john", "jane", "bob").
		OrderBy("age", orm.OrderDesc).
		Limit(10).
		Execute().ScanAll(&filteredUsers)
	if err != nil {
		log.Fatalf("Error selecting filtered users: %v", err)
	}

	// Example 4: Find by primary key
	fmt.Println("Example 4: Find by primary key")
	foundUser := &User{}
	err = ormInstance.FindByPK(1).Execute().Scan(foundUser)
	if err != nil {
		log.Fatalf("Error finding user by ID: %v", err)
	}

	// Example 5: Count
	fmt.Println("Example 5: Count")
	count, err := ormInstance.Count().
		Where("active", orm.Equals, true).
		Execute().Value()
	if err != nil {
		log.Fatalf("Error counting active users: %v", err)
	}
	fmt.Printf("Active users count: %v\n", count)

	// Example 7: Delete
	fmt.Println("Example 7: Delete")
	result := ormInstance.Delete().
		Where("active", orm.Equals, false).
		Execute()
	if result.err != nil {
		log.Fatalf("Error deleting inactive users: %v", result.err)
	}

	// Example 8: Between
	fmt.Println("Example 8: Between")
	var middleAgedUsers []User
	err = ormInstance.Select().
		WhereBetween("age", 30, 50).
		Execute().ScanAll(&middleAgedUsers)
	if err != nil {
		log.Fatalf("Error selecting middle-aged users: %v", err)
	}

	// Example 9: Null checks
	fmt.Println("Example 9: Null checks")
	var usersWithoutEmail []User
	err = ormInstance.Select().
		WhereNull("email").
		Execute().ScanAll(&usersWithoutEmail)
	if err != nil {
		log.Fatalf("Error selecting users without email: %v", err)
	}

	// Example 10: Raw query
	fmt.Println("Example 10: Raw query")
	var rawUsers []User
	err = ormInstance.Raw("SELECT * FROM users WHERE age > $1 AND active = $2", 18, true).
		Execute().ScanAll(&rawUsers)
	if err != nil {
		log.Fatalf("Error executing raw query: %v", err)
	}
}
*/
