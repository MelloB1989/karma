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

	// Get a booking by primary key
	ser, err := serviceORM.GetByPrimaryKey("3abb1ealf_")
	if err != nil {
		log.Println("Failed to get booking by primary key:", err)
	} else {
		log.Printf("Fetched booking: %+v\n", ser)
	}

	// Get all bookings
	allServices, err := serviceORM.GetAll()
	if err != nil {
		log.Println("Failed to fetch all bookings:", err)
	} else {
		log.Printf("All bookings: %+v\n", allServices)
	}
}
