package postgrest

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder_Select_HeadOptionUsesHeadMethod(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	fb := qb.Select("id, name", &SelectOptions{Head: true})

	assert.Equal(t, "HEAD", fb.method)
	assert.Equal(t, "id,name", fb.url.Query().Get("select"))
}

func TestQueryBuilder_Select_EmptyColumnsDefaultsToStar(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	fb := qb.Select("", nil)

	assert.Equal(t, "GET", fb.method)
	assert.Equal(t, "*", fb.url.Query().Get("select"))
}

func TestQueryBuilder_Select_QuotedWhitespacePreserved(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	fb := qb.Select(`id, "full name", email`, nil)

	assert.Equal(t, `id,"full name",email`, fb.url.Query().Get("select"))
}

func TestQueryBuilder_Insert_AddsCountPreferHeaderForSupportedValues(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")
	data := map[string]interface{}{"name": "newuser"}

	tests := []string{"exact", "planned", "estimated"}
	for _, count := range tests {
		t.Run(count, func(t *testing.T) {
			fb := qb.Insert(data, &InsertOptions{
				Count:         count,
				DefaultToNull: true,
			})

			prefer := fb.headers.Values("Prefer")
			assert.Contains(t, prefer, "count="+count)
			assert.NotContains(t, prefer, "missing=default")
		})
	}
}

func TestQueryBuilder_Insert_AddsMissingDefaultPreferWhenDefaultToNullFalse(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	fb := qb.Insert(map[string]interface{}{"name": "newuser"}, &InsertOptions{
		DefaultToNull: false,
	})

	prefer := fb.headers.Values("Prefer")
	assert.Contains(t, prefer, "missing=default")
}

func TestQueryBuilder_Upsert_IgnoreDuplicatesAndOnConflict(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	fb := qb.Upsert(map[string]interface{}{"id": 1, "name": "updated"}, &UpsertOptions{
		IgnoreDuplicates: true,
		OnConflict:       "id,name",
		DefaultToNull:    true,
	})

	prefer := fb.headers.Values("Prefer")
	assert.Contains(t, prefer, "resolution=ignore-duplicates")
	assert.Equal(t, "id,name", fb.url.Query().Get("on_conflict"))
}

func TestQueryBuilder_Upsert_AddsCountPreferHeaderForSupportedValues(t *testing.T) {
	tests := []string{"exact", "planned", "estimated"}
	for _, count := range tests {
		t.Run(count, func(t *testing.T) {
			c := NewClient("http://localhost:3000", "", nil)
			qb := NewQueryBuilder[map[string]interface{}](c, "users")

			fb := qb.Upsert(map[string]interface{}{"id": 1, "name": "updated"}, &UpsertOptions{
				Count:         count,
				DefaultToNull: true,
			})

			prefer := fb.headers.Values("Prefer")
			assert.Contains(t, prefer, "resolution=merge-duplicates")
			assert.Contains(t, prefer, "count="+count)
			assert.NotContains(t, prefer, "missing=default")
		})
	}
}

func TestQueryBuilder_Upsert_AddsMissingDefaultPreferWhenDefaultToNullFalse(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	fb := qb.Upsert(map[string]interface{}{"id": 1, "name": "updated"}, &UpsertOptions{
		DefaultToNull: false,
	})

	prefer := fb.headers.Values("Prefer")
	assert.Contains(t, prefer, "resolution=merge-duplicates")
	assert.Contains(t, prefer, "missing=default")
}

func TestQueryBuilder_Upsert_ArrayPayloadSetsColumnsQuery(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	qb := NewQueryBuilder[map[string]interface{}](c, "users")

	data := []map[string]interface{}{
		{"id": 1, "name": "alice"},
		{"id": 2, "email": "bob@test.com"},
	}

	fb := qb.Upsert(data, &UpsertOptions{DefaultToNull: true})

	columns := fb.url.Query().Get("columns")
	assert.NotEmpty(t, columns)

	parts := strings.Split(columns, ",")
	assert.Len(t, parts, 3)
	assert.Contains(t, parts, `"id"`)
	assert.Contains(t, parts, `"name"`)
	assert.Contains(t, parts, `"email"`)
}

func TestQueryBuilder_Update_AddsCountPreferHeaderForSupportedValues(t *testing.T) {
	tests := []string{"exact", "planned", "estimated"}
	for _, count := range tests {
		t.Run(count, func(t *testing.T) {
			c := NewClient("http://localhost:3000", "", nil)
			qb := NewQueryBuilder[map[string]interface{}](c, "users")

			fb := qb.Update(map[string]interface{}{"name": "updated"}, &UpdateOptions{
				Count: count,
			})

			prefer := fb.headers.Values("Prefer")
			assert.Contains(t, prefer, "count="+count)
		})
	}
}

func TestQueryBuilder_Delete_AddsCountPreferHeaderForSupportedValues(t *testing.T) {
	tests := []string{"exact", "planned", "estimated"}
	for _, count := range tests {
		t.Run(count, func(t *testing.T) {
			c := NewClient("http://localhost:3000", "", nil)
			qb := NewQueryBuilder[map[string]interface{}](c, "users")

			fb := qb.Delete(&DeleteOptions{
				Count: count,
			})

			prefer := fb.headers.Values("Prefer")
			assert.Contains(t, prefer, "count="+count)
		})
	}
}

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
