package chatgpt

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// The OpenAI API endpoint for chat completions.
const OPENAI_HOST = "https://api.openai.com/v1/chat/completions"

// The default "system" message when starting a new conversation.
const DEFAULT_INIT_MESSAGE = "You are chatGPT, trained on a very huge dataset of conversations. Act conversationally"

type AskOpts struct {
	// The conversation ID to use for this request. If not specified, a new conversation ID will be generated.
	ConversationID string
	// The parent ID to use for this request. If not specified, a new parent ID will be generated.
	ParentID string
}

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
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
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
	done           bool
}

// Done returns true if the chat response is done.
func (r *ChatResponse) Done() bool {
	return r.done
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
func (c *Client) Ask(ctx context.Context, prompt string, askOpts ...AskOpts) (*ChatResponse, error) { // TODO: Add support for streamChannel
	if !c.auth.clientStarted {
		return nil, fmt.Errorf("client is not started, call Start() first")
	}
	if c.authmode == AccessTokenMode {
		return c.askWithAccessToken(ctx, prompt, askOpts...)
	}
	var conversation Conversation
	var conversationId string

	if len(askOpts) > 0 {
		if askOpts[0].ConversationID != "" {
			conversationId = askOpts[0].ConversationID
		}
	}

	// Use "default" as the conversation ID if none is provided.
	if conversationId == "" {
		conversationId = "default"
	}

	// If there's no existing conversation with the given ID, create a new one with a system message.
	if _, ok := c.conversations[conversationId]; !ok {
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
		c.conversations[conversationId] = conversation
	} else { // Otherwise, retrieve the existing conversation and add the user's message to it.
		conversation = c.conversations[conversationId]
		conversation.addMessage(Message{
			Role:    "user",
			Content: prompt,
		})
		c.conversations[conversationId] = conversation
	}

	// Check the number of tokens in the conversation and tokenize it if necessary.
	tokens := conversation.getTokenCount()
	if tokens > getEngineTokenLimit(c.engine) {
		conversation.tokenizeMessage(c.engine)
		c.conversations[conversationId] = conversation
	}

	// Send the conversation messages to OpenAI API and return its response/error.
	response, err := c.askOpenAI(ctx, conversation.Messages, nil)
	if err == nil {
		// If there was no error, add the response message to the conversation and update it.
		conversation.addMessage(Message{
			Role:    "assistant",
			Content: response.GetResponse(),
		})
		c.conversations[conversationId] = conversation
	}
	return &ChatResponse{
		Message:        response.GetResponse(),
		ConversationID: conversationId,
		Model:          c.engine,
	}, err
}

// AskStream sends a question to OpenAI API using the specified conversation ID or the default one and streams the response.
func (c *Client) AskStream(ctx context.Context, prompt string, askOpts ...AskOpts) (chan *ChatResponse, error) {
	if !c.auth.clientStarted {
		return nil, fmt.Errorf("client is not started, call Start() first")
	}
	if c.authmode == AccessTokenMode {
		newChannel := make(chan *ChatResponse, 60)
		return newChannel, c.askStreamWithAccessToken(ctx, prompt, newChannel, askOpts...)
	}
	return nil, fmt.Errorf("streaming is not yet implemented for API key mode")
}

// askOpenAI makes a POST request to OpenAI's API with the given messages, and returns the response.
// If there is an HTTP error or a non-200 status code, an error is returned instead.
func (c *Client) askOpenAI(ctx context.Context, messages []Message, streamChannel chan string) (*OpenAIResponse, error) {
	// Create a new request with the payload and headers set.
	req, _ := http.NewRequestWithContext(ctx, "POST", OPENAI_HOST, strings.NewReader(c.makePayload(messages)))
	c.setHeaders(req, c.auth.apiKey)

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
func (c *Client) setHeaders(req *http.Request, key string) {
	req.Header.Set("Authorization", "Bearer "+key)
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

// askWithAccessToken sends a question to Custom API using the specified conversation ID or the default one.
func (c *Client) askWithAccessToken(ctx context.Context, prompt string, askOpts ...AskOpts) (*ChatResponse, error) {
	_req_url := c.baseUrl + "conversation"

	var conversationId string
	var parentId string

	if len(askOpts) > 0 {
		if askOpts[0].ConversationID != "" {
			conversationId = askOpts[0].ConversationID
		}
		if askOpts[0].ParentID != "" {
			parentId = askOpts[0].ParentID
		}
	}

	data := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{
			{
				"id":   genUUID(),
				"role": "user",
				"content": map[string]interface{}{
					"content_type": "text",
					"parts":        []string{prompt},
				},
			},
		},
		"model": c.engine,
	}

	if conversationId != "" {
		data["conversation_id"] = conversationId
	}

	if parentId != "" {
		data["parent_message_id"] = parentId
	} else {
		data["parent_message_id"] = genUUID()
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequestWithContext(ctx, "POST", _req_url, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("error in get %s: %w", _req_url, err)
	}
	c.setHeaders(req, c.auth.accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error in get %s: %w", _req_url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {

		msgs, err := c.parseResponse(resp.Body, nil)
		if err != nil {
			return nil, err
		}

		if len(msgs) > 0 {
			return msgs[len(msgs)-1], nil
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return nil, &ChatError{
		Message: string(body),
		Code:    resp.StatusCode,
	}
}

// askStreamWithAccessToken sends a question to Custom API using the specified conversation ID or the default one.
func (c *Client) askStreamWithAccessToken(ctx context.Context, prompt string, ch chan *ChatResponse, askOpts ...AskOpts) error {
	_req_url := c.baseUrl + "conversation"

	var conversationId string
	var parentId string

	if len(askOpts) > 0 {
		if askOpts[0].ConversationID != "" {
			conversationId = askOpts[0].ConversationID
		}
		if askOpts[0].ParentID != "" {
			parentId = askOpts[0].ParentID
		}
	}

	data := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{
			{
				"id":   genUUID(),
				"role": "user",
				"content": map[string]interface{}{
					"content_type": "text",
					"parts":        []string{prompt},
				},
			},
		},
		"model": c.engine,
	}

	if conversationId != "" {
		data["conversation_id"] = conversationId
	}

	if parentId != "" {
		data["parent_message_id"] = parentId
	} else {
		data["parent_message_id"] = genUUID()
	}

	payload, _ := json.Marshal(data)
	req, err := http.NewRequestWithContext(ctx, "POST", _req_url, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("error in get %s: %w", _req_url, err)
	}
	c.setHeaders(req, c.auth.accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error in get %s: %w", _req_url, err)
	}

	if resp.StatusCode == http.StatusOK {
		_, err := c.parseResponse(resp.Body, ch)
		return err
	} else {
		return &ChatError{Code: resp.StatusCode}
	}
}

// parseResponse parses the response body and returns a list of ChatResponse, or an error if the response is not valid
func (c *Client) parseResponse(response io.ReadCloser, streamChannel chan *ChatResponse) ([]*ChatResponse, error) {
	messages := make([]*ChatResponse, 0)
	var err error
	scanner := bufio.NewScanner(response)

	// if { "detail": "..." } is found in first line, return error
	if scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `{"detail":}`) {
			message := regexp.MustCompile(`{"detail":.*}`).FindString(line)
			return nil, fmt.Errorf(message)
		}
	}
	if streamChannel != nil {
		go c.startScan(scanner, streamChannel, response)
	} else {
		messages, err = c.startScan(scanner, nil, response)
	}

	return messages, err
}

// startScan starts the scan of the response body
// if streamChannel is not nil, it will send the messages to the channel as they are received
func (c *Client) startScan(scanner *bufio.Scanner, streamChannel chan *ChatResponse, respBody io.ReadCloser) ([]*ChatResponse, error) {
	var messages []*ChatResponse
	defer respBody.Close()
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "event: ") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		if strings.Contains(line, `{"detail":}`) {
			message := regexp.MustCompile(`{"detail":.*}`).FindString(line)
			return nil, fmt.Errorf(message)
		}
		line = strings.TrimPrefix(line, "data: ")
		if line == "[DONE]" {
			break
		}
		var parsedLine map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsedLine); err != nil || !checkFields(parsedLine) {
			continue
		}
		content := parsedLine["message"].(map[string]interface{})["content"].(map[string]interface{})
		if messageContextType, ok := content["content_type"].(string); ok && messageContextType == "text" {
			parts := content["parts"].([]interface{})
			if len(parts) > 0 {
				message := fmt.Sprintf("%v", parts[0])
				conversationID := parsedLine["conversation_id"].(string)
				parentID := parsedLine["message"].(map[string]interface{})["id"].(string)

				if streamChannel != nil && message != "" {
					streamChannel <- &ChatResponse{
						ConversationID: conversationID,
						ParentID:       parentID,
						Message:        strings.TrimSpace(message),
					}
					continue
				}

				messages = append(messages, &ChatResponse{
					ConversationID: conversationID,
					ParentID:       parentID,
					Message:        strings.TrimSpace(message),
				})
			}
		} else {
			c.logger.Warn("Unsupported message type: " + messageContextType)
		}
	}
	close(streamChannel)
	return messages, nil
}

// checkFields checks if the fields are present in the parsed line.
func checkFields(parsedLine map[string]interface{}) bool {
	message, messageExists := parsedLine["message"].(map[string]interface{})
	if !messageExists {
		return false
	}
	content, contentExists := message["content"].(map[string]interface{})
	if !contentExists {
		return false
	}
	parts, partsExists := content["parts"].([]interface{})
	if !partsExists {
		return false
	}
	if len(parts) == 0 {
		return false
	}
	return true
}

// genUUID generates a random UUID.
func genUUID() string {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
