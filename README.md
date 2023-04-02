<p align="center">
    <a href="https://github.com/amarnathcjd/chatgpt">
        <img src="https://i.imgur.com/isfTY5X.png" alt="ChatGPT" width="128">
    </a>
    <br>
    <b>ChatGPT - A Golang wrapper for the GPT model from OpenAI</b>
</p>

# ChatGPT

**ChatGPT** is a wrapper for the `GPT` model from OpenAI purely built on Golang. The application has various features, including asking questions and getting responses, remembering the context of the conversation, exporting the conversation to a file, and importing the conversation from a file. 

The app also allows users to get the conversation history, use multiple models such as `davinci`, `gpt 3`, `gpt 3.5`, and `gpt 4`. Other features include API Key Authentication, Customizable temperature of the model, Inbuilt Tokenizer for the model, and http/https proxy support for the client. 

## Table of Contents

- [CHATGPT](#chatgpt)
  - [Features](#features)
  - [Documentation](#documentation)
  - [Installation](#installation)
  - [Usage](#usage)
  - [License](#license)

## Features
Here are some notable features of the wrapper:

- Authenticate using the API Key, which can be obtained from the [OpenAI Dashboard](https://beta.openai.com/).
- Authenticate using the `access_token` or `email/password`.
- Cache the authentication token for future use.
- Multiple Accounts support, with same cache file.
- Remember the context of the conversation.
- Export the conversation to a file.
- Get the conversation history.
- Models: `davinci`, `gpt 3`, `gpt 3.5`, `gpt 4`.
- API Key Authentication.
- Customizable temperature of the model.
- Inbuilt Tokenizer for the model.
- https Proxy support for the client.

## TODO
Below are some things that need to be added to the application:

- Implement the `internet plugin` for the `GPT` model.
- Add support for the `top_p` parameter.
- Add support for streaming the response via a channel.

## Documentation

You can find the complete documentation for the `CHATGPT` package at [Go Reference](https://pkg.go.dev/github.com/amarnathcjd/chatgpt).

## Installation

You can install ChatGPT by running the following command on your terminal:

```bash
go get github.com/amarnathcjd/chatgpt
```

## Usage

Here is an example of how to use ChatGPT package:

```go
package main

import (
    "fmt"
    "context"

    "github.com/amarnathcjd/chatgpt"
)

func main() {
    client := chatgpt.NewClient(&chatgpt.Config{
        Email: "email@domain.com",
        Password: "password", // or ApiKey: "sk-xxxxxxxxxxxx",
    }, "session-1")
    if err := client.Start(); err != nil {
        panic(err) 
    }
    ask, err := client.Ask(context.Background(), "Hello, nice to meet you")
    if err != nil {
        panic(err)
    }
    fmt.Println("ChatGPT: ", ask.Message)
}

```

More examples can be found in the [examples folder](https//github.com/amarnathcjd/chatgpt/tree/master/examples).

## License

`CHATGPT` is released under the terms of the GPT-3 License. See [LICENSE](https://github.com/amarnathcjd/chatgpt/blob/master/LICENSE) for more information or see https://openai.com/terms.
