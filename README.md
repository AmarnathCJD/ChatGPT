# CHATGPT

A Wrapper for the CHATGPT model from OpenAI.
Purely built on Golang.

## Table of Contents

- [CHATGPT](#chatgpt)
  - [Features](#features)
  - [Documentation](#documentation)
  - [Installation](#installation)
  - [Usage](#usage)
  - [License](#license)


## Features

- [x] Ask a question and get a response
- [x] Remember the context of the conversation
- [x] Export the conversation to a file
- [x] Import the conversation from a file
- [x] Get the conversation history
- [x] Models: `davinci`, `gpt 3`, `gpt 3.5`, `gpt 4`
- [x] API Key Authentication
- [x] Customizable temperature of the model
- [x] Inbuilt Tokenizer for the model
- [x] http/https proxy support for the client

## TODO

- [ ] Add support for the `top_p` parameter
- [ ] Add support for authentication using the `access_token`
- [ ] Add support for streaming the response via a channel
- [ ] Add support for email/password authentication

## Documentation

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathcjd/chatgpt.svg)](https://pkg.go.dev/github.com/amarnathcjd/chatgpt)

## Installation

```bash
go get github.com/amarnathcjd/chatgpt
```

## Usage

```go
package main

import (
    "fmt"

    "github.com/amarnathcjd/chatgpt"
)

func main() {
    client := chatgpt.NewClient(&chatgpt.Config{
        APIKey: "sk-xxxxxxxx",
    })
    ask, err := client.Ask("Hi, how are you?")
    if err != nil {
        panic(err)
    }
    fmt.Println("ChatGPT: ", ask)
}

```

## License

[MIT](https://choosealicense.com/licenses/mit/)
