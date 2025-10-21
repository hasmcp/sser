package ssergo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors" // Added errors package
	"fmt"
	"io"
	"log" // Added log package
	"net/http"
	"os"
	"time"
)

// EventCallback defines the function signature for processing a single event line from the SSE stream.
type EventCallback func(line string)

// SSERClient defines the interface for interacting with the PubSub API.
type SSERClient interface {
	CreatePubSub() error
	DeletePubSub(id string) error
	PublishEvent(id string, message string) error
	SubscribeToTopic(id string, topicAccessToken string, callback EventCallback) error
}

// Params holds configuration parameters for the New constructor.
type Params struct {
	BaseURL        string
	APIAccessToken string
	Logger         *log.Logger
	HTTPClient     *http.Client
}

// sserClient holds the base configuration for API interaction. It is private to enforce
// that it must be accessed via the SSERClient interface.
type sserClient struct {
	baseURL    string
	apiToken   string
	logger     *log.Logger
	httpClient *http.Client
}

// New creates a new instance of SSERClient and returns it as the interface.
// It initializes default dependencies (Logger, HTTPClient) if they are not provided in the Params.
func New(p Params) (SSERClient, error) {
	if p.BaseURL == "" {
		return nil, errors.New("BaseURL cannot be empty")
	}
	if p.APIAccessToken == "" {
		return nil, errors.New("APIAccessToken cannot be empty")
	}

	// Default HTTP client with a timeout for standard operations
	if p.HTTPClient == nil {
		p.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}

	// Default Logger using the standard log library, writing to os.Stdout
	if p.Logger == nil {
		// Set default log output and prefix
		p.Logger = log.New(os.Stdout, "PUBSUB_SDK: ", log.LstdFlags)
	}

	return &sserClient{
		baseURL:    p.BaseURL,
		apiToken:   p.APIAccessToken,
		logger:     p.Logger,
		httpClient: p.HTTPClient,
	}, nil
}

// =============================================================================
// STRUCTS FOR JSON PAYLOADS AND RESPONSES
// =============================================================================

// PublishPayload matches the expected body for the publish endpoint.
type PublishPayload struct {
	Event EventPayload `json:"event"`
}

// EventPayload holds the message content.
type EventPayload struct {
	Message string `json:"message"`
}

// =============================================================================
// CORE API METHODS
// =============================================================================

// CreatePubSub sends a POST request to create a new PubSub topic.
func (c *sserClient) CreatePubSub() error {
	url := fmt.Sprintf("%s/api/v1/pubsubs", c.baseURL)
	c.logger.Println("Attempting to create a new PubSub topic...")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Printf("HTTP Status: %s\n", resp.Status)
	io.Copy(os.Stdout, resp.Body)
	c.logger.Println("\nCreation command finished. Check the response above for the new PubSub ID.")
	return nil
}

// DeletePubSub sends a DELETE request to remove a PubSub topic by ID.
func (c *sserClient) DeletePubSub(id string) error {
	url := fmt.Sprintf("%s/api/v1/pubsubs/%s", c.baseURL, id)
	c.logger.Printf("Attempting to delete PubSub topic ID: %s\n", id)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Printf("HTTP Status: %s\n", resp.Status)
	// Output body (which might be empty or contain confirmation)
	io.Copy(os.Stdout, resp.Body)
	c.logger.Println("\nDeletion command finished.")
	return nil
}

// PublishEvent sends a POST request to publish a message to a topic.
func (c *sserClient) PublishEvent(id string, message string) error {
	url := fmt.Sprintf("%s/api/v1/pubsubs/%s/events", c.baseURL, id)
	c.logger.Printf("Attempting to publish message to ID: %s\n", id)

	payload := PublishPayload{
		Event: EventPayload{Message: message},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	c.logger.Printf("Payload: %s\n", string(body))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Printf("HTTP Status: %s\n", resp.Status)
	io.Copy(os.Stdout, resp.Body)
	c.logger.Println("\nPublish command finished.")
	return nil
}

// SubscribeToTopic establishes an SSE connection and streams events, calling the provided callback function for each line received.
func (c *sserClient) SubscribeToTopic(id string, topicAccessToken string, callback EventCallback) error {
	url := fmt.Sprintf("%s/api/v1/pubsubs/%s/events", c.baseURL, id)
	c.logger.Println("--------------------------------------------------------")
	c.logger.Printf("Subscribing to %s. Listening for Server-Sent Events (SSE). Press Ctrl+C to stop.\n", id)
	c.logger.Println("--------------------------------------------------------")

	// We temporarily use a client with no timeout for the long-lived SSE connection.
	// We make a copy of the default client to modify the timeout.
	streamingClient := *c.httpClient
	streamingClient.Timeout = 0

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// NOTE: This uses the specific SSER_TOPIC_ACCESS_TOKEN
	req.Header.Set("Authorization", "Bearer "+topicAccessToken)

	resp, err := streamingClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Printf("Subscription failed. HTTP Status: %s\n", resp.Status)
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: %s", string(body))
	}

	// Use a scanner to read the streaming response line by line (SSE format)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		callback(scanner.Text()) // Call the provided callback function
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading stream: %w", err)
	}

	c.logger.Println("\nSubscription closed by server.")
	return nil
}
