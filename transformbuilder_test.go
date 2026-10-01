package postgrest

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestTransformBuilder_Select_QuotedWhitespacePreserved(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	result := tb.Select(`id, "full name", email`)
	query := result.url.Query()

	assert.Equal(t, `id,"full name",email`, query.Get("select"))
	assert.Contains(t, result.headers.Values("Prefer"), "return=representation")
}

func TestTransformBuilder_Select_EmptyDefaultsToStar(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	result := tb.Select("")
	query := result.url.Query()

	assert.Equal(t, "*", query.Get("select"))
}

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

func TestTransformBuilder_Explain_OptionsAndDefaultMediaType(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	explained := tb.Explain(&ExplainOptions{
		Format:   "text",
		Analyze:  true,
		Verbose:  true,
		Settings: true,
		Buffers:  true,
		WAL:      true,
	})

	accept := explained.headers.Get("Accept")
	assert.Contains(t, accept, "application/vnd.pgrst.plan+text")
	assert.Contains(t, accept, `for="application/json"`)
	assert.Contains(t, accept, "options=analyze|verbose|settings|buffers|wal;")
}

func TestTransformBuilder_Range_ReferencedTable(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	tb.Range(2, 5, &RangeOptions{ReferencedTable: "messages"})
	query := tb.url.Query()

	assert.Equal(t, "2", query.Get("messages.offset"))
	assert.Equal(t, "4", query.Get("messages.limit"))
	assert.Empty(t, query.Get("offset"))
	assert.Empty(t, query.Get("limit"))
}

func TestTransformBuilder_Limit_ReferencedTable(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	tb.Limit(7, &LimitOptions{ReferencedTable: "messages"})
	query := tb.url.Query()

	assert.Equal(t, "7", query.Get("messages.limit"))
	assert.Empty(t, query.Get("limit"))
}

func TestTransformBuilder_Order_ReferencedTableAndNulls(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	nullsFirst := true
	tb.Order("created_at", &OrderOptions{
		Ascending:       true,
		NullsFirst:      &nullsFirst,
		ReferencedTable: "messages",
	})
	query := tb.url.Query()

	assert.Equal(t, "created_at.asc.nullsfirst", query.Get("messages.order"))
	assert.Empty(t, query.Get("order"))
}

func TestTransformBuilder_Order_NullsLastAndExistingOrderAppend(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:3000/users?order=id.desc")
	tb := &TransformBuilder[[]map[string]interface{}]{
		Builder: &Builder[[]map[string]interface{}]{
			method:  "GET",
			url:     testURL,
			headers: make(http.Header),
		},
	}

	nullsFirst := false
	tb.Order("name", &OrderOptions{
		Ascending:  true,
		NullsFirst: &nullsFirst,
	})
	query := tb.url.Query()

	assert.Equal(t, "id.desc,name.asc.nullslast", query.Get("order"))
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
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, "[]")
			return resp, nil
		}),
	}

	// AbortSignal is a no-op in Go; verify it does not break execution.
	response, err := c.From("users").
		Select("*", nil).
		Order("id", nil).
		AbortSignal(nil).
		Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
}
