package orm

import (
	"fmt"
	"time"

	"github.com/MelloB1989/karma/database"
)

type Referrals struct {
	Model
	ReferralId string `json:"referral_id"`
	ReferredBy string `json:"referred_by"`
	CreatedAt  string `json:"created_at"`
}

type User struct {
	Model
	Id           string    `json:"id" karma:"primary_key;unique"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	Email        string    `json:"email"`
	Address      string    `json:"address" karma:"embedded_json"`
	Phone        string    `json:"phone"`
	Dob          time.Time `json:"dob"`
	ReferralCode string    `json:"referral_code" karma:"foreign:referral_id"`
	Region       string    `json:"region"`
}

func NewReferralCode() *Referrals {
	return &Referrals{
		Model: Model{
			TableName:  "referral_codes",
			PrimaryKey: "referral_code",
		},
	}
}

func NewUser() *User {
	return &User{
		Model: Model{
			TableName:  "users",
			PrimaryKey: "id",
		},
	}
}

func ORMTest() {
	// Connect to your database
	db, err := database.PostgresConn()
	if err != nil {
		fmt.Println("Error connecting to database:", err)
		return
	}

	user := Load(NewUser())

	// Load schema and initialize ORM
	orm := &ORM{
		DB:     db,
		Schema: user,
	}

	// Fetch all users
	var users []User
	if err := orm.GetAll(&users); err != nil {
		fmt.Println("Error fetching users:", err)
	} else {
		fmt.Println("Fetched Users:", users)
	}
}
