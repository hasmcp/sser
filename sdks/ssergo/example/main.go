package main

import (
	"fmt"
	"os"

	ssergocli "github.com/mustafaturan/sser/sdks/ssergo"
)

func main() {
	// 1. Get configuration from environment variables
	baseURL := os.Getenv("SSER_API_BASE_URL")
	apiToken := os.Getenv("SSER_API_ACCESS_TOKEN")

	// 2. Setup client using the New constructor
	client, err := ssergocli.New(ssergocli.Params{
		BaseURL:        baseURL,
		APIAccessToken: apiToken,
		// Logger and HTTPClient are nil, so New will use defaults
	})
	if err != nil {
		// New function handles BaseURL/APIAccessToken emptiness check
		fmt.Fprintf(os.Stderr, "Configuration Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Setup CLI flags/subcommands
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "create":
		if err := client.CreatePubSub(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating pubsub: %v\n", err)
			os.Exit(1)
		}

	case "delete":
		if len(args) < 1 {
			fmt.Println("Error: Missing PubSub ID for delete command.")
			printUsage()
			os.Exit(1)
		}
		id := args[0]
		if err := client.DeletePubSub(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting pubsub: %v\n", err)
			os.Exit(1)
		}

	case "publish":
		if len(args) < 2 {
			fmt.Println("Error: Missing PubSub ID or message for publish command.")
			printUsage()
			os.Exit(1)
		}
		id := args[0]
		message := args[1]
		if err := client.PublishEvent(id, message); err != nil {
			fmt.Fprintf(os.Stderr, "Error publishing event: %v\n", err)
			os.Exit(1)
		}

	case "subscribe":
		if len(args) < 1 {
			fmt.Println("Error: Missing PubSub ID for subscribe command.")
			printUsage()
			os.Exit(1)
		}
		id := args[0]

		// For subscription, we need the specific SSER_TOPIC_ACCESS_TOKEN
		topicAccessToken := os.Getenv("SSER_TOPIC_ACCESS_TOKEN")
		if topicAccessToken == "" {
			fmt.Println("ERROR: The SSER_TOPIC_ACCESS_TOKEN environment variable must be set for subscription.")
			os.Exit(1)
		}

		// Define the callback function to handle each received line (keeping the original CLI behavior)
		printEventLine := func(line string) {
			fmt.Println(line)
		}

		if err := client.SubscribeToTopic(id, topicAccessToken, printEventLine); err != nil {
			fmt.Fprintf(os.Stderr, "Error subscribing to topic: %v\n", err)
			os.Exit(1)
		}

	case "help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// printUsage displays the usage instructions.
func printUsage() {
	fmt.Println("--------------------------------------------------------")
	fmt.Println("Usage: go run pubsub_client.go <command> [arguments]")
	fmt.Println("Configuration is read from environment variables:")
	fmt.Println("  SSER_API_BASE_URL (required)")
	fmt.Println("  SSER_API_ACCESS_TOKEN (required for client initialization)")
	fmt.Println("  SSER_TOPIC_ACCESS_TOKEN (required for subscribe command)")
	fmt.Println("Available commands:")
	fmt.Println("  create                - Create a new PubSub topic.")
	fmt.Println("  delete <id>           - Delete a PubSub topic by ID.")
	fmt.Println("  publish <id> <message>- Publish a message to a PubSub topic ID.")
	fmt.Println("  subscribe <id>        - Subscribe to events on a PubSub topic ID.")
	fmt.Println("  help                  - Show this help menu.")
	fmt.Println("--------------------------------------------------------")
}
