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
	if r.Choices == nil {
		return "malformed response"
	}
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
}

// ChatError represents a chat/auth-specific error returned by this client.
type ChatError struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// Error returns the string representation of a ChatError.
func (e *ChatError) Error() string {
	var message struct {
		Detail string `json:"detail"`
	}
	json.Unmarshal([]byte(e.Message), &message)
	return "chatgpt error: " + message.Detail + " (error code " + strconv.Itoa(e.Code) + ")"
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
		conversation.addMessage(Message{
			Role:    "user",
			Content: prompt,
		}) // add current message to the conversation flow
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
	// Check if the client has been started and is using access token mode
	if !c.auth.clientStarted {
		return nil, fmt.Errorf("client is not started, call Start() first")
	}
	if c.authmode == AccessTokenMode {
		// Create a new channel for the response messages
		newChannel := make(chan *ChatResponse, 60)

		// Call the askStreamWithAccessToken method to send the question and stream the response
		return newChannel, c.askStreamWithAccessToken(ctx, prompt, newChannel, askOpts...)
	}
	// If the client is not using access token mode, return an error
	return nil, fmt.Errorf("streaming is not yet implemented for API key mode")
}

// AskInternet sends a question to the specified internet engine and returns the response/error.
func (c *Client) AskInternet(ctx context.Context, prompt string) (*ChatResponse, error) {
	// Check if the client has been started
	if !c.auth.clientStarted {
		return nil, fmt.Errorf("client is not started, call Start() first")
	}

	// Format the prompt as a query to an internet search engine.
	query_fmt := "This is a prompt from a user to a chatbot: '%s'. Respond with 'none' if it is directed at the chatbot or cannot be answered by an internet search. Otherwise, respond with a possible search query to a search engine. Do not write any additional text. Make it as minimal as possible"

	// Send the prompt to the ChatGPT engine and get a response.
	response, err := c.Ask(ctx, fmt.Sprintf(query_fmt, prompt))
	if err != nil {
		return nil, err
	}

	// If the response is "none", return an error indicating that an internet search query could not be found.
	if response.Message == "none" {
		return nil, fmt.Errorf("no internet search query found")
	}

	// Otherwise, send the response to an internet search engine and return the resulting snippet as the response.
	response.Message, err = c.askInternet(ctx, prompt, response.Message)
	if err != nil {
		return nil, err
	}
	response, err = c.Ask(ctx, response.Message)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// InternetResponse is a slice of structs representing search results obtained from an internet search engine.
type InternetResponse []struct {
	// The title of the search result.
	Title string `json:"title"`

	// The URL of the search result.
	Link string `json:"link"`

	// A brief snippet of text from the search result.
	Snippet string `json:"snippet"`
}

// askInternet queries the internet using the specified query and returns the response or an error.
func (c *Client) askInternet(ctx context.Context, actual_qn, query_fmt string) (string, error) {
	// Remove any prefix indicating that the query is a search query.
	query_fmt = strings.ReplaceAll(query_fmt, "Possible search query: ", "")

	// Set the URL and payload for the external search API.
	query_url := "https://ddg-api.herokuapp.com/search"
	var query_payload struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	query_payload.Query = query_fmt
	query_payload.Limit = 3

	// Marshal the query payload to JSON and create an HTTP request with the JSON payload.
	query_json, _ := json.Marshal(query_payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", query_url, strings.NewReader(string(query_json)))
	req.Header.Add("Content-Type", "application/json")

	// Send the request and handle the response.
	if resp, err := c.httpx.Do(req); err != nil {
		// If there is an error sending the request, return the error.
		return "", err
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			// If the response status code is not 200 (OK), parse the error response from the API and return a ChatError.
			var errResp struct {
				Error string `json:"error"`
			}
			respBody, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(respBody, &errResp); err != nil {
				return "", fmt.Errorf("error: %s", resp.Status)
			}
			return "", &ChatError{errResp.Error, resp.StatusCode}
		}
		// If the response status code is 200, parse the response body as an InternetResponse and return the snippet from the first search result.
		respBody, _ := io.ReadAll(resp.Body)
		var response InternetResponse
		if err := json.Unmarshal(respBody, &response); err != nil {
			return "", fmt.Errorf("error: %s", resp.Status)
		}
		snippets := make([]string, 0)
		for _, result := range response {
			snippets = append(snippets, result.Snippet)
		}
		query_fmt = fmt.Sprintf("Here is the piece of data: %s, formulate an answer to the prompt in chatgpt style for the prompt: %s based solely on the data provided. dont mention anything about the source of the answer in answer, also try to minimise the length of answer as far as possible, if necessary split answer into paragraphs", strings.Join(snippets, " "), query_fmt)
		return query_fmt, nil
	}
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
	// The ID of the OpenAI model to use for the request.
	Model string `json:"model"`

	// An array of Message structs representing the conversation so far.
	Messages []Message `json:"messages"`

	// The "temperature" parameter controls the "creativity" of the AI's responses. Higher values will generate more diverse and unexpected responses.
	Temperature float64 `json:"temperature"`

	// The "top-p" parameter controls the "conservatism" of the AI's responses. Lower values will generate more predictable and "safe" responses.
	TopP float64 `json:"top_p"`
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

// getEngineTokenLimit returns the maximum number of tokens that can be sent to the OpenAI API for a given engine.
func getEngineTokenLimit(engine string) int {
	// If the engine is "gpt-4-32k", return a limit of 32000 tokens.
	if engine == "gpt-4-32k" {
		return 32000
	} else if engine == "gpt-4" { // If the engine is "gpt-4", return a limit of 8000 tokens.
		return 8000
	} else {
		return 4000 // default to 4000 tokens
	}
}

// askWithAccessToken sends a question to Custom API using the specified conversation ID or the default one.
func (c *Client) askWithAccessToken(ctx context.Context, prompt string, askOpts ...AskOpts) (*ChatResponse, error) {
	var conversationId string
	var parentId string

	// Parse the conversation ID and parent ID from the askOpts parameter, if provided
	if len(askOpts) > 0 {
		if askOpts[0].ConversationID != "" {
			conversationId = askOpts[0].ConversationID
		}
		if askOpts[0].ParentID != "" {
			parentId = askOpts[0].ParentID
		}
	}

	// Construct the payload for the POST request
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

	// Add the conversation ID and parent ID to the payload, if provided
	if conversationId != "" {
		data["conversation_id"] = conversationId
	}

	if parentId != "" {
		data["parent_message_id"] = parentId
	} else {
		data["parent_message_id"] = genUUID()
	}

	// Convert the payload to JSON and create a new HTTP request
	payload, _ := json.Marshal(data)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseUrl, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("system error: %w", err)
	}

	// Set the authorization header using the access token
	c.setHeaders(req, c.auth.accessToken)

	// Send the HTTP request and handle the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("system error: %w", err)
	}

	// Close the response body when we're done with it
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Parse the response body and return the last message in the conversation
		msgs, err := c.parseResponse(resp.Body, nil)
		if err != nil {
			return nil, err
		}

		if len(msgs) > 0 {
			return msgs[len(msgs)-1], nil
		}
	}

	// If the API returned an error, return a ChatError containing the error message and HTTP status code
	body, _ := io.ReadAll(resp.Body)
	return nil, &ChatError{Message: string(body), Code: resp.StatusCode}
}

// askStreamWithAccessToken sends a question to Custom API using the specified conversation ID or the default one.
func (c *Client) askStreamWithAccessToken(ctx context.Context, prompt string, ch chan *ChatResponse, askOpts ...AskOpts) error {
	var conversationId string
	var parentId string

	// Parse the conversation ID and parent ID from the askOpts parameter, if provided
	if len(askOpts) > 0 {
		if askOpts[0].ConversationID != "" {
			conversationId = askOpts[0].ConversationID
		}
		if askOpts[0].ParentID != "" {
			parentId = askOpts[0].ParentID
		}
	}

	// Construct the payload for the POST request
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

	// Add the conversation ID and parent ID to the payload, if provided
	if conversationId != "" {
		data["conversation_id"] = conversationId
	}

	if parentId != "" {
		data["parent_message_id"] = parentId
	} else {
		data["parent_message_id"] = genUUID()
	}

	// Convert the payload to JSON and create a new HTTP request
	payload, _ := json.Marshal(data)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseUrl, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("system error: %w", err)
	}

	// Set the authorization header using the access token
	c.setHeaders(req, c.auth.accessToken)

	// Send the HTTP request and handle the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("system error: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		// Parse the response body and send any messages to the channel
		_, err := c.parseResponse(resp.Body, ch)
		return err
	} else {
		// Return a ChatError containing the HTTP status code
		return &ChatError{Code: resp.StatusCode}
	}
}

// parseResponse parses the response body and returns a list of ChatResponse, or an error if the response is not valid
func (c *Client) parseResponse(response io.ReadCloser, streamChannel chan *ChatResponse) ([]*ChatResponse, error) {
	// Create an empty slice to store ChatResponse objects
	messages := make([]*ChatResponse, 0)
	var err error

	// Create a scanner to read the response body
	scanner := bufio.NewScanner(response)

	// If the first line contains {"detail": }, return an error
	if scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `{"detail":`) {
			message := regexp.MustCompile(`{"detail":.*}`).FindString(line)
			return nil, fmt.Errorf(message)
		}
	}

	// If streamChannel is not nil, start scanning the response body in a separate goroutine
	if streamChannel != nil {
		go c.startScan(scanner, streamChannel, response)
	} else {
		// Otherwise, scan the response body synchronously and store the messages in the messages slice
		messages, err = c.startScan(scanner, nil, response)
	}

	// Return the messages slice and any errors
	return messages, err
}

// startScan starts the scan of the response body
// if streamChannel is not nil, it will send the messages to the channel as they are received
func (c *Client) startScan(scanner *bufio.Scanner, streamChannel chan *ChatResponse, respBody io.ReadCloser) ([]*ChatResponse, error) {
	var messages []*ChatResponse
	defer respBody.Close()

	// Loop through each line in the response body
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and lines that start with "event: "
		if line == "" || strings.HasPrefix(line, "event: ") {
			continue
		}

		// Skip lines that do not start with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Handle error messages that contain {"detail": }
		if strings.Contains(line, `{"detail":`) {
			message := regexp.MustCompile(`{"detail":.*}`).FindString(line)
			return nil, fmt.Errorf(message)
		}

		// Remove "data: " prefix from line
		line = strings.TrimPrefix(line, "data: ")

		// Stop scanning if line is "[DONE]"
		if line == "[DONE]" {
			break
		}

		// Parse the line as JSON and check if it contains the necessary fields
		var parsedLine map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsedLine); err != nil || !checkFields(parsedLine) {
			continue
		}

		// Extract message content and check if it is of type "text"
		content := parsedLine["message"].(map[string]interface{})["content"].(map[string]interface{})
		if messageContextType, ok := content["content_type"].(string); ok && messageContextType == "text" {
			parts := content["parts"].([]interface{})

			// Only process messages that have at least one part
			if len(parts) > 0 {
				message := fmt.Sprintf("%v", parts[0])
				conversationID := parsedLine["conversation_id"].(string)
				parentID := parsedLine["message"].(map[string]interface{})["id"].(string)

				// If streamChannel is not nil, send the message to the channel
				if streamChannel != nil && message != "" {
					streamChannel <- &ChatResponse{
						ConversationID: conversationID,
						ParentID:       parentID,
						Message:        strings.TrimSpace(message),
					}
					continue
				}

				// Add the message to the messages slice
				messages = append(messages, &ChatResponse{
					ConversationID: conversationID,
					ParentID:       parentID,
					Message:        strings.TrimSpace(message),
				})
			}
		} else {
			// Log a warning for unsupported message types
			c.logger.Warn("Unsupported message type: " + messageContextType)
		}
	}

	// Close the streamChannel and return the messages slice
	if streamChannel != nil {
		close(streamChannel)
	}
	return messages, nil
}

// checkFields checks if the necessary fields exist in the parsed line map
func checkFields(parsedLine map[string]interface{}) bool {
	// Check if "message" field exists in parsedLine map
	message, messageExists := parsedLine["message"].(map[string]interface{})
	if !messageExists {
		return false
	}
	// Check if "content" field exists in "message" map
	content, contentExists := message["content"].(map[string]interface{})
	if !contentExists {
		return false
	}
	// Check if "parts" field exists in "content" map
	parts, partsExists := content["parts"].([]interface{})
	if !partsExists {
		return false
	}
	// Check if "parts" slice is not empty
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
