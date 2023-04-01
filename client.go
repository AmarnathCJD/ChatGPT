package chatgpt

import (
	"fmt"
	"net/http"
	"net/url"
)

// Client represents a connection to the OpenAI API.
// It contains the client's API key, access token, HTTP client, conversation history, settings, and stream details.
type Client struct {
	apiKey         string                  // The API key used for authentication with OpenAI.
	accessToken    string                  // The access token used for conversations with OpenAI.
	httpx          *http.Client            // The HTTP client used for sending requests to OpenAI.
	conversations  map[string]Conversation // A map of conversation IDs to Conversation objects.
	temperature    float64                 // The sampling temperature for generating text.
	Engine         string                  // The name of the GPT model being used by this client.
	initMessage    string                  // The initial message sent to start a new conversation.
	enableInternet bool                    // Whether or not to allow the use of external websites in responses.
	stream         bool                    // Whether or not to stream response messages as they come in.
	proxy          *url.URL                // The URL of the proxy server to use for requests.
}

// Config represents the configuration options for a connection to the OpenAI API.
// Each field is optional and can be omitted from the JSON representation of the config object.
type Config struct {
	APIKey         string   `json:"api_key,omitempty"`         // The API key used for authentication with OpenAI.
	AccessToken    string   `json:"access_token,omitempty"`    // The access token used for conversations with OpenAI.
	Engine         string   `json:"engine,omitempty"`          // The name of the GPT model being used.
	InitMessage    string   `json:"init_message,omitempty"`    // The initial message sent to start a new conversation.
	Temperature    float64  `json:"temperature,omitempty"`     // The sampling temperature for generating text.
	EnableInternet bool     `json:"enable_internet,omitempty"` // Whether or not to allow the use of external websites in responses.
	Stream         bool     `json:"stream,omitempty"`          // Whether or not to stream response messages as they come in.
	Proxy          *url.URL `json:"proxy,omitempty"`           // The URL of the proxy server to use for requests.
}

// NewClient creates a new OpenAI API client with the given configuration.
func NewClient(config *Config) *Client {
	// Initialize a new client with default values, then update its fields
	// based on the provided configuration.
	client := &Client{
		apiKey:         config.APIKey,
		accessToken:    config.AccessToken,
		conversations:  make(map[string]Conversation),
		Engine:         config.Engine,
		temperature:    config.Temperature,
		enableInternet: config.EnableInternet,
		stream:         config.Stream,
		httpx:          &http.Client{},
		initMessage:    config.InitMessage,
	}

	// Set default values for missing fields in the configuration.
	if client.temperature == 0 {
		client.temperature = 0.9
	}
	if client.Engine == "" {
		client.Engine = "gpt-3.5-turbo" // default engine
	}

	// Set up a proxy if one is specified in the configuration.
	if config.Proxy != nil {
		client.httpx.Transport = &http.Transport{
			Proxy: http.ProxyURL(config.Proxy),
		}
	}
	return client
}

// SetAPIKey sets the API key used for authentication.
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// SetAccessToken sets the access token used for conversations.
func (c *Client) SetAccessToken(accessToken string) {
	c.accessToken = accessToken
}

// SetEngine sets the GPT model being used.
func (c *Client) SetEngine(engine string) {
	c.Engine = engine
}

// SetEnableInternet sets whether or not external websites can be accessed in responses.
func (c *Client) SetEnableInternet(enableInternet bool) {
	c.enableInternet = enableInternet
}

// SetStream sets whether or not response messages are streamed as they come in.
func (c *Client) SetStream(stream bool) {
	c.stream = stream
}

// SetProxy sets the proxy server to use for requests.
func (c *Client) SetProxy(proxy *url.URL) {
	c.proxy = proxy
}

// GetAPIKey returns the API key used for authentication.
func (c *Client) GetAPIKey() string {
	return c.apiKey
}

// GetAccessToken returns the access token used for conversations.
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

// GetEngine returns the name of the GPT model being used.
func (c *Client) GetEngine() string {
	return c.Engine
}

// GetEnableInternet returns true if external websites can be accessed in responses.
func (c *Client) GetEnableInternet() bool {
	return c.enableInternet
}

// GetStream returns true if response messages are streamed as they come in.
func (c *Client) GetStream() bool {
	return c.stream
}

// GetProxy returns the URL of the proxy server to use for requests.
func (c *Client) GetProxy() *url.URL {
	return c.proxy
}

// GetConversations returns a map of all conversations currently stored in memory.
func (c *Client) GetConversations() map[string]Conversation {
	return c.conversations
}

// GetConversation returns a specific conversation by ID, or an error if it doesn't exist.
func (c *Client) GetConversation(id string) (*Conversation, error) {
	if conv, ok := c.conversations[id]; ok {
		return &conv, nil
	}
	return nil, fmt.Errorf("conversation with id %s not found", id)
}

// ResetConversation deletes a specific conversation by ID, or returns an error if it doesn't exist.
func (c *Client) ResetConversation(id string) error {
	if _, ok := c.conversations[id]; ok {
		delete(c.conversations, id)
		return nil
	}
	return fmt.Errorf("conversation with id %s not found", id)
}

// ResetConversations deletes all conversations from memory.
func (c *Client) ResetConversations() {
	c.conversations = make(map[string]Conversation)
}
