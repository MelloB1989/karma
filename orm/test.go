package orm

import (
	"log"
	"time"
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

func ORMTest() {
	serviceORM := Load(&Service{})

	// Get a service by primary key
	ser, err := serviceORM.GetByPrimaryKey("3abb1ealf_")
	if err != nil {
		log.Println("Failed to get service by primary key:", err)
	} else {
		log.Printf("Fetched service: %+v\n", ser)
	}

	// Get all services
	allServices, err := serviceORM.GetAll()
	if err != nil {
		log.Println("Failed to fetch all services:", err)
	} else {
		log.Printf("All services: %+v\n", allServices)
	}

	// Get services by type ("local")
	servicesByType, err := serviceORM.GetByFieldCompare("type", "local", "=")
	if err != nil {
		log.Println("Failed to fetch services by type:", err)
	} else {
		// Assert the result to []*Service
		services, ok := servicesByType.([]*Service)
		if !ok {
			log.Println("Failed to assert servicesByType to []*Service")
			return
		}
		log.Printf("Services of type 'local': %+v\n", services)
	}

	// Get services with specific categories
	categoryList := []any{"food", "clothing", "electronics"}
	servicesByCategory, err := serviceORM.GetByFieldIn("Category", categoryList)
	if err != nil {
		log.Println("Failed to fetch services by category:", err)
	} else {
		// Assert the result to []*Service
		services, ok := servicesByCategory.([]*Service)
		if !ok {
			log.Println("Failed to assert servicesByCategory to []*Service")
			return
		}
		log.Printf("Services in categories: %+v\n", services)
	}

	// Get the count of services with a specific type
	count, err := serviceORM.GetCount("Type", "local", "=")
	if err != nil {
		log.Println("Failed to get count of services by type:", err)
	} else {
		log.Printf("Count of 'local' services: %d\n", count)
	}
}
