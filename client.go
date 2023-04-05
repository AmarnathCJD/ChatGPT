package chatgpt

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

const (
	ApiKeyMode      = iota // Set a value of 0 to ApiKeyMode. This indicates that the user of the program has set up API key properly.
	AccessTokenMode        // Set a value of 1 to AccessTokenMode. This indicates that the user of the program has set up access token properly.
)

// Client represents a connection to the OpenAI API.
// It contains the client's API key, access token, HTTP client, conversation history, settings, and stream details.
type Client struct {
	auth           *Auth                   // The authentication object used for authenticating with OpenAI.
	httpx          *http.Client            // The HTTP client used for sending requests to OpenAI.
	conversations  map[string]Conversation // A map of conversation IDs to Conversation objects.
	temperature    float64                 // The sampling temperature for generating text.
	engine         string                  // The name of the GPT model being used by this client.
	initMessage    string                  // The initial message sent to start a new conversation.
	baseUrl        string                  // Custom base URL for the API.
	enableInternet bool                    // Whether or not to allow the use of external websites in responses.
	stream         bool                    // Whether or not to stream response messages as they come in.
	proxy          *url.URL                // The URL of the proxy server to use for requests.
	authmode       int                     // The authentication mode used by this client.
	ispaid         bool                    // Whether or not the account is a paid account.
	logger         *Logger                 // The logger used for logging messages.
}

// Config represents the configuration options for a connection to the OpenAI API.
// Each field is optional and can be omitted from the JSON representation of the config object.
type Config struct {
	ApiKey         string   `json:"api_key,omitempty"`         // The API key used for authentication with OpenAI.
	Email          string   `json:"email,omitempty"`           // The email used for authentication with OpenAI.
	Password       string   `json:"password,omitempty"`        // The password used for authentication with OpenAI.
	AccessToken    string   `json:"access_token,omitempty"`    // The access token used for conversations with OpenAI.
	Engine         string   `json:"engine,omitempty"`          // The name of the GPT model being used.
	InitMessage    string   `json:"init_message,omitempty"`    // The initial message sent to start a new conversation.
	BaseURL        string   `json:"base_url,omitempty"`        // Custom base URL for the OpenAI API.
	Temperature    float64  `json:"temperature,omitempty"`     // The sampling temperature for generating text.
	LogLevel       LogLevel `json:"log_level,omitempty"`       // The log level to use for logging messages.
	IsPaid         bool     `json:"is_paid,omitempty"`         // Whether or not the account is a paid account.
	EnableInternet bool     `json:"enable_internet,omitempty"` // Whether or not to allow the use of external websites in responses.
	Stream         bool     `json:"stream,omitempty"`          // Whether or not to stream response messages as they come in.
	DisableCache   bool     `json:"disable_cache,omitempty"`   // Whether or not to disable caching of access tokens.
	Proxy          *url.URL `json:"proxy,omitempty"`           // The URL of the proxy server to use for requests.
}

// NewClient creates a new OpenAI API client with the given configuration.
// sessionName is an optional parameter that can be used to identify the client in case of multiple clients.
func NewClient(config *Config, sessionName ...string) *Client {
	// Initialize a new client with default values, then update its fields
	// based on the provided configuration.
	if config == nil {
		config = &Config{}
	}

	client := &Client{
		auth: &Auth{
			email:       config.Email,
			password:    config.Password,
			apiKey:      config.ApiKey,
			accessToken: config.AccessToken,
			enableCache: !config.DisableCache,
		},
		conversations:  make(map[string]Conversation),
		engine:         config.Engine,
		baseUrl:        config.BaseURL,
		temperature:    config.Temperature,
		enableInternet: config.EnableInternet,
		stream:         config.Stream,
		httpx:          &http.Client{},
		initMessage:    config.InitMessage,
		ispaid:         config.IsPaid,
		logger:         &Logger{},
	}

	// Set default values for missing fields in the configuration.
	if client.temperature == 0 {
		client.temperature = 0.9
	}
	if client.engine == "" {
		client.engine = "gpt-3.5-turbo" // default engine
	}
	// set the default base URL if one is not specified in the configuration.
	if client.baseUrl == "" {
		client.baseUrl = "https://bypass.churchless.tech/api/conversation"
	}

	// Set the log level if one is specified in the configuration.
	if config.LogLevel != 0 {
		client.logger.SetLevel(config.LogLevel)
	} else {
		client.logger.SetLevel(LogLevelInfo)
	}

	// Set the session name if one is specified.
	if len(sessionName) > 0 {
		client.auth.sessionName = sessionName[0]
		client.logger.sessionName = sessionName[0]
	} else {
		client.auth.sessionName = "default"
		client.logger.sessionName = "default"
	}

	// Set up a proxy if one is specified in the configuration.
	if config.Proxy != nil {
		client.httpx.Transport = &http.Transport{
			Proxy: http.ProxyURL(config.Proxy),
		}
	}
	return client
}

// SetEmailAndPassword sets the email and password used for authentication.
func (c *Client) SetEmailAndPassword(email, password string) {
	c.auth.email = email
	c.auth.password = password
}

// SetSessionName sets the session name used for authentication.
func (c *Client) SetSessionName(sessionName string) {
	c.auth.sessionName = sessionName
}

// SetAPIKey sets the API key used for authentication.
func (c *Client) SetAPIKey(apiKey string) {
	c.auth.apiKey = apiKey
}

// SetAccessToken sets the access token used for conversations.
func (c *Client) SetAccessToken(accessToken string) {
	c.auth.accessToken = accessToken
}

// SetEngine sets the GPT model being used.
func (c *Client) SetEngine(engine string) {
	c.logger.Debug(fmt.Sprintf("Setting engine to %s", engine))
	c.engine = engine
}

// ToggleInternet toggles whether or not to allow the use of external websites in responses.
func (c *Client) ToggleInternet(t bool) {
	c.logger.Debug(fmt.Sprintf("Setting enableInternet to %t", t))
	c.enableInternet = t
}

// ToggleStream toggles whether or not to stream response messages as they come in.
func (c *Client) ToggleStream(t bool) {
	c.logger.Debug(fmt.Sprintf("Setting stream to %t", t))
	c.stream = t
}

// SetProxy sets the proxy server to use for requests.
func (c *Client) SetProxy(proxy *url.URL) {
	c.proxy = proxy
}

// GetAPIKey returns the API key used for authentication.
func (c *Client) GetAPIKey() string {
	return c.auth.apiKey
}

// GetAccessToken returns the access token used for conversations.
func (c *Client) GetAccessToken() string {
	return c.auth.accessToken
}

// GetEngine returns the name of the GPT model being used.
func (c *Client) GetEngine() string {
	return c.engine
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

// SetConversation sets a specific conversation by ID.
func (c *Client) SetConversation(id string, conv Conversation) {
	c.conversations[id] = conv
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
	c.logger.Info("All conversations have been reset.")
}

// PingProxy checks if the proxy server is reachable.
func (c *Client) pingProxy() error {
	if c.proxy == nil {
		return fmt.Errorf("no proxy server set")
	}
	_, err := c.httpx.Get(c.proxy.String())
	return err
}

// CheckCredentials checks that the client has been initialized with credentials.
// If not, it returns an error.
//
//	Multiple credentials can be provided, but they are used in the following order:
//	 1. API key
//	 2. Email and password
//	 3. Access token
func (c *Client) checkCredentials() error {
	if c.auth.apiKey == "" && (c.auth.email == "" || c.auth.password == "") && c.auth.accessToken == "" {
		return fmt.Errorf("no credentials provided, please set an API key, email and password, or access token")
	}
	if (c.auth.email != "" && c.auth.password == "") || (c.auth.email == "" && c.auth.password != "") {
		return fmt.Errorf("email and password must be set together")
	}
	return nil
}

// Start initializes the client by checking credentials and authenticating with the OpenAI API.
func (c *Client) Start() error {
	// Check that the client has been initialized with credentials.
	c.auth.loadCachedAccessToken()
	if err := c.checkCredentials(); err != nil {
		return err
	}

	if c.proxy != nil {
		// check if proxy is alive, ping it
		// if not, return error
		if err := c.pingProxy(); err != nil {
			return err
		}
		c.logger.Debug("Proxy server is alive")
	}
	if c.auth.apiKey != "" {
		c.authmode = ApiKeyMode
		c.logger.Info("Starting client with API key Authentication")
	} else if c.auth.accessToken != "" {
		c.authmode = AccessTokenMode
		if c.auth.enableCache {
			if err := c.auth.cacheAccessToken(); err != nil {
				return err
			}
		}
		c.logger.Info("Starting client with access token Authentication")
		if !c.ispaid {
			c.engine = "text-davinci-002-render-sha"
			c.logger.Debug("Using free engine: " + c.engine)
		}
	} else if c.auth.email != "" && c.auth.password != "" {
		// Authenticate with the OpenAI API and set the access token.
		c.logger.Info("Starting client with email and password Authentication")
		accessToken, err := c.auth.GetAccessToken()
		if err != nil {
			return err
		}
		c.logger.Info("Successfully authenticated with OpenAI")
		c.auth.accessToken = accessToken
		c.authmode = AccessTokenMode
		if !c.ispaid {
			c.engine = "text-davinci-002-render-sha"
			c.logger.Debug("Using free engine: " + c.engine)
		}
	}
	c.auth.clientStarted = true
	return nil
}

// Logger Module

// Logger is a simple logger that can be used to log messages to the console.
type Logger struct {
	// The minimum level of messages to log.
	Level LogLevel
	// sessionName is the name of the session.
	sessionName string
}

// SetLevel sets the minimum level of messages to log.
func (l *Logger) SetLevel(level LogLevel) {
	l.Level = level
}

// LogLevel is an enum for the different log levels.
type LogLevel int

const (
	// LogLevelNone disables all logging.
	LogLevelNone LogLevel = iota
	// LogLevelDebug logs debug messages.
	LogLevelDebug
	// LogLevelInfo logs informational messages.
	LogLevelInfo
	// LogLevelWarn logs warning messages.
	LogLevelWarn
	// LogLevelError logs error messages.
	LogLevelError
)

// SessionName returns the name of the session, or an empty string if it's the default session.
func (l *Logger) SessionName() string {
	if l.sessionName == "default" {
		return ""
	}
	return l.sessionName
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	if l.Level <= LogLevelDebug {
		log.Printf("chatgpt%s - Debug - %s", l.SessionName(), msg)
	}
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	if l.Level <= LogLevelInfo {
		log.Printf("chatgpt%s - Info - %s", l.SessionName(), msg)
	}
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	if l.Level <= LogLevelWarn {
		log.Printf("chatgpt%s - Warn - %s", l.SessionName(), msg)
	}
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	if l.Level <= LogLevelError {
		log.Printf("chatgpt%s - Error - %s", l.SessionName(), msg)
	}
}
