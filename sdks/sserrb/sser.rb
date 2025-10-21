require 'net/http'
require 'uri'
require 'json'

# Custom exception for client-side errors
class SSERError < StandardError; end

# The SSERClient class provides methods to interact with the PubSub API.
class SSERClient
  # A simple struct to hold client configuration, including optional dependencies.
  Params = Struct.new(:base_url, :api_access_token, :http_client, :logger, keyword_init: true)

  # Internal implementation details
  private attr_reader :base_url, :api_access_token, :http_client, :logger

  # Initializes the SSERClient.
  #
  # @param params [Params] Configuration struct containing:
  #   - :base_url [String] The API base URL (required).
  #   - :api_access_token [String] The main API bearer token (required).
  #   - :http_client [Net::HTTP] An optional custom HTTP client.
  #   - :logger [Logger] An optional logger instance.
  # @return [SSERClient] An initialized client instance.
  # @raise [SSERError] if required parameters are missing.
  def initialize(params)
    raise SSERError, "Base URL cannot be empty" if params.base_url.to_s.empty?
    raise SSERError, "API Access Token cannot be empty" if params.api_access_token.to_s.empty?

    @base_url = params.base_url.chomp('/')
    @api_access_token = params.api_access_token

    # Use default Net::HTTP if no client is provided
    @http_client = params.http_client || Net::HTTP

    # Use Ruby's built-in Logger or a custom one
    @logger = params.logger || begin
      require 'logger'
      Logger.new($stdout, progname: 'PUBSUB_SDK')
    end
  end

  # Helper method to generate the full API URL
  private def api_url(path)
    URI.parse("#{base_url}/api/v1/#{path}")
  end

  # Helper method to execute a standard API request and handle the response.
  private def execute_request(request)
    request['Authorization'] = "Bearer #{api_access_token}"

    # We must explicitly start and finish the connection for standard requests
    uri = request.uri

    # Determine if HTTPS is needed and set port if necessary
    http = Net::HTTP.new(uri.host, uri.port)
    http.use_ssl = (uri.scheme == 'https')

    logger.info "Executing request: #{request.method} #{uri}"

    response = http.request(request)

    logger.info "Response Status: #{response.code} #{response.message}"

    if response.is_a?(Net::HTTPSuccess) || response.is_a?(Net::HTTPRedirection)
      response.body
    else
      raise SSERError, "API Error #{response.code}: #{response.message}\nBody: #{response.body}"
    end
  end

  # --- Public API Methods ---

  # Creates a new PubSub topic.
  # @return [String] The API response body (usually JSON with the new ID).
  def create_pubsub
    uri = api_url('pubsubs')
    request = Net::HTTP::Post.new(uri)
    request['Content-Type'] = 'application/json'
    request.body = '{}'
    execute_request(request)
  end

  # Deletes a PubSub topic by ID.
  # @param id [String] The unique ID of the topic to delete.
  # @return [String] The API response body.
  def delete_pubsub(id)
    uri = api_url("pubsubs/#{id}")
    request = Net::HTTP::Delete.new(uri)
    execute_request(request)
  end

  # Publishes an event (message) to a specified PubSub topic.
  # @param id [String] The unique ID of the topic.
  # @param message [String, Hash, Array] The content to publish.
  # @return [String] The API response body.
  def publish_event(id, message)
    uri = api_url("pubsubs/#{id}/events")
    request = Net::HTTP::Post.new(uri)
    request['Content-Type'] = 'application/json'

    # Construct the required JSON payload structure
    payload = { event: { message: message } }
    request.body = payload.to_json

    logger.info "Publishing payload: #{request.body}"
    execute_request(request)
  end

  # Establishes an SSE connection and calls a block for each received event.
  # NOTE: This method requires a different access token.
  #
  # @param id [String] The unique ID of the topic to subscribe to.
  # @param topic_access_token [String] The specific token for read access.
  # @yieldparam line [String] The raw line received from the SSE stream (e.g., 'data: ...').
  # @raise [SSERError] if the subscription fails.
  def subscribe_to_topic(id, topic_access_token)
    uri = api_url("pubsubs/#{id}/events")

    # We use Net::HTTP.start block for long-lived connection management
    Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == 'https') do |http|
      request = Net::HTTP::Get.new(uri)
      request['Authorization'] = "Bearer #{topic_access_token}"

      logger.info "Subscribing to #{id}. Press Ctrl+C to stop."

      http.request(request) do |response|
        unless response.code == '200'
          raise SSERError, "Subscription failed. HTTP Status: #{response.code} #{response.message}"
        end

        # Stream the body line by line
        response.read_body do |chunk|
          # Split chunk by newline to ensure we process full SSE lines
          chunk.each_line do |line|
            # Call the provided block (callback) with the raw line
            yield line.strip unless line.strip.empty?
          end
        end
      end
    end
    logger.info "Subscription closed."
  rescue Net::OpenTimeout, Net::ReadTimeout => e
    raise SSERError, "Connection timeout during subscription: #{e.message}"
  rescue Errno::ECONNREFUSED => e
    raise SSERError, "Connection refused: #{e.message}"
  end
end