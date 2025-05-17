package main

import "github.com/MelloB1989/karma/tests"

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
	// tests.TestSendingSingleMail()
	// tests.ORMTest()
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
	tests.TestORMCaching()
	// tests.TestAPIGen()
}
