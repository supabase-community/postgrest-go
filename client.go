package postgrest

import (
	"net/http"
	"net/url"
)

func NewClient(rawURL, schema string, headers map[string]string) Client {
	// Create URL from rawURL
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return Client{}
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
	if headers != nil {
		for key, value := range headers {
			c.clientTransport.header.Set(key, value)
		}
	}

	return c
}

type Client struct {
	session         http.Client
	clientTransport transport
}

func (c Client) TokenAuth(token string) Client {
	c.clientTransport.header.Set("Authorization", "Basic "+token)
	return c
}

func (c Client) ChangeSchema(schema string) Client {
	c.clientTransport.header.Set("Accept-Profile", schema)
	c.clientTransport.header.Set("Content-Profile", schema)
	return c
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
