package examples

import (
	"fmt"

	"github.com/amarnathcjd/chatgpt"
)

func main() {
	gpt := chatgpt.NewClient(&chatgpt.Config{
		APIKey: "sk-XXXX",
		// Engine: "gpt-3.5-turbo",
		// Temperature: 0.9,
		// EnableInternet: true, // TODO: implement this
		// Stream: true, // TODO: implement this
		// Proxy: &url.URL{Host: ""}
	})

	conversationID := "new-conversation1"

	for {
		var promt string
		fmt.Println("Enter your message: ")
		fmt.Scanln(&promt)
		response, err := gpt.Ask(conversationID, promt)
		if err != nil {
			panic(err)
		}
		fmt.Println(response.GetResponse())
	}
}
