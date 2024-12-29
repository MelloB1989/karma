package bedrock

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

var brc *bedrockruntime.Client
var verbose *bool

func StartChatSession() {
	brc = createClient()
	verbose = flag.Bool("verbose", false, "setting to true will log messages being exchanged with LLM")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	var chatHistory string

	for {
		fmt.Print("\nEnter your message: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		msg := chatHistory + fmt.Sprintf(claudePromptFormat, input)

		response, err := send(msg)

		if err != nil {
			log.Fatal(err)
		}

		chatHistory = msg + response

		// fmt.Println("\n--- Response ---")
		// fmt.Println(response)
	}
}

const claudePromptFormat = "\n\nHuman: %s\n\nAssistant:"

func send(msg string) (string, error) {

	if *verbose {
		fmt.Println("[sending message]", msg)
	}

	stream, err := PromptModelStream(msg, 0.1, 0.9, 50, "meta.llama3-8b-instruct-v1:0")
	if err != nil {
		return "", err
	}
	var response string
	fmt.Println("\n--- Response ---")
	_, err = ProcessStreamingOutput(stream, func(ctx context.Context, part Generation) error {
		fmt.Print(string(part.Generation))
		response += string(part.Generation)
		return nil
	})
	return response, err
}
