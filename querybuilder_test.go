package postgrest

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder_Insert(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("POST", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(201, []map[string]interface{}{
				{"id": 3, "name": "newuser", "email": "newuser@test.com"},
			})
			return resp, nil
		})
	}

	data := map[string]interface{}{
		"name":  "newuser",
		"email": "newuser@test.com",
	}

	response, err := c.From("users").
		Insert(data, nil).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestQueryBuilder_Insert_Array(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("POST", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(201, []map[string]interface{}{
				{"id": 3, "name": "user1"},
				{"id": 4, "name": "user2"},
			})
			return resp, nil
		})
	}

	data := []map[string]interface{}{
		{"name": "user1"},
		{"name": "user2"},
	}

	response, err := c.From("users").
		Insert(data, nil).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestQueryBuilder_Upsert(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("POST", mockPath, func(req *http.Request) (*http.Response, error) {
			prefer := req.Header.Values("Prefer")
			assert.Contains(t, prefer, "resolution=merge-duplicates")
			resp, _ := httpmock.NewJsonResponse(201, []map[string]interface{}{
				{"id": 1, "name": "updated"},
			})
			return resp, nil
		})
	}

	data := map[string]interface{}{
		"id":   1,
		"name": "updated",
	}

	response, err := c.From("users").
		Upsert(data, nil).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestQueryBuilder_Update(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("PATCH", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, []map[string]interface{}{
				{"id": 1, "name": "updated"},
			})
			return resp, nil
		})
	}

	data := map[string]interface{}{
		"name": "updated",
	}

	response, err := c.From("users").
		Update(data, nil).
		Eq("id", 1).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestQueryBuilder_Delete(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("DELETE", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, []map[string]interface{}{
				{"id": 1},
			})
			return resp, nil
		})
	}

	response, err := c.From("users").
		Delete(nil).
		Eq("id", 1).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}
