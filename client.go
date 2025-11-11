package postgrest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
)

var (
	version = "v0.1.1"
)

// Client represents a PostgREST client
// Similar to PostgrestClient in postgrest-js
type Client struct {
	ClientError error
	session     *http.Client
	Transport   *transport
	schemaName  string
}

// NewClientWithError constructs a new client given a URL to a Postgrest instance.
func NewClientWithError(rawURL, schema string, headers map[string]string) (*Client, error) {
	// Create URL from rawURL
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	t := transport{
		header:  http.Header{},
		baseURL: *baseURL,
		Parent:  nil,
	}

	c := Client{
		session:    &http.Client{Transport: &t},
		Transport:  &t,
		schemaName: schema,
	}

	if schema == "" {
		schema = "public"
		c.schemaName = schema
	}

	// Set required headers
	c.Transport.SetHeaders(map[string]string{
		"Accept":          "application/json",
		"Content-Type":    "application/json",
		"Accept-Profile":  schema,
		"Content-Profile": schema,
		"X-Client-Info":   "postgrest-go/" + version,
	})
	// Set optional headers if they exist
	c.Transport.SetHeaders(headers)

	return &c, nil
}

// NewClient constructs a new client given a URL to a Postgrest instance.
func NewClient(rawURL, schema string, headers map[string]string) *Client {
	client, err := NewClientWithError(rawURL, schema, headers)
	if err != nil {
		return &Client{ClientError: err}
	}
	return client
}

func (c *Client) PingWithError() error {
	req, err := http.NewRequest("GET", path.Join(c.Transport.baseURL.Path, ""), nil)
	if err != nil {
		return err
	}

	resp, err := c.session.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("ping failed")
	}

	return nil
}

func (c *Client) Ping() bool {
	err := c.PingWithError()
	if err != nil {
		c.ClientError = err
		return false
	}
	return true
}

// SetApiKey sets api key header for subsequent requests.
func (c *Client) SetApiKey(apiKey string) *Client {
	c.Transport.SetHeader("apikey", apiKey)
	return c
}

// SetAuthToken sets authorization header for subsequent requests.
func (c *Client) SetAuthToken(authToken string) *Client {
	c.Transport.SetHeader("Authorization", "Bearer "+authToken)
	return c
}

// ChangeSchema modifies the schema for subsequent requests.
func (c *Client) ChangeSchema(schema string) *Client {
	c.schemaName = schema
	c.Transport.SetHeaders(map[string]string{
		"Accept-Profile":  schema,
		"Content-Profile": schema,
	})
	return c
}

// Schema selects a schema to query or perform an function (rpc) call
func (c *Client) Schema(schema string) *Client {
	newClient := &Client{
		session:    c.session,
		Transport:  c.Transport,
		schemaName: schema,
	}

	// Update schema headers
	newClient.Transport.SetHeaders(map[string]string{
		"Accept-Profile":  schema,
		"Content-Profile": schema,
	})

	return newClient
}

// From sets the table or view to query from
func (c *Client) From(relation string) *QueryBuilder[map[string]interface{}] {
	return NewQueryBuilder[map[string]interface{}](c, relation)
}

// RpcOptions contains options for RPC
type RpcOptions struct {
	Head  bool
	Get   bool
	Count string // "exact", "planned", or "estimated"
}

// Rpc performs a function call
func (c *Client) Rpc(fn string, args interface{}, opts *RpcOptions) *FilterBuilder[interface{}] {
	if opts == nil {
		opts = &RpcOptions{}
	}

	var method string
	var body interface{}

	rpcURL := c.Transport.baseURL.JoinPath("rpc", fn)

	headers := make(http.Header)
	if c.Transport != nil {
		c.Transport.mu.RLock()
		for key, values := range c.Transport.header {
			for _, val := range values {
				headers.Add(key, val)
			}
		}
		c.Transport.mu.RUnlock()
	}

	if opts.Head || opts.Get {
		if opts.Head {
			method = "HEAD"
		} else {
			method = "GET"
		}
		// Add args as query parameters
		if argsMap, ok := args.(map[string]interface{}); ok {
			query := rpcURL.Query()
			for name, value := range argsMap {
				if value != nil {
					// Handle array values
					if arr, ok := value.([]interface{}); ok {
						var strValues []string
						for _, v := range arr {
							strValues = append(strValues, fmt.Sprintf("%v", v))
						}
						query.Set(name, fmt.Sprintf("{%s}", strings.Join(strValues, ",")))
					} else {
						query.Set(name, fmt.Sprintf("%v", value))
					}
				}
			}
			rpcURL.RawQuery = query.Encode()
		}
	} else {
		method = "POST"
		body = args
	}

	if opts.Count != "" && (opts.Count == "exact" || opts.Count == "planned" || opts.Count == "estimated") {
		headers.Add("Prefer", fmt.Sprintf("count=%s", opts.Count))
	}

	builder := NewBuilder[interface{}](c, method, rpcURL, &BuilderOptions{
		Headers: headers,
		Schema:  c.schemaName,
		Body:    body,
	})

	return &FilterBuilder[interface{}]{Builder: builder}
}

// RpcWithError executes a Postgres function (a.k.a., Remote Procedure Call), given the
// function name and, optionally, a body, returning the result as a string.
func (c *Client) RpcWithError(name string, count string, rpcBody interface{}) (string, error) {
	opts := &RpcOptions{Count: count}
	filterBuilder := c.Rpc(name, rpcBody, opts)
	response, err := filterBuilder.Execute(context.Background())
	if err != nil {
		return "", err
	}
	if response.Error != nil {
		return "", response.Error
	}
	// Convert response.Data to string
	dataBytes, _ := json.Marshal(response.Data)
	return string(dataBytes), nil
}

type transport struct {
	baseURL url.URL
	Parent  http.RoundTripper

	mu     sync.RWMutex
	header http.Header
}

func (t *transport) SetHeader(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.header.Set(key, value)
}

func (t *transport) SetHeaders(headers map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, value := range headers {
		t.header.Set(key, value)
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.RLock()
	for headerName, values := range t.header {
		for _, val := range values {
			req.Header.Add(headerName, val)
		}
	}
	t.mu.RUnlock()

	req.URL = t.baseURL.ResolveReference(req.URL)

	// This is only needed with usage of httpmock in testing. It would be better to initialize
	// t.Parent with http.DefaultTransport and then use t.Parent.RoundTrip(req)
	if t.Parent != nil {
		return t.Parent.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
