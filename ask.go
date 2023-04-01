package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
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

// ChatResponse represents the response returned by this client.
type ChatResponse struct {
	Message        string `json:"message,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	ParentID       string `json:"parent_id,omitempty"`
	Model          string `json:"model,omitempty"`
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
func (c *Client) Ask(question string, conversationID, parentID string) (*ChatResponse, error) { // TODO: Add support for streamChannel
	if c.authmode == AccessTokenMode {
		return c.AskWithAccessToken(question, conversationID, parentID)
	}
	var conversation Conversation

	// Use "default" as the conversation ID if none is provided.
	if conversationID == "" {
		conversationID = "default"
	}

	// If there's no existing conversation with the given ID, create a new one with a system message.
	if _, ok := c.conversations[conversationID]; !ok {
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
		c.conversations[conversationID] = conversation
	} else { // Otherwise, retrieve the existing conversation and add the user's message to it.
		conversation = c.conversations[conversationID]
		conversation.addMessage(Message{
			Role:    "user",
			Content: question,
		})
		c.conversations[conversationID] = conversation
	}

	// Check the number of tokens in the conversation and tokenize it if necessary.
	tokens := conversation.getTokenCount()
	if tokens > getEngineTokenLimit(c.engine) {
		conversation.tokenizeMessage(c.engine)
		c.conversations[conversationID] = conversation
	}

	// Send the conversation messages to OpenAI API and return its response/error.
	response, err := c.askOpenAI(conversation.Messages, nil)
	if err == nil {
		// If there was no error, add the response message to the conversation and update it.
		conversation.addMessage(Message{
			Role:    "assistant",
			Content: response.GetResponse(),
		})
		c.conversations[conversationID] = conversation
	}
	return &ChatResponse{
		Message:        response.GetResponse(),
		ConversationID: conversationID,
		Model:          c.engine,
	}, err
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
		Model:       c.engine,
		Messages:    messages,
		Temperature: c.temperature,
		TopP:        1.0,
	}
	jsonified, _ := json.Marshal(payload)
	return string(jsonified)
}

// setHeaders sets the Authorization and Content-Type headers on the given request.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.auth.apiKey)
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

// AskWithAccessToken sends a question to Custom API using the specified conversation ID or the default one.
func (c *Client) AskWithAccessToken(question string, conversationID, parentID string) (*ChatResponse, error) {
	// Construct the base URL of the Custom API endpoint.
	var base_url = c.baseUrl + "conversation"

	// Create a data map to hold the message information and other metadata.
	data := map[string]interface{}{
		"action": "next", // This indicates that we're requesting the next response from the chat server.
		"messages": []map[string]interface{}{ // An array of message objects consisting of user input.
			{
				"id":   genUUID(), // Generate a unique ID for this message.
				"role": "user",    // Set the role of this message to "user".
				"content": map[string]interface{}{ // The content of the message object.
					"content_type": "text",             // The type of message content - text in this case.
					"parts":        []string{question}, // The actual message content - an array of strings.
				},
			},
		},
		"model": c.engine, // Specify the GPT-3 model to use for generating the response.
	}

	// Add conversationID and parentID metadata to the data object if they were specified.
	if conversationID != "" {
		data["conversation_id"] = conversationID
	}
	if parentID != "" {
		data["parent_message_id"] = parentID
	} else {
		parentID = genUUID()
		data["parent_message_id"] = parentID
	}

	// Convert the data map to a JSON string.
	jsonified, _ := json.Marshal(data)

	// Create an HTTP POST request with the JSON payload.
	req, _ := http.NewRequest("POST", base_url, io.NopCloser(strings.NewReader(string(jsonified))))

	// Set the authorization and content-type headers.
	req.Header.Set("authorization", "Bearer "+c.auth.accessToken)
	req.Header.Set("content-type", "application/json")

	// Send the request and handle the response.
	if resp, err := c.httpx.Do(req); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()

		// If the response status code is 200 OK, parse the response and return it.
		if resp.StatusCode == 200 {
			parsedResp, err := c.parseResponse(resp)
			if err != nil {
				return nil, err
			}
			parsedResp.ParentID = parentID
			return parsedResp, nil
		} else {
			b, _ := io.ReadAll(resp.Body)
			return nil, &ChatError{
				Message: string(b),
				Code:    resp.StatusCode,
			}
		}
	}
}

// RawResponse is a struct representing the raw JSON response received from a chat server.
type RawResponse struct {
	// The message object containing the ID and content of the response.
	Message struct {
		ID      string   `json:"id"` // The unique identifier for the message.
		Content struct { // The raw message content.
			ContentType string   `json:"content_type"` // The content type of the message, e.g., text, audio, etc.
			Parts       []string `json:"parts"`        // The parts that make up the actual message content.
		} `json:"content"`
	} `json:"message"`

	// The conversation ID associated with the response.
	ConversationID string `json:"conversation_id"`

	// Any error information associated with the response.
	Error any `json:"error"`
}

// parseResponse parses the raw string response received from the chat server and returns a ChatResponse object
// containing the extracted message text and conversation ID, if everything went well.
func (c *Client) parseResponse(resp *http.Response) (*ChatResponse, error) {
	// Initialize some variables for parsing the response.
	last_message := ""
	scanner := bufio.NewScanner(resp.Body)

	// Iterate over each line of the response.
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines and event-type headers.
		if line == "" || strings.HasPrefix(line, "event: ") {
			continue
		}

		// Trim the "data: " prefix off the line and exit the loop if line contains the "[DONE]" sentinel value.
		line = strings.TrimPrefix(line, "data: ")
		if line == "[DONE]" {
			break
		}

		// Store the last non-empty line read so far.
		last_message = line
	}

	// Parse the extracted (last) line as a RawResponse object.
	var parsedLine RawResponse
	if err := json.Unmarshal([]byte(last_message), &parsedLine); err != nil {
		return nil, err
	}

	// Handle errors using possible error message content and return appropriate ChatError objects.
	if parsedLine.Error != nil {
		return nil, &ChatError{
			Message: fmt.Sprintf("%v", parsedLine.Error),
			Code:    500,
		}
	}
	if parsedLine.Message.Content.ContentType != "text" {
		return nil, &ChatError{
			Message: "Message content type not supported, " + parsedLine.Message.Content.ContentType,
			Code:    500,
		}
	}
	if len(parsedLine.Message.Content.Parts) == 0 {
		return nil, &ChatError{
			Message: "No message content",
			Code:    500,
		}
	}

	// Return a new ChatResponse object with the extracted message text and conversation ID.
	return &ChatResponse{Message: parsedLine.Message.Content.Parts[0], ConversationID: parsedLine.ConversationID}, nil
}

// genUUID generates a random UUID.
func genUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}
