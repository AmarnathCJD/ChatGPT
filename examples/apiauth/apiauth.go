package examples

import (
	"fmt"

	"github.com/amarnathcjd/chatgpt"
)

func main() {
	gpt := chatgpt.NewClient(&chatgpt.Config{
		ApiKey: "sk-xxxxxxxx",
	})
	if err := gpt.Start(); err != nil {
		panic(err)
	}
	response, err := gpt.Ask("Hello")
	if err != nil {
		panic(err)
	}
	fmt.Println(response)

	// convID := response.ConversationID
}
