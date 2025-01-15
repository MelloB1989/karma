package tests

import (
	"fmt"
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

type User struct {
	TableName    string `karma_table:"users"`
	Id           string `json:"id"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Region       string `json:"region"`
	Address      string `json:"address"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Age          int    `json:"age"`
	ProfileImage string `json:"profile_image"`
	Location     string `json:"location"`
	ReferralCode string `json:"referral_code"`
}

type Booking struct {
	TableName         string    `karma_table:"bookings"`
	BookingId         string    `json:"booking_id"`
	UserId            string    `json:"user_id"`
	ServiceProviderId string    `json:"service_provider_id"`
	ServiceId         string    `json:"service_id"`
	VenueId           string    `json:"venue_id"`
	BookingDate       time.Time `json:"booking_date"`
	Status            string    `json:"status"`
	AdvancePaid       int       `json:"advance_paid"`
	PlanId            string    `json:"plan_id"`
}
type Slot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type DaySlot struct {
	WeekDay string `json:"week_day"`
	Slots   []Slot `json:"slots"`
}
type ServicePolicies struct {
	Cancellation   int  `json:"cancellation"`
	AdvanceBooking int  `json:"advance_booking"`
	AdvancePayment int  `json:"advance_payment"`
	Refund         bool `json:"refund"`
}
type ServiceProvider struct {
	TableName        string          `karma_table:"service_providers"`
	Id               string          `json:"id"`
	Name             string          `json:"name"`
	Email            string          `json:"email"`
	Password         string          `json:"password"`
	Phone            string          `json:"phone"`
	LogoImage        string          `json:"logo_image"`
	BannerImage      string          `json:"banner_image"`
	Address          string          `json:"address"`
	RegionsAvailable []string        `json:"regions_available" db:"regions_available"`
	LegalName        string          `json:"legal_name"`
	LegalDocuments   []string        `json:"legal_documents" db:"legal_documents"`
	Policies         ServicePolicies `json:"policies" db:"policies"`
}
type Venue struct {
	TableName       string    `karma_table:"venues"`
	Id              string    `json:"id"`
	ServiceProvider string    `json:"service_provider"`
	Name            string    `json:"name"`
	Address         string    `json:"address"`
	Description     string    `json:"description"`
	Media           []string  `json:"media" db:"media"`
	Location        string    `json:"location"`
	Slots           []DaySlot `json:"slots" db:"slots"`
	PricePerSlot    int       `json:"price_per_slot"`
	MaxCapacity     int       `json:"max_capacity"`
	Region          string    `json:"region"`
	Essentials      []string  `json:"essentials" db:"essentials"`
	Accessibility   bool      `json:"accessibility"`
	Catering        bool      `json:"catering"`
	Dining          bool      `json:"dining"`
	Type            string    `json:"type"`
}

type JoinedBookingResult struct {
	Users    map[string]interface{} `json:"users"`
	Bookings map[string]interface{} `json:"bookings"`
}

func GetBookingsWithUsers(serviceProviderId string) ([]*JoinedBookingResult, error) {
	bookingOrm := orm.Load(&Booking{})
	results, err := bookingOrm.Join(orm.JoinCondition{
		Target:      &User{},
		OnField:     "user_id",
		TargetField: "id",
	}).Into(&JoinedBookingResult{}).Where("service_provider_id", serviceProviderId).Execute()

	if err != nil {
		log.Printf("Failed to get bookings with users: %v", err)
		return nil, fmt.Errorf("error retrieving joined data: %v", err)
	}

	// Convert results to the correct type
	joinedResults := make([]*JoinedBookingResult, len(results))
	for i, result := range results {
		joinedResults[i] = result.(*JoinedBookingResult)
	}

	return joinedResults, nil
}

type JoinedResult struct {
	ServiceProviders ServiceProvider `json:"service_providers" db:"service_providers"`
	Venues           Venue           `json:"venues" db:"venues"`
}

func GetVenues(serviceProviderId string) ([]*JoinedResult, error) {
	venueOrm := orm.Load(&Venue{})
	results, err := venueOrm.Join(orm.JoinCondition{
		Target:      &ServiceProvider{},
		OnField:     "service_provider",
		TargetField: "id",
	}).Into(&JoinedResult{}).Where("id", serviceProviderId).Where("name", "Nice Restraurant").Execute()

	if err != nil {
		log.Printf("Failed to get bookings with users: %v", err)
		return nil, fmt.Errorf("error retrieving joined data: %v", err)
	}

	// Convert results to the correct type
	joinedResults := make([]*JoinedResult, len(results))
	for i, result := range results {
		joinedResults[i] = result.(*JoinedResult)
	}

	return joinedResults, nil
}

func ORMTest() {
	h, _ := GetVenues("1j86yuzaxn")
	fmt.Println(h)
	for _, v := range h {
		fmt.Println(v)
	}

	// Results will contain all fields from both tables
	// Each table's data will be in its own JSON object
	// userORM := orm.Load(&Users{})
	// user := &Users{
	// 	Id:       "1",
	// 	Email:    "",
	// 	Username: "",
	// 	Name:     "",
	// 	Phone:    "",
	// 	Profile:  "",
	// 	Bio:      "",
	// 	College:  "0",
	// 	Year:     2024,
	// 	Branch:   "0",
	// 	Roll:     "",
	// 	JoinedAt: time.Now(),
	// }
	// err := userORM.Insert(user)
	// if err != nil {
	// 	log.Println("Failed to insert user:", err)
	// }
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
	//
}
