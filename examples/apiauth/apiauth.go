package examples

import (
	"context"
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
	ctx := context.Background()
	response, err := gpt.Ask(ctx, "Hello")
	if err != nil {
		panic(err)
	}
	fmt.Println(response)

	// convID := response.ConversationID
}
