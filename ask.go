package chatgpt

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// The OpenAI API endpoint for chat completions.
const OPENAI_HOST = "https://api.openai.com/v1/chat/completions"

// The default "system" message when starting a new conversation.
const DEFAULT_INIT_MESSAGE = "You are chatGPT, trained on a very huge dataset of conversations. Act conversationally"

// Choice represents a possible response and its finish reason from OpenAI's API.
type Choice struct {
	Message      Message `json:"message,omitempty"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// OpenAIResponse represents the response returned by OpenAI's API.
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []Choice `json:"choices"`
}

// GetResponse returns the response message from the OpenAI API response.

func (r *OpenAIResponse) GetResponse() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content
	}
	return ""
}

// OpenAIError represents an error returned by OpenAI's API.
type OpenAIError struct {
	ErrorData struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// ChatError represents a chat-specific error returned by this client.
type ChatError struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// Error returns the string representation of a ChatError.
func (e *ChatError) Error() string {
	return e.Message + " (code: " + strconv.Itoa(e.Code) + ")"
}

// Ask sends a question to OpenAI API using the specified conversation ID or the default one.
func (c *Client) Ask(question string, conversationID ...string) (*OpenAIResponse, error) { // TODO: Add support for streamChannel
	var conversation Conversation

	// Use "default" as the conversation ID if none is provided.
	if len(conversationID) == 0 {
		conversationID = append(conversationID, "default")
	}

	// If there's no existing conversation with the given ID, create a new one with a system message.
	if _, ok := c.conversations[conversationID[0]]; !ok {
		conversation = Conversation{}
		initMessage := Message{
			Role:    "system",
			Content: DEFAULT_INIT_MESSAGE,
		}
		// If a custom init message is provided, use it instead of the default one.
		if c.initMessage != "" {
			initMessage.Content = c.initMessage
		}
		conversation.initMessage(initMessage)
		c.conversations[conversationID[0]] = conversation
	} else { // Otherwise, retrieve the existing conversation and add the user's message to it.
		conversation = c.conversations[conversationID[0]]
		conversation.addMessage(Message{
			Role:    "user",
			Content: question,
		})
		c.conversations[conversationID[0]] = conversation
	}

	// Check the number of tokens in the conversation and tokenize it if necessary.
	tokens := conversation.getTokenCount()
	if tokens > getEngineTokenLimit(c.Engine) {
		conversation.tokenizeMessage(c.Engine)
		c.conversations[conversationID[0]] = conversation
	}

	// Send the conversation messages to OpenAI API and return its response/error.
	return c.askOpenAI(conversation.Messages, nil)
}

// askOpenAI makes a POST request to OpenAI's API with the given messages, and returns the response.
// If there is an HTTP error or a non-200 status code, an error is returned instead.
func (c *Client) askOpenAI(messages []Message, streamChannel chan string) (*OpenAIResponse, error) {
	// Create a new request with the payload and headers set.
	req, _ := http.NewRequest("POST", OPENAI_HOST, strings.NewReader(c.makePayload(messages)))
	c.setHeaders(req)

	// Send the request and handle the response.
	if resp, err := c.httpx.Do(req); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			// If the response has a 200 status code, parse it as an OpenAIResponse.
			var response OpenAIResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				return nil, err
			}
			return &response, nil
		} else {
			// If the response has an error status code, parse it as an OpenAIError and create a ChatError from it.
			var response OpenAIError
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				return nil, err
			}
			return nil, &ChatError{
				Message: response.ErrorData.Message,
				Code:    response.ErrorData.Code,
			}
		}
	}
}

// Payload represents the structure of the JSON payload sent by askOpenAI.
type Payload struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	TopP        float64   `json:"top_p"`
}

// makePayload returns the JSON payload for the given messages with the client's settings.
func (c *Client) makePayload(messages []Message) string {
	payload := Payload{
		Model:       c.Engine,
		Messages:    messages,
		Temperature: c.temperature,
		TopP:        1.0,
	}
	jsonified, _ := json.Marshal(payload)
	return string(jsonified)
}

// setHeaders sets the Authorization and Content-Type headers on the given request.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

// getEngineTokenLimit retrieves the maximum token limit for the given engine.
// It searches through ENGINES to find the one that matches the engine prefix in the client's settings.
// If no match is found, a default limit of 4000 tokens is used.
func getEngineTokenLimit(engine string) int {
	for a, e := range Engines {
		if strings.Contains(engine, a) {
			return e
		}
	}
	return 4000
}
