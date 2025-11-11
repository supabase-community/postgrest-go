package postgrest

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

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
