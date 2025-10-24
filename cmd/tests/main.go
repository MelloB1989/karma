package main

import (
	"log"
	"time"

	"github.com/MelloB1989/karma/utils"
	"github.com/MelloB1989/karma/v2/orm"
)

func main() {
	// bedrock.StartChatSession()
	// err := godotenv.Load()
	// if err != nil {
	// 	panic(err)
	// }
	// embeddings, error := tests.GetEmbedding("Hello this is test embeddings")
	// if error != nil {
	// 	panic(error)
	// }
	// fmt.Print(embeddings)
	// tests.TestKai()
	// tests.TestImageGen()
	// tests.TestMCPServer()
	// tests.TestSendingSingleMail()
	// tests.ORMTest()
	type User struct {
		TableName              struct{}       `karma_table:"user"`
		Id                     string         `json:"id" karma:"primary"`
		Email                  string         `json:"email"`
		FirstName              string         `json:"firstName"`
		LastName               string         `json:"lastName"`
		Password               string         `json:"password"`
		PersonalizationAnswers map[string]any `json:"personalizationAnswers" db:"personalizationAnswers"`
		CreatedAt              time.Time      `json:"createdAt"`
		UpdatedAt              time.Time      `json:"updatedAt"`
		Settings               map[string]any `json:"settings" db:"settings"`
		Disabled               bool           `json:"disabled"`
		MfaEnabled             bool           `json:"mfaEnabled"`
		MfaSecret              string         `json:"mfaSecret"`
		MfaRecoveryCodes       string         `json:"mfaRecoveryCodes"`
		LastActiveAt           time.Time      `json:"lastActiveAt"`
		RoleSlug               string         `json:"roleSlug"`
	}
	devORM := orm.Load(&User{}, orm.WithDatabasePrefix("LYZNFLOW"))

	var du []User
	if err := devORM.GetByFieldEquals("Email", "kartikdd90@gmail.com").Scan(&du); err != nil {
		log.Println(err)
	}
	utils.PrintAsJson(du)
	// tests.GoogleAuth()
	// tests.TestKarmaErrorPackage()
	// tests.TestKarmaFiles()
	// tests.TestAmplifyFuncs()
	// tests.TestECSIntegration()
	// tests.TestDBConnection()
	// tests.TestKarmaParser()
	// tests.TestAICodeParser()
	// tests.TestS3Upload()
	// transcribe.StartStream()
	// tests.TestORMCaching()
	// tests.TestAPIGen()
}
