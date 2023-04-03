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
	if ch, err := gpt.AskStream(ctx, "Hello"); err != nil {
		panic(err)
	} else {
		for i := range ch {
			fmt.Println(i.Message) // stream the response
		}
	}
}
