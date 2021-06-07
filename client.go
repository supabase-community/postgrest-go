package postgrest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func NewClient(rawURL, schema string, headers map[string]string) *Client {
	// Create URL from rawURL
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return &Client{ClientError: err}
	}

	t := transport{
		params:  url.Values{},
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

	// Set optional headers if exist
	for key, value := range headers {
		c.clientTransport.header.Set(key, value)
	}

	return &c
}

type Client struct {
	ClientError     error
	session         http.Client
	clientTransport transport
}

func (c *Client) TokenAuth(token string) *Client {
	c.clientTransport.header.Set("Authorization", "Basic "+token)
	return c
}

func (c *Client) ChangeSchema(schema string) *Client {
	c.clientTransport.header.Set("Accept-Profile", schema)
	c.clientTransport.header.Set("Content-Profile", schema)
	return c
}

func (c *Client) From(table string) *QueryBuilder {
	c.clientTransport.baseURL.Path += table
	return &QueryBuilder{client: c}
}

func (c *Client) Rpc(name string, count string, rpcBody interface{}) string {

	// Get body if exist
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
	req, err := http.NewRequest("POST", "/rpc/"+name, readerBody)
	if err != nil {
		c.ClientError = err
		return ""
	}

	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		if c.clientTransport.header.Get("Prefer") == "" {
			c.clientTransport.header.Set("Prefer", "count="+count)
		} else {
			c.clientTransport.header.Set("Prefer", c.clientTransport.header.Get("Prefer")+",count="+count)
		}
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
	params  url.Values
	header  http.Header
	baseURL url.URL
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header = t.header
	req.URL = t.baseURL.ResolveReference(req.URL)
	req.URL.RawQuery = t.params.Encode()
	return http.DefaultTransport.RoundTrip(req)
}
