package examples

import (
	"context"
	"fmt"

	"github.com/amarnathcjd/chatgpt"
)

func main() {
	gpt := chatgpt.NewClient(&chatgpt.Config{
		Email:    "yourmail@domain.com",
		Password: "yourpassword",
	}, "default-session") // you can use any session name
	if err := gpt.Start(); err != nil {
		panic(err)
	}
	ctx := context.Background()
	response, err := gpt.Ask(ctx, "Hello")
	if err != nil {
		panic(err)
	}
	convID := response.ConversationID
	parentID := response.ParentID

	fmt.Println(response)

	response, err = gpt.Ask(ctx, "How are you?", chatgpt.AskOpts{ConversationID: convID, ParentID: parentID})
	// continue the conversation with the same conversation ID and parent ID
	// to keep the conversation going

	if err != nil {
		panic(err)
	}

	fmt.Println(response)
}
