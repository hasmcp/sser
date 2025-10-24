#!/bin/bash

# =========================================================
# PubSub Command Line Client
# This script provides a simple interface to a hypothetical PubSub API
# using curl for various operations.
# =========================================================

# --- Global Variables ---
# Use environment variables if they are set, otherwise initialize to empty strings.
SSER_API_BASE_URL="${SSER_API_BASE_URL:-}"
SSER_API_ACCESS_TOKEN="${SSER_API_ACCESS_TOKEN:-}"

# --- Initialization Function ---
initialize() {
    echo "--- PubSub CLI Initialization ---"
    local url_source="Prompt"
    local token_source="Prompt"

    # 1. Base API URL Check and Prompt
    # If the global variable is already set (from environment), skip prompting.
    if [ -n "$SSER_API_BASE_URL" ]; then
        url_source="Environment"
    else
        while [ -z "$SSER_API_BASE_URL" ]; do
            read -r -p "Enter the Base API URL (e.g., http://localhost:8889): " input_url
            if [ -n "$input_url" ]; then
                SSER_API_BASE_URL="$input_url"
            else
                echo "Base URL cannot be empty. Please try again."
            fi
        done
    fi

    # 2. Main API Access Token Check and Prompt
    # If the global variable is already set (from environment), skip prompting.
    if [ -n "$SSER_API_ACCESS_TOKEN" ]; then
        token_source="Environment"
    else
        while [ -z "$SSER_API_ACCESS_TOKEN" ]; do
            read -r -s -p "Enter the SSER_API_ACCESS_TOKEN: " input_token
            echo "" # Newline after silent input
            if [ -n "$input_token" ]; then
                SSER_API_ACCESS_TOKEN="$input_token"
            else
                echo "API Token cannot be empty. Please try again."
            fi
        done
    fi

    echo "Initialization complete."
    echo "  Base URL (Source: $url_source): $SSER_API_BASE_URL"
    # Do not print the token value, only its source and status
    echo "  API Token (Source: $token_source): Set"
    echo "Use './sser-cli.sh help' for command list."
}

# --- Utility Functions ---

# Function to create a new PubSub topic
create_pubsub() {
    echo "Attempting to create a new PubSub topic."
    local payload='{}'
    local persist_input

    # Prompt user for persistence
    while true; do
        read -r -p "Do you want this topic to be persisted to storage? (y/N): " persist_input
        persist_input=$(echo "$persist_input" | tr '[:upper:]' '[:lower:]') # Convert to lowercase

        if [[ "$persist_input" == "y" || "$persist_input" == "yes" ]]; then
            # Set the payload for persistence: {"pubsub": {"persist": true}}
            payload='{"pubsub": {"persist": true}}'
            echo "Persistence enabled."
            break
        elif [[ "$persist_input" == "n" || "$persist_input" == "no" || -z "$persist_input" ]]; then
            # Default payload: {}
            echo "Persistence disabled (default)."
            break
        else
            echo "Invalid input. Please enter 'y' or 'n'."
        fi
    done

    echo "Payload: $payload"

    curl -s -w "\nHTTP Status: %{http_code}\n" \
        -H "Authorization: Bearer $SSER_API_ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -X POST \
        -d "$payload" \
        "$SSER_API_BASE_URL/api/v1/pubsubs"

    echo "--------------------------------------------------------"
    echo "Creation command finished. Check the response above for the new PubSub ID."
}

# Function to delete a PubSub topic by ID
delete_pubsub() {
    if [ -z "$1" ]; then
        read -r -p "Enter the PubSub ID to delete: " id
    else
        id="$1"
    fi

    if [ -z "$id" ]; then
        echo "Error: PubSub ID is required for deletion."
        return 1
    fi

    echo "Attempting to delete PubSub topic ID: $id"

    curl -s -w "\nHTTP Status: %{http_code}\n" \
        -H "Authorization: Bearer $SSER_API_ACCESS_TOKEN" \
        -X DELETE \
        "$SSER_API_BASE_URL/api/v1/pubsubs/$id"

    echo "--------------------------------------------------------"
    echo "Deletion command finished."
}

# Function to publish an event to a PubSub topic
publish_event() {
    if [ -z "$1" ]; then
        read -r -p "Enter the PubSub ID to publish to: " id
    else
        id="$1"
    fi

    if [ -z "$id" ]; then
        echo "Error: PubSub ID is required for publishing."
        return 1
    fi

    if [ -z "$2" ]; then
        read -r -p "Enter the message content (e.g., 'Hello World'): " message_content
    else
        # Allow passing multiple arguments as a message
        message_content="$2"
    fi

    if [ -z "$message_content" ]; then
        echo "Error: Message content cannot be empty."
        return 1
    fi

    # Wrap the user-provided content into the JSON payload
    payload="{\"event\": {\"message\": \"$message_content\"}}"

    echo "Attempting to publish message to ID: $id"
    echo "Payload: $payload"

    curl -s -w "\nHTTP Status: %{http_code}\n" \
        -H "Authorization: Bearer $SSER_API_ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -X POST \
        -d "$payload" \
        "$SSER_API_BASE_URL/api/v1/pubsubs/$id/events"

    echo "--------------------------------------------------------"
    echo "Publish command finished."
}

# Function to subscribe and listen to a PubSub topic
subscribe_topic() {
    if [ -z "$1" ]; then
        read -r -p "Enter the PubSub ID to subscribe to: " id
    else
        id="$1"
    fi

    if [ -z "$id" ]; then
        echo "Error: PubSub ID is required for subscription."
        return 1
    fi

    # The prompt specified a different token for subscription
    local topic_token
    read -r -s -p "Enter the SSER_TOPIC_ACCESS_TOKEN for subscription: " topic_token
    echo "" # Newline after silent input

    if [ -z "$topic_token" ]; then
        echo "Error: Topic Access Token cannot be empty."
        return 1
    fi

    echo "--------------------------------------------------------"
    echo "Subscribing to $id. Listening for Server-Sent Events (SSE). Press Ctrl+C to stop."
    echo "--------------------------------------------------------"

    # Use curl's -i option to show headers as requested by the prompt,
    # and keep the connection open for SSE.
    curl -i \
        -H "Authorization: Bearer $topic_token" \
        -X GET \
        "$SSER_API_BASE_URL/api/v1/pubsubs/$id/events"
}

# --- Help Menu ---
show_help() {
    echo "--------------------------------------------------------"
    echo "Usage: ./sser-cli.sh <command> [arguments]"
    echo "Available commands:"
    echo "  help                  - Show this help menu."
    echo "  init                  - Manually re-initialize API URL and token."
    echo "  create                - Create a new PubSub topic (prompts for persistence)."
    echo "  delete <id>           - Delete a PubSub topic by ID."
    echo "  publish <id> <message>- Publish a message to a PubSub topic ID."
    echo "  subscribe <id>        - Subscribe to events on a PubSub topic ID (requires SSER_TOPIC_ACCESS_TOKEN)."
    echo "--------------------------------------------------------"
}

# --- Main Execution ---

# Run initialization upon script execution
initialize

# Check if a command was passed as an argument
if [ -z "$1" ]; then
    show_help
    exit 0
fi

# Handle the command
case "$1" in
    "init")
        initialize
        ;;
    "create")
        create_pubsub
        ;;
    "delete")
        delete_pubsub "$2"
        ;;
    "publish")
        publish_event "$2" "$3"
        ;;
    "subscribe")
        subscribe_topic "$2"
        ;;
    "help")
        show_help
        ;;
    *)
        echo "Error: Unknown command '$1'"
        show_help
        exit 1
        ;;
esac

exit 0