package main

import "encoding/json"

const (
	TextDavinciEngine = "text-davinci-"
	Gpt3_5Engine      = "gpt-3.5-"
	Gpt4_32Engine     = "gpt-4-32"
	Gpt4_8Engine      = "gpt-4-8"
)

var Engines = map[string]int{
	TextDavinciEngine: 4000,
	Gpt3_5Engine:      4000,
	Gpt4_32Engine:     32000,
	Gpt4_8Engine:      8000,
}

// Message represents a struct with two fields: Role and Content.
type Message struct {
	Role    string `json:"role,omitempty"`    // Tag defies the JSON key name as "role" or omits the key if the value is empty.
	Content string `json:"content,omitempty"` // Tag defies the JSON key name as "content" or omits the key if the value is empty.
}

// Conversation represents a struct with three fields: InitMessage, LastMessage, and Messages.
type Conversation struct {
	InitMessage string    // First message sent in the conversation.
	LastMessage string    // Most recent message sent in the conversation.
	Messages    []Message // Slice of Message structs representing all messages sent in the conversation.
}

// Method to add a message to the Conversation struct.
func (c *Conversation) addMessage(m Message) {
	c.Messages = append(c.Messages, m) // Append the new message to the Messages slice within the Conversation.
	c.LastMessage = m.Content          // Update the LastMessage property of the Conversation with the content of the new message.
}

// Method to initialize the Conversation struct with an initial message.
func (c *Conversation) initMessage(m Message) {
	c.InitMessage = m.Content          // Set the InitMessage property of the Conversation to the content of the provided message.
	c.LastMessage = m.Content          // Set the LastMessage property of the Conversation to the content of the provided message.
	c.Messages = append(c.Messages, m) // Append the new message to the empty Messages slice within the Conversation.
}

// Method to retrieve the number of tokens (i.e. 4-byte substrings) in the InitMessage property of the Conversation struct.
func (c *Conversation) getTokenCount() int {
	return len(c.InitMessage) / 4 // Return the length of the InitMessage divided by 4 to get the number of 4-byte substrings.
}

func (c *Conversation) Marshal() string {
	// Marshal the Conversation struct to JSON.
	json, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	// Return the JSON string.
	return string(json)
}

func (c *Conversation) tokenizeMessage(engine string) {
	// Get the number of tokens in the InitMessage property of the Conversation struct.
	tokenCount := c.getTokenCount()

	// Get the maximum number of tokens allowed for the provided engine.
	maxTokens := getEngineTokenLimit(engine)

	// if the number of tokens in the message is greater than the maximum allowed, truncate the message to init_message and last_message.
	if tokenCount > maxTokens {
		c.Messages = []Message{
			{Role: "system", Content: c.InitMessage},
			{Role: "user", Content: c.LastMessage},
		}
	}
}
