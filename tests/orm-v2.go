package tests

import (
	"fmt"

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

func TestORMV2() {
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
