package tests

import (
	"log"
	"time"

	"github.com/MelloB1989/karma/orm"
)

type Service struct {
	TableName string    `karma_table:"services"`
	ServiceId string    `json:"service_id" karma:"primary;unique"`
	Type      string    `json:"type"` //"local", "online", "offline"
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Banner    string    `json:"banner"`
	Category  string    `json:"category"`   // "food", "clothing", "electronics", "services", "entertainment", "education", "health", "beauty", "travel", "venues
	OfferedBy string    `json:"offered_by"` // "global", "service_provider_id"
	Timestamp time.Time `json:"timestamp"`
}

type Users struct {
	TableName string    `karma_table:"users"`
	Id        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Profile   string    `json:"profile"`
	Bio       string    `json:"bio"`
	College   string    `json:"college"`
	Year      int       `json:"year"`
	Branch    string    `json:"branch"`
	Roll      string    `json:"roll"`
	JoinedAt  time.Time `json:"joined_at"`
}

func ORMTest() {
	userORM := orm.Load(&Users{})
	user := &Users{
		Id:       "1",
		Email:    "",
		Username: "",
		Name:     "",
		Phone:    "",
		Profile:  "",
		Bio:      "",
		College:  "0",
		Year:     2024,
		Branch:   "0",
		Roll:     "",
		JoinedAt: time.Now(),
	}
	err := userORM.Insert(user)
	if err != nil {
		log.Println("Failed to insert user:", err)
	}
	// serviceORM := Load(&Service{})
	// // r, e := serviceORM.GetByFieldCompare("Type", "local", "=")
	// // s, err := AssertAndReturnSlice(reflect.TypeOf(&Service{}), r, e)
	// // fmt.Println(s[0])
	// servicesByType, err := serviceORM.GetByFieldCompare("Type", "local", "=")
	// if err != nil {
	// 	log.Println("Failed to fetch services by type:", err)
	// } else {
	// 	// Assert the result to []*Service
	// 	services, ok := servicesByType.([]*Service)
	// 	if !ok {
	// 		log.Println("Failed to assert servicesByType to []*Service")
	// 		return
	// 	}
	// 	log.Printf("Services of type 'local': %+v\n", services[0].Banner)
	// }

	// // Get services with specific categories
	// categoryList := []any{"venues", "clothing", "electronics"}
	// servicesByCategory, err := serviceORM.GetByFieldIn("Category", categoryList)
	// if err != nil {
	// 	log.Println("Failed to fetch services by category:", err)
	// } else {
	// 	// Assert the result to []*Service
	// 	services, ok := servicesByCategory.([]*Service)
	// 	if !ok {
	// 		log.Println("Failed to assert servicesByCategory to []*Service")
	// 		return
	// 	}
	// 	log.Printf("Services in categories: %+v\n", services)
	// }

	// // Get the count of services with a specific type
	// count, err := serviceORM.GetCount("Type", "local", "=")
	// if err != nil {
	// 	log.Println("Failed to get count of services by type:", err)
	// } else {
	// 	log.Printf("Count of 'local' services: %d\n", count)
	// }
}
