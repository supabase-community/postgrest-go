package postgrest

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestTransformBuilder_CSV(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "text/csv", req.Header.Get("Accept"))
			resp := httpmock.NewStringResponse(200, "id,name,email\n1,sean,sean@test.com")
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		CSV().
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Contains(t, response.Data, "sean")
	}
}

func TestTransformBuilder_GeoJSON(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "application/geo+json", req.Header.Get("Accept"))
			resp, _ := httpmock.NewJsonResponse(200, map[string]interface{}{
				"type":     "FeatureCollection",
				"features": []interface{}{},
			})
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		GeoJSON().
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestTransformBuilder_Explain(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			accept := req.Header.Get("Accept")
			assert.Contains(t, accept, "application/vnd.pgrst.plan+text")
			resp := httpmock.NewStringResponse(200, "Seq Scan on users")
			return resp, nil
		})
	}

	opts := &ExplainOptions{
		Format:  "text",
		Analyze: true,
		Verbose: true,
	}

	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		Explain(opts).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestTransformBuilder_Explain_NilOptions(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "Seq Scan"))
	}

	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		Explain(nil).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestTransformBuilder_Rollback(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			prefer := req.Header.Values("Prefer")
			assert.Contains(t, prefer, "tx=rollback")
			resp, _ := httpmock.NewJsonResponse(200, []interface{}{})
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		Rollback().
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestTransformBuilder_MaxAffected(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			prefer := req.Header.Values("Prefer")
			assert.Contains(t, prefer, "handling=strict")
			assert.Contains(t, prefer, "max-affected=10")
			resp, _ := httpmock.NewJsonResponse(200, []interface{}{})
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		MaxAffected(10).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestTransformBuilder_AbortSignal(t *testing.T) {
	c := createClient(t)

	// AbortSignal is a no-op in Go, just test it doesn't break
	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		AbortSignal(nil).
		Execute(context.Background())

	// Should not error
	assert.NoError(t, err)
	assert.NotNil(t, response)
}
