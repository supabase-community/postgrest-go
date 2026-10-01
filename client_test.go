package postgrest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type pingBodySpy struct {
	closed bool
}

func (b *pingBodySpy) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (b *pingBodySpy) Close() error {
	b.closed = true
	return nil
}

type errorReadCloser struct {
	err error
}

func (r *errorReadCloser) Read(_ []byte) (int, error) {
	return 0, r.err
}

func (r *errorReadCloser) Close() error {
	return nil
}

func TestNewClientWithError_InvalidURL(t *testing.T) {
	client, err := NewClientWithError("http://[::1", "", nil)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestNewClient_InvalidURLSetsClientError(t *testing.T) {
	client := NewClient("http://[::1", "", nil)
	assert.NotNil(t, client)
	assert.Error(t, client.ClientError)
}

func TestClient_SetApiKey(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.SetApiKey("test-api-key")

	// Verify the header is set
	assert.Equal(t, "test-api-key", c.Transport.header.Get("apikey"))
}

func TestClient_SetAuthToken(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.SetAuthToken("test-token")

	// Verify the header is set
	assert.Equal(t, "Bearer test-token", c.Transport.header.Get("Authorization"))
}

func TestClient_ChangeSchema(t *testing.T) {
	c := NewClient("http://localhost:3000", "public", nil)
	c.ChangeSchema("private")

	assert.Equal(t, "private", c.schemaName)
	assert.Equal(t, "private", c.Transport.header.Get("Accept-Profile"))
	assert.Equal(t, "private", c.Transport.header.Get("Content-Profile"))
}

func TestClient_Schema(t *testing.T) {
	c := NewClient("http://localhost:3000", "public", nil)
	newClient := c.Schema("private")

	// Should return a new client with different schema
	assert.NotEqual(t, c, newClient)
	assert.Equal(t, "private", newClient.schemaName)
	assert.Equal(t, "private", newClient.Transport.header.Get("Accept-Profile"))
	assert.Equal(t, "private", newClient.Transport.header.Get("Content-Profile"))

	// Original client should still have original schema
	assert.Equal(t, "public", c.schemaName)
}

func TestClient_Ping(t *testing.T) {
	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://localhost:3000/", httpmock.NewStringResponder(200, "OK"))
	}

	c := NewClient("http://localhost:3000", "", nil)
	result := c.Ping()

	if mockResponses {
		assert.True(t, result)
	}
}

func TestClient_PingWithError(t *testing.T) {
	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://localhost:3000/", httpmock.NewStringResponder(500, "Error"))
	}

	c := NewClient("http://localhost:3000", "", nil)
	err := c.PingWithError()

	if mockResponses {
		assert.Error(t, err)
	}
}

func TestClient_PingWithError_RequestError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	transportErr := errors.New("network down")
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, transportErr
		}),
	}

	err := c.PingWithError()
	assert.ErrorIs(t, err, transportErr)
}

func TestClient_PingWithError_Non200ReturnsPingFailed(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	body := &pingBodySpy{}
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				Body:       body,
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := c.PingWithError()
	assert.EqualError(t, err, "ping failed")
	assert.True(t, body.closed)
}

func TestClient_PingWithError_SuccessClosesBody(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	body := &pingBodySpy{}
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       body,
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := c.PingWithError()
	assert.NoError(t, err)
	assert.True(t, body.closed)
}

func TestClient_Ping_ReturnsTrueOnSuccess(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("OK")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result := c.Ping()
	assert.True(t, result)
	assert.NoError(t, c.ClientError)
}

func TestClient_Rpc(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("POST", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, map[string]interface{}{
				"result": 42,
			})
			return resp, nil
		})
	}

	args := map[string]interface{}{"a": 1, "b": 2}
	response, err := c.Rpc("test_function", args, nil).Execute(context.Background())

	if mockResponses {
		assert.NoError(err)
		assert.NotNil(response)
	}
}

func TestClient_RpcWithError(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("POST", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, map[string]interface{}{
				"result": "success",
			})
			return resp, nil
		})
	}

	args := map[string]interface{}{"param": "value"}
	result, err := c.RpcWithError("test_function", "exact", args)

	if mockResponses {
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	}
}

func TestClient_RpcWithError_ReturnsExecuteError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       &errorReadCloser{err: errors.New("read failed")},
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := c.RpcWithError("test_function", "", map[string]interface{}{"param": "value"})

	assert.Equal(t, "", result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading response")
}

func TestClient_RpcWithError_ReturnsResponseError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Body: io.NopCloser(strings.NewReader(
					`{"message":"rpc failed","details":"invalid args","hint":"","code":"PGRST200"}`,
				)),
				Header: make(http.Header),
			}, nil
		}),
	}

	result, err := c.RpcWithError("test_function", "", map[string]interface{}{"param": "value"})

	assert.Equal(t, "", result)
	assert.Error(t, err)
	assert.Equal(t, "rpc failed", err.Error())
}

func TestTransport_RoundTrip_UsesParentWhenPresent(t *testing.T) {
	baseURL, _ := url.Parse("http://example.com/api/")
	tr := &transport{
		baseURL: *baseURL,
		header:  make(http.Header),
	}
	tr.SetHeader("X-Test", "1")

	parentCalled := false
	tr.Parent = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		parentCalled = true
		assert.Equal(t, "1", req.Header.Get("X-Test"))
		assert.Equal(t, "http://example.com/users", req.URL.String())
		return &http.Response{
			StatusCode: 204,
			Status:     "204 No Content",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	req, err := http.NewRequest("GET", "/users", nil)
	assert.NoError(t, err)

	resp, err := tr.RoundTrip(req)
	assert.NoError(t, err)
	assert.True(t, parentCalled)
	assert.NotNil(t, resp)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestClient_Rpc_GetWithQueryParams(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "GET", req.Method)
			assert.Equal(t, "1", req.URL.Query().Get("a"))
			assert.Equal(t, "two", req.URL.Query().Get("b"))
			assert.Equal(t, "{1,two,true}", req.URL.Query().Get("arr"))
			assert.Empty(t, req.URL.Query().Get("skip"))

			resp, _ := httpmock.NewJsonResponse(200, map[string]interface{}{
				"ok": true,
			})
			return resp, nil
		})
	}

	args := map[string]interface{}{
		"a":    1,
		"b":    "two",
		"arr":  []interface{}{1, "two", true},
		"skip": nil,
	}

	response, err := c.Rpc("test_function", args, &RpcOptions{Get: true}).Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestClient_Rpc_HeadWithQueryParams(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("HEAD", mockPath, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "HEAD", req.Method)
			assert.Equal(t, "value", req.URL.Query().Get("param"))
			assert.Equal(t, "{10,20}", req.URL.Query().Get("ids"))

			resp := httpmock.NewStringResponse(200, "")
			return resp, nil
		})
	}

	args := map[string]interface{}{
		"param": "value",
		"ids":   []interface{}{10, 20},
	}

	response, err := c.Rpc("test_function", args, &RpcOptions{Head: true}).Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 200, response.Status)
	}
}
