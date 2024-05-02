package postgrest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
)

var (
	version = "v0.1.1"
)

type Client struct {
	ClientError error
	session     http.Client
	Transport   *transport
}

// NewClient constructs a new client given a URL to a Postgrest instance.
func NewClient(rawURL, schema string, headers map[string]string) *Client {
	// Create URL from rawURL
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return &Client{ClientError: err}
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
	c.Transport.header.Set("Accept", "application/json")
	c.Transport.header.Set("Content-Type", "application/json")
	c.Transport.header.Set("Accept-Profile", schema)
	c.Transport.header.Set("Content-Profile", schema)
	c.Transport.header.Set("X-Client-Info", "postgrest-go/"+version)

	// Set optional headers if they exist
	for key, value := range headers {
		c.Transport.header.Set(key, value)
	}

	return &c
}

func (c *Client) Ping() bool {
	req, err := http.NewRequest("GET", path.Join(c.Transport.baseURL.Path, ""), nil)
	if err != nil {
		c.ClientError = err

		return false
	}

	resp, err := c.session.Do(req)
	if err != nil {
		c.ClientError = err

		return false
	}

	if resp.Status != "200 OK" {
		c.ClientError = errors.New("ping failed")

		return false
	}

	return true
}

// SetApiKey sets api key header for subsequent requests.
func (c *Client) SetApiKey(apiKey string) *Client {
	c.Transport.header.Set("apikey", apiKey)
	return c
}

// SetAuthToken sets authorization header for subsequent requests.
func (c *Client) SetAuthToken(authToken string) *Client {
	c.Transport.header.Set("Authorization", "Bearer "+authToken)
	return c
}

// ChangeSchema modifies the schema for subsequent requests.
func (c *Client) ChangeSchema(schema string) *Client {
	c.Transport.header.Set("Accept-Profile", schema)
	c.Transport.header.Set("Content-Profile", schema)
	return c
}

// From sets the table to query from.
func (c *Client) From(table string) *QueryBuilder {
	return &QueryBuilder{client: c, tableName: table, headers: map[string]string{}, params: map[string]string{}}
}

// Rpc executes a Postgres function (a.k.a., Remote Prodedure Call), given the
// function name and, optionally, a body, returning the result as a string.
func (c *Client) Rpc(name string, count string, rpcBody interface{}) string {
	// Get body if it exists
	var byteBody []byte = nil
	if rpcBody != nil {
		jsonBody, err := json.Marshal(rpcBody)
		if err != nil {
			c.ClientError = err
			return ""
		}
		byteBody = jsonBody
	}

	readerBody := bytes.NewBuffer(byteBody)
	url := path.Join(c.Transport.baseURL.Path, "rpc", name)
	req, err := http.NewRequest("POST", url, readerBody)
	if err != nil {
		c.ClientError = err
		return ""
	}

	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		req.Header.Add("Prefer", "count="+count)
	}

	resp, err := c.session.Do(req)
	if err != nil {
		c.ClientError = err
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.ClientError = err
		return ""
	}

	result := string(body)

	err = resp.Body.Close()
	if err != nil {
		c.ClientError = err
		return ""
	}

	return result
}

type transport struct {
	header  http.Header
	baseURL url.URL
	Parent  http.RoundTripper
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for headerName, values := range t.header {
		for _, val := range values {
			req.Header.Add(headerName, val)
		}
	}

	req.URL = t.baseURL.ResolveReference(req.URL)

	// This is only needed with usage of httpmock in testing. It would be better to initialize
	// t.Parent with http.DefaultTransport and then use t.Parent.RoundTrip(req)
	if t.Parent != nil {
		return t.Parent.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
