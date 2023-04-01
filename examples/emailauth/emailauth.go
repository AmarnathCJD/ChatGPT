package examples

import (
	"fmt"

	"github.com/amarnathcjd/chatgpt"
)

func main() {
	gpt := chatgpt.NewClient(&chatgpt.Config{
		Email:    "yourmail@domain.com",
		Password: "yourpassword",
	})
	if err := gpt.Start(); err != nil {
		panic(err)
	}
	response, err := gpt.Ask("Hello", "", "")
	if err != nil {
		panic(err)
	}
	convID := response.ConversationID
	parentID := response.ParentID

	fmt.Println(response)

	response, err = gpt.Ask("How are you?", convID, parentID)
	// continue the conversation with the same conversation ID and parent ID
	// to keep the conversation going

	if err != nil {
		panic(err)
	}

	fmt.Println(response)
}
