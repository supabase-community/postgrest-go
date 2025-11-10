package postgrest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
)

var (
	version = "v0.1.1"
)

type Client struct {
	ClientError error
	session     http.Client
	Transport   *transport
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
		session:   http.Client{Transport: &t},
		Transport: &t,
	}

	if schema == "" {
		schema = "public"
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

	if resp.Status != "200 OK" {
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
	c.Transport.SetHeaders(map[string]string{
		"Accept-Profile":  schema,
		"Content-Profile": schema,
	})
	return c
}

// From sets the table to query from.
func (c *Client) From(table string) *QueryBuilder {
	return &QueryBuilder{client: c, tableName: table, headers: map[string]string{}, params: map[string]string{}}
}

// RpcWithError executes a Postgres function (a.k.a., Remote Prodedure Call), given the
// function name and, optionally, a body, returning the result as a string.
func (c *Client) RpcWithError(name string, count string, rpcBody interface{}) (string, error) {
	// Get body if it exists
	var byteBody []byte = nil
	if rpcBody != nil {
		jsonBody, err := json.Marshal(rpcBody)
		if err != nil {
			return "", err
		}
		byteBody = jsonBody
	}

	readerBody := bytes.NewBuffer(byteBody)
	url := path.Join(c.Transport.baseURL.Path, "rpc", name)
	req, err := http.NewRequest("POST", url, readerBody)
	if err != nil {
		return "", err
	}

	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		req.Header.Add("Prefer", "count="+count)
	}

	resp, err := c.session.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := string(body)

	err = resp.Body.Close()
	if err != nil {
		return "", err
	}

	return result, nil
}

// Rpc executes a Postgres function (a.k.a., Remote Prodedure Call), given the
// function name and, optionally, a body, returning the result as a string.
func (c *Client) Rpc(name string, count string, rpcBody interface{}) string {
	result, err := c.RpcWithError(name, count, rpcBody)
	if err != nil {
		c.ClientError = err
		return ""
	}
	return result
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
