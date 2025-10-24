/**
 * Custom error class for API-related issues.
 */
class SSERError extends Error {
  constructor(message, status = null, body = null) {
    super(message);
    this.name = 'SSERError';
    this.status = status;
    this.body = body;
  }
}

/**
 * Manages configuration for the SSERClient.createPubSub method.
 * This implements the Functional Options Pattern adapted for JavaScript classes.
 */
class CreatePubSubOptions {
  /**
   * @private
   */
  constructor() {
    this.persist = false;
  }

  /**
   * Factory method to create an options object for default creation.
   * @returns {CreatePubSubOptions}
   */
  static defaults() {
    return new CreatePubSubOptions();
  }

  /**
   * Sets the persistence option for the new topic.
   * @param {boolean} value - If true, the topic will be persisted to storage.
   * @returns {CreatePubSubOptions}
   */
  static WithPersist(value) {
    const opts = new CreatePubSubOptions();
    opts.persist = !!value;
    return opts;
  }
}


/**
 * SSERClient provides methods to interact with the PubSub API.
 */
class SSERClient {
  /**
   * Creates a new instance of SSERClient.
   * @param {object} params - Configuration parameters.
   * @param {string} params.baseURL - The API base URL (e.g., "http://localhost:8889"). (Required)
   * @param {string} params.apiAccessToken - The main API bearer token. (Required)
   * @param {function} [params.fetchClient=fetch] - The fetch function to use (e.g., global fetch or a custom implementation).
   * @param {object} [params.logger=console] - A logger object with a .log() method (e.g., console).
   * @throws {SSERError} If required parameters are missing.
   */
  constructor(params) {
    if (!params || !params.baseURL) {
      throw new SSERError("baseURL cannot be empty.");
    }
    if (!params.apiAccessToken) {
      throw new SSERError("apiAccessToken cannot be empty.");
    }

    this.baseURL = params.baseURL.replace(/\/$/, ''); // Remove trailing slash
    this.apiAccessToken = params.apiAccessToken;
    this.fetchClient = params.fetchClient || (typeof fetch !== 'undefined' ? fetch : this._throwNoFetch);
    this.logger = params.logger || console;

    this.logger.log("SSERClient initialized.");
  }

  // Fallback if fetch is not defined in the environment
  _throwNoFetch() {
    throw new SSERError("Fetch API is not available. Please provide a custom fetchClient implementation in params.");
  }

  _apiURL(path) {
    return `${this.baseURL}/api/v1/${path}`;
  }

  /**
   * Executes a standard API request and handles non-200 responses.
   * @param {string} path - The path fragment (e.g., 'pubsubs' or 'pubsubs/123').
   * @param {object} options - Fetch options.
   * @returns {Promise<object|string>} The JSON response body or plain text.
   * @throws {SSERError} On API error.
   */
  async _executeRequest(path, options = {}) {
    const url = this._apiURL(path);
    const method = options.method || 'GET';

    // Default headers
    options.headers = {
      'Authorization': `Bearer ${this.apiAccessToken}`,
      ...options.headers,
    };

    this.logger.log(`[${method}] Executing request to ${url}`);

    const response = await this.fetchClient(url, options);

    if (!response.ok) {
      let bodyText = await response.text();

      // Try to parse JSON for better error messaging
      let bodyJson = bodyText;
      try {
        bodyJson = JSON.parse(bodyText);
      } catch (e) {
        // If parsing fails, use raw text
      }

      throw new SSERError(
        `API request failed with status ${response.status}: ${response.statusText}`,
        response.status,
        bodyJson
      );
    }

    // Return raw text if content-type is not JSON, otherwise parse JSON
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return response.json();
    }
    return response.text();
  }

  /**
   * Creates a new PubSub topic using optional configuration objects.
   * * @param {...CreatePubSubOptions} opts - Optional configuration objects (e.g., CreatePubSubOptions.WithPersist(true)).
   * @returns {Promise<object|string>} The API response body (usually JSON with the new ID).
   * * Example:
   * client.createPubSub(CreatePubSubOptions.WithPersist(true));
   */
  async createPubSub(...opts) {
    // Merge options into a single configuration object
    const config = CreatePubSubOptions.defaults();

    // In a full implementation, you'd iterate over opts and merge them.
    // Since we only have one option (persist) right now, we'll just check the first one.
    if (opts.length > 0 && opts[0].persist !== undefined) {
      config.persist = opts[0].persist;
    }

    this.logger.log(`Attempting to create a new PubSub topic (Persist: ${config.persist})...`);

    let requestBody = {};

    if (config.persist) {
      // Structure the body as required: {"pubsub": {"persist": true}}
      requestBody = {
        pubsub: {
          persist: true
        }
      };
    }

    this.logger.log('Request body:', JSON.stringify(requestBody));

    return this._executeRequest('pubsubs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(requestBody)
    });
  }

  /**
   * Deletes a PubSub topic by ID.
   * @param {string} id - The unique ID of the topic to delete.
   * @returns {Promise<object|string>} The API response body.
   */
  async deletePubSub(id) {
    this.logger.log(`Attempting to delete PubSub topic ID: ${id}`);
    return this._executeRequest(`pubsubs/${id}`, {
      method: 'DELETE'
    });
  }

  /**
   * Publishes an event (message) to a specified PubSub topic.
   * @param {string} id - The unique ID of the topic.
   * @param {*} message - The content to publish (will be wrapped in {"event": {"message": ...}}).
   * @returns {Promise<object|string>} The API response body.
   */
  async publishEvent(id, message) {
    this.logger.log(`Attempting to publish event to ID: ${id}`);
    const payload = {
      event: { message: message }
    };

    return this._executeRequest(`pubsubs/${id}/events`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
  }

  /**
   * Establishes an SSE connection and streams events, calling the provided callback for each line.
   * * IMPORTANT: This method requires the SSER_TOPIC_ACCESS_TOKEN and uses the streaming capabilities of Fetch.
   *
   * @param {string} id - The unique ID of the topic to subscribe to.
   * @param {string} topicAccessToken - The specific token for read access to the topic.
   * @param {function(string): void} callback - Function called with each raw line of the SSE stream.
   * @returns {Promise<void>} Resolves when the connection is closed.
   * @throws {SSERError} On initial API connection error or streaming error.
   */
  async subscribeToTopic(id, topicAccessToken, callback) {
    const url = this._apiURL(`pubsubs/${id}/events`);
    this.logger.log(`Subscribing to ${id}. Listening for SSE...`);

    // Options for the long-lived SSE request
    const options = {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${topicAccessToken}`, // NOTE: Different token
        'Accept': 'text/event-stream'
      }
    };

    const response = await this.fetchClient(url, options);

    if (!response.ok) {
      let bodyText = await response.text();
      throw new SSERError(
        `Subscription failed. HTTP Status ${response.status}: ${response.statusText}`,
        response.status,
        bodyText
      );
    }

    // Use the reader API to process the stream chunk by chunk
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          this.logger.log("Subscription stream finished.");
          break;
        }

        // Convert chunk to text and add to buffer
        buffer += decoder.decode(value, { stream: true });

        // Process lines in the buffer
        const lines = buffer.split('\n');
        buffer = lines.pop(); // Keep the last, incomplete line in the buffer

        for (const line of lines) {
          if (line.trim() !== '') {
            callback(line);
          }
        }
      }
    } catch (error) {
      this.logger.log("Error during subscription stream:", error);
      throw new SSERError(`Stream processing error: ${error.message}`);
    } finally {
      // Ensure the reader is released
      reader.releaseLock();
    }
  }
}