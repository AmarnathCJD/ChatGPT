<p align="center">
    <a href="https://github.com/amarnathcjd/chatgpt">
        <img src="https://i.imgur.com/isfTY5X.png" alt="ChatGPT" width="128">
    </a>
    <br>
    <b>ChatGPT - A Golang wrapper for the GPT model from OpenAI</b>
</p>

# ChatGPT


[![GoDoc](https://godoc.org/github.com/amarnathcjd/chatgpt?status.svg)](https://godoc.org/github.com/amarnathcjd/chatgpt)
[![Go Report Card](https://goreportcard.com/badge/github.com/amarnathcjd/chatgpt)](https://goreportcard.com/report/github.com/amarnathcjd/chatgpt)
[![GitHub license](https://img.shields.io/github/license/amarnathcjd/chatgpt)](nhttps://github.com/amarnathcjd/chatgpt/blob/master/LICENSE)
[![GitHub Actions](https://github.com/tucnak/telebot/actions/workflows/go.yml/badge.svg)](https://github.com/tucnak/telebot/actions)
[![Discuss on Telegram](https://img.shields.io/badge/telegram-discuss-0088cc.svg)](https://t.me/rosexchat)

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
- Support for streaming the response via a channel.

## TODO
Below are some things that need to be added to the application:

- Implement the `internet plugin` for the `GPT` model.
- Add support for the `top_p` parameter.

## Documentation

You can find the complete documentation for the `CHATGPT` package at [GoDoc](https://godoc.org/github.com/amarnathcjd/chatgpt)
## Installation

You can install ChatGPT by running the following command on your terminal:

```bash
go get github.com/amarnathcjd/chatgpt
```
Or usking gopkg.in:

```bash
go get gopkg.in/chatgpt.v1
```

## Usage

Here is an example of how to use ChatGPT package:

```go
package chatgpt

import (
    "fmt"
    "context"

    "github.com/amarnathcjd/chatgpt"
)

func main() {
    client := chatgpt.NewClient(&chatgpt.Config{
        Email: "email@domain.com",
        Password: "password", // or ApiKey: "sk-xxxxxxxxxxxx",
    })
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

ChatGPT is released under the terms of the GPT-3 License. See [LICENSE](https://github.com/amarnathcjd/chatgpt/blob/master/LICENSE) for more information or see [The GPT-3 License](https://openai.com/blog/gpt-3-license/).

