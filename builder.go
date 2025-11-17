package postgrest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// Builder is the base builder for PostgREST queries
// Similar to PostgrestBuilder in postgrest-js
type Builder[T any] struct {
	method             string
	url                *url.URL
	headers            http.Header
	schema             string
	body               interface{}
	shouldThrowOnError bool
	signal             context.Context
	client             *Client
	isMaybeSingle      bool
}

// NewBuilder creates a new Builder instance
func NewBuilder[T any](client *Client, method string, url *url.URL, opts *BuilderOptions) *Builder[T] {
	if opts == nil {
		opts = &BuilderOptions{}
	}

	b := &Builder[T]{
		method:             method,
		url:                url,
		headers:            make(http.Header),
		schema:             opts.Schema,
		body:               opts.Body,
		shouldThrowOnError: opts.ShouldThrowOnError,
		signal:             opts.Signal,
		client:             client,
		isMaybeSingle:      opts.IsMaybeSingle,
	}

	// Copy headers from client
	if client != nil && client.Transport != nil {
		client.Transport.mu.RLock()
		for key, values := range client.Transport.header {
			for _, val := range values {
				b.headers.Add(key, val)
			}
		}
		client.Transport.mu.RUnlock()
	}

	// Copy additional headers
	if opts.Headers != nil {
		for key, values := range opts.Headers {
			for _, val := range values {
				b.headers.Add(key, val)
			}
		}
	}

	return b
}

// BuilderOptions contains options for creating a Builder
type BuilderOptions struct {
	Headers            http.Header
	Schema             string
	Body               interface{}
	ShouldThrowOnError bool
	Signal             context.Context
	IsMaybeSingle      bool
}

// ThrowOnError sets the builder to throw errors instead of returning them
func (b *Builder[T]) ThrowOnError() *Builder[T] {
	b.shouldThrowOnError = true
	return b
}

// SetHeader sets an HTTP header for the request
func (b *Builder[T]) SetHeader(name, value string) *Builder[T] {
	b.headers.Set(name, value)
	return b
}

// Execute executes the query and returns the response
func (b *Builder[T]) Execute(ctx context.Context) (*PostgrestResponse[T], error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if b.signal != nil {
		ctx = b.signal
	}

	// Set schema headers
	if b.schema != "" {
		if b.method == "GET" || b.method == "HEAD" {
			b.headers.Set("Accept-Profile", b.schema)
		} else {
			b.headers.Set("Content-Profile", b.schema)
		}
	}

	// Set Content-Type for non-GET/HEAD requests
	if b.method != "GET" && b.method != "HEAD" {
		b.headers.Set("Content-Type", "application/json")
	}

	// Prepare request body
	var bodyReader io.Reader
	if b.body != nil {
		bodyBytes, err := json.Marshal(b.body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	// Check if context is already canceled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, b.method, b.url.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	for key, values := range b.headers {
		for _, val := range values {
			req.Header.Add(key, val)
		}
	}

	// Execute request
	resp, err := b.client.session.Do(req)
	if err != nil {
		// Check if error is due to context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if b.shouldThrowOnError {
			return nil, err
		}
		return &PostgrestResponse[T]{
			Error: NewPostgrestError(
				fmt.Sprintf("FetchError: %s", err.Error()),
				fmt.Sprintf("%v", err),
				"",
				"",
			),
			Data:       *new(T),
			Count:      nil,
			Status:     0,
			StatusText: "",
		}, nil
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Parse response
	response := &PostgrestResponse[T]{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
	}

	// Handle errors
	if resp.StatusCode >= 400 {
		var errorData map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorData); err != nil {
			// Workaround for https://github.com/supabase/postgrest-js/issues/295
			if resp.StatusCode == 404 && len(bodyBytes) == 0 {
				response.Status = 204
				response.StatusText = "No Content"
				return response, nil
			}
			response.Error = NewPostgrestError(
				string(bodyBytes),
				"",
				"",
				"",
			)
			return response, nil
		}

		// Workaround for https://github.com/supabase/postgrest-js/issues/295
		if resp.StatusCode == 404 && len(bodyBytes) > 0 {
			var arr []interface{}
			if err := json.Unmarshal(bodyBytes, &arr); err == nil {
				response.Data = *new(T)
				response.Status = 200
				response.StatusText = "OK"
				return response, nil
			}
		}

		errorMsg := ""
		if msg, ok := errorData["message"].(string); ok {
			errorMsg = msg
		}

		errorDetails := ""
		if details, ok := errorData["details"].(string); ok {
			errorDetails = details
		}

		errorHint := ""
		if hint, ok := errorData["hint"].(string); ok {
			errorHint = hint
		}

		errorCode := ""
		if code, ok := errorData["code"].(string); ok {
			errorCode = code
		}

		response.Error = NewPostgrestError(errorMsg, errorDetails, errorHint, errorCode)

		// Handle maybeSingle case
		if b.isMaybeSingle && response.Error != nil && strings.Contains(errorDetails, "0 rows") {
			response.Error = nil
			response.Status = 200
			response.StatusText = "OK"
		}

		// When Single() is used and there's an error, return the error
		acceptHeader := b.headers.Get("Accept")
		if acceptHeader == "application/vnd.pgrst.object+json" && response.Error != nil {
			return nil, response.Error
		}

		if b.shouldThrowOnError && response.Error != nil {
			return nil, response.Error
		}

		return response, nil
	}

	// Parse successful response
	if b.method != "HEAD" {
		acceptHeader := b.headers.Get("Accept")
		if acceptHeader == "text/csv" || strings.Contains(acceptHeader, "application/vnd.pgrst.plan+text") {
			// For CSV and plan text, try to unmarshal as string
			strData := string(bodyBytes)
			// Use type assertion to check if T is string
			var zeroT T
			if _, ok := any(zeroT).(string); ok {
				response.Data = any(strData).(T)
			} else {
				// If T is not string, for plan text we can't unmarshal as JSON
				// Just leave Data as zero value - this is expected behavior for non-string types
				// The response body is available as plain text but can't be unmarshaled into non-string T
			}
		} else if len(bodyBytes) > 0 {
			acceptHeader := b.headers.Get("Accept")
			// Handle Single() case - application/vnd.pgrst.object+json returns a single object
			if acceptHeader == "application/vnd.pgrst.object+json" {
				// Single() returns a single object, but T might be []T (array type)
				// Use reflection to check if T is a slice type
				var zeroT T
				tType := reflect.TypeOf(zeroT)
				if tType != nil && tType.Kind() == reflect.Slice {
					// T is a slice type, unmarshal single object and wrap in array
					// Create an array JSON with the single object
					var arrJSON []byte
					arrJSON = append(arrJSON, '[')
					arrJSON = append(arrJSON, bodyBytes...)
					arrJSON = append(arrJSON, ']')
					// Unmarshal the array into response.Data
					if err := json.Unmarshal(arrJSON, &response.Data); err != nil {
						return nil, fmt.Errorf("error unmarshaling single object array: %w", err)
					}
				} else {
					// T is not a slice, unmarshal directly
					if err := json.Unmarshal(bodyBytes, &response.Data); err != nil {
						return nil, fmt.Errorf("error unmarshaling single object: %w", err)
					}
				}
			} else if b.isMaybeSingle {
				// Handle maybeSingle case
				var arr []interface{}
				if err := json.Unmarshal(bodyBytes, &arr); err == nil {
					if len(arr) > 1 {
						response.Error = NewPostgrestError(
							"JSON object requested, multiple (or no) rows returned",
							fmt.Sprintf("Results contain %d rows, application/vnd.pgrst.object+json requires 1 row", len(arr)),
							"",
							"PGRST116",
						)
						response.Status = 406
						response.StatusText = "Not Acceptable"
						return response, nil
					} else if len(arr) == 1 {
						// Unmarshal single item
						itemBytes, _ := json.Marshal(arr[0])
						json.Unmarshal(itemBytes, &response.Data)
					} else {
						// Empty array, return null equivalent
						response.Data = *new(T)
					}
				} else {
					// Not an array, it's a single object
					// Check if T is a slice type - if so, wrap the object in an array
					var zeroT T
					tType := reflect.TypeOf(zeroT)
					if tType != nil && tType.Kind() == reflect.Slice {
						// T is a slice type, wrap single object in array
						var arrJSON []byte
						arrJSON = append(arrJSON, '[')
						arrJSON = append(arrJSON, bodyBytes...)
						arrJSON = append(arrJSON, ']')
						if err := json.Unmarshal(arrJSON, &response.Data); err != nil {
							return nil, fmt.Errorf("error unmarshaling single object array: %w", err)
						}
					} else {
						// T is not a slice, unmarshal directly
						if err := json.Unmarshal(bodyBytes, &response.Data); err != nil {
							return nil, fmt.Errorf("error unmarshaling response: %w", err)
						}
					}
				}
			} else {
				json.Unmarshal(bodyBytes, &response.Data)
			}
		}
	}

	// Parse count from Content-Range header
	contentRange := resp.Header.Get("Content-Range")
	if contentRange != "" {
		parts := strings.Split(contentRange, "/")
		if len(parts) > 1 && parts[1] != "*" {
			if count, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				response.Count = &count
			}
		}
	}

	return response, nil
}

// ExecuteTo executes the query and unmarshals the result into the provided interface
func (b *Builder[T]) ExecuteTo(ctx context.Context, to interface{}) (*int64, error) {
	response, err := b.Execute(ctx)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, response.Error
	}

	// Marshal and unmarshal to convert to target type
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling response data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, to); err != nil {
		return nil, fmt.Errorf("error unmarshaling to target: %w", err)
	}

	return response.Count, nil
}
