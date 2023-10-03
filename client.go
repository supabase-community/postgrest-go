package postgrest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
)

var (
	version = "v0.0.7"
)

type Client struct {
	ClientError     error
	session         http.Client
	clientTransport transport
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
	}

	c := Client{
		session:         http.Client{Transport: t},
		clientTransport: t,
	}

	if schema == "" {
		schema = "public"
	}

	// Set required headers
	c.clientTransport.header.Set("Accept", "application/json")
	c.clientTransport.header.Set("Content-Type", "application/json")
	c.clientTransport.header.Set("Accept-Profile", schema)
	c.clientTransport.header.Set("Content-Profile", schema)
	c.clientTransport.header.Set("X-Client-Info", "postgrest-go/"+version)

	// Set optional headers if they exist
	for key, value := range headers {
		c.clientTransport.header.Set(key, value)
	}

	return &c
}

// TokenAuth sets authorization headers for subsequent requests.
func (c *Client) TokenAuth(token string) *Client {
	c.clientTransport.header.Set("Authorization", "Bearer "+token)
	c.clientTransport.header.Set("apikey", token)
	return c
}

// ChangeSchema modifies the schema for subsequent requests.
func (c *Client) ChangeSchema(schema string) *Client {
	c.clientTransport.header.Set("Accept-Profile", schema)
	c.clientTransport.header.Set("Content-Profile", schema)
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
	url := path.Join(c.clientTransport.baseURL.Path, "rpc", name)
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
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	for headerName, values := range t.header {
		for _, val := range values {
			req.Header.Add(headerName, val)
		}
	}
	req.URL = t.baseURL.ResolveReference(req.URL)
	return http.DefaultTransport.RoundTrip(req)
}
