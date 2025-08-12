package postgrest

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFilterBuilder_ExecuteTo(t *testing.T) {
	assert := assert.New(t)
	c := createClient(t)

	t.Run("ValidResult", func(t *testing.T) {
		want := []TestResult{
			{
				ID:    float64(1),
				Name:  "sean",
				Email: "sean@test.com",
			},
		}

		var got []TestResult

		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			responder, _ := httpmock.NewJsonResponder(200, []map[string]interface{}{
				users[0],
			})
			httpmock.RegisterRegexpResponder("GET", mockPath, responder)
		}

		count, err := c.From("users").Select("id, name, email", "", false).Eq("name", "sean").ExecuteTo(&got)
		assert.NoError(err)
		assert.EqualValues(want, got)
		assert.Equal(countType(0), count)
	})

	t.Run("WithCount", func(t *testing.T) {
		want := []TestResult{
			{
				ID:    float64(1),
				Name:  "sean",
				Email: "sean@test.com",
			},
		}

		var got []TestResult

		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
				resp, _ := httpmock.NewJsonResponse(200, []map[string]interface{}{
					users[0],
				})

				resp.Header.Add("Content-Range", "0-1/1")
				return resp, nil
			})
		}

		count, err := c.From("users").Select("id, name, email", "exact", false).Eq("name", "sean").ExecuteTo(&got)
		assert.NoError(err)
		assert.EqualValues(want, got)
		assert.Equal(countType(1), count)
	})
}

func ExampleFilterBuilder_ExecuteTo() {
	// Given a database with a "users" table containing "id", "name" and "email"
	// columns:
	var res []struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	client := NewClient("http://localhost:3000", "", nil)
	count, err := client.From("users").Select("*", "exact", false).ExecuteTo(&res)
	if err == nil && count > 0 {
		// The value for res will contain all columns for all users, and count will
		// be the exact number of rows in the users table.
	}
}

func TestFilterBuilder_Limit(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	want := []map[string]interface{}{users[0]}
	got := []map[string]interface{}{}

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, want)
			resp.Header.Add("Content-Range", "*/2")
			return resp, nil
		})
	}

	bs, count, err := c.From("users").Select("id, name, email", "exact", false).Limit(1, "").Execute()
	assert.NoError(err)

	err = json.Unmarshal(bs, &got)
	assert.NoError(err)
	assert.EqualValues(want, got)

	// Matching supabase-js, the count returned is not the number of transformed
	// rows, but the number of filtered rows.
	assert.Equal(countType(len(users)), count, "expected count to be %v", len(users))
}

func TestFilterBuilder_ContextCanceled(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Nanosecond)

	_, _, err := c.From("users").Select("id, name, email", "exact", false).Limit(1, "").ExecuteWithContext(ctx)
	// This test should immediately fail on a canceled context.
	assert.Error(err)
}

func TestFilterBuilder_Order(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	want := make([]map[string]interface{}, len(users))
	copy(want, users)

	sort.Slice(want, func(i, j int) bool {
		return j < i
	})

	got := []map[string]interface{}{}

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, want)
			resp.Header.Add("Content-Range", "*/2")
			return resp, nil
		})
	}

	bs, count, err := c.
		From("users").
		Select("id, name, email", "exact", false).
		Order("name", &OrderOpts{Ascending: true}).
		Execute()
	assert.NoError(err)

	err = json.Unmarshal(bs, &got)
	assert.NoError(err)
	assert.EqualValues(want, got)
	assert.Equal(countType(len(users)), count)
}

func TestFilterBuilder_Range(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	want := users
	got := []map[string]interface{}{}

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, want)
			resp.Header.Add("Content-Range", "*/2")
			return resp, nil
		})
	}

	bs, count, err := c.
		From("users").
		Select("id, name, email", "exact", false).
		Range(0, 1, "").
		Execute()
	assert.NoError(err)

	err = json.Unmarshal(bs, &got)
	assert.NoError(err)
	assert.EqualValues(want, got)
	assert.Equal(countType(len(users)), count)
}

func TestFilterBuilder_Single(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	want := users[0]
	got := make(map[string]interface{})

	t.Run("ValidResult", func(t *testing.T) {
		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
				resp, _ := httpmock.NewJsonResponse(200, want)
				resp.Header.Add("Content-Range", "*/2")
				return resp, nil
			})
		}

		bs, count, err := c.
			From("users").
			Select("id, name, email", "exact", false).
			Limit(1, "").
			Single().
			Execute()
		assert.NoError(err)

		err = json.Unmarshal(bs, &got)
		assert.NoError(err)
		assert.EqualValues(want, got)
		assert.Equal(countType(len(users)), count)
	})

	// An error will be returned from PostgREST if the total count of the result
	// set > 1, so Single can pretty easily err.
	t.Run("Error", func(t *testing.T) {
		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
				resp, _ := httpmock.NewJsonResponse(500, ExecuteError{
					Message: "error message",
				})

				resp.Header.Add("Content-Range", "*/2")
				return resp, nil
			})
		}

		_, _, err := c.From("users").Select("*", "", false).Single().Execute()
		assert.Error(err)
	})
}

func TestFilterAppend(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*FilterBuilder) *FilterBuilder
		expected map[string]string
	}{
		{
			name: "Single filter on column",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.Eq("age", "25")
			},
			expected: map[string]string{
				"age": "eq.25",
			},
		},
		{
			name: "Multiple filters on same column",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.Gte("age", "25").Lte("age", "35")
			},
			expected: map[string]string{
				"and": "(age.gte.25,age.lte.35)",
			},
		},
		{
			name: "Three filters on same column",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.Gte("age", "25").Lte("age", "35").Neq("age", "30")
			},
			expected: map[string]string{
				"and": "(age.gte.25,age.lte.35,age.neq.30)",
			},
		},
		{
			name: "Multiple columns with multiple filters",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.Eq("status", "active").Gte("age", "25").Lte("age", "35")
			},
			expected: map[string]string{
				"status": "eq.active",
				"and":    "(age.gte.25,age.lte.35)",
			},
		},
		{
			name: "In filter followed by another filter",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.In("id", []string{"1", "2", "3"}).Eq("id", "4")
			},
			expected: map[string]string{
				"and": "(id.in.(1,2,3),id.eq.4)",
			},
		},
		{
			name: "Contains filter followed by another filter",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.Contains("tags", []string{"golang", "postgres"}).Overlaps("tags", []string{"javascript"})
			},
			expected: map[string]string{
				"and": "(tags.cs.{\"golang\",\"postgres\"},tags.ov.{\"javascript\"})",
			},
		},
		{
			name: "Text search followed by Like filter",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.TextSearch("title", "golang", "", "plain").Like("title", "%tutorial%")
			},
			expected: map[string]string{
				"and": "(title.plfts.golang,title.like.%tutorial%)",
			},
		},
		{
			name: "Range filters on same column",
			build: func(fb *FilterBuilder) *FilterBuilder {
				return fb.RangeGt("period", "[2022-01-01,2022-12-31]").RangeLt("period", "[2023-01-01,2023-12-31]")
			},
			expected: map[string]string{
				"and": "(period.sr.[2022-01-01,2022-12-31],period.sl.[2023-01-01,2023-12-31])",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("http://localhost:3000", "", nil)
			fb := &FilterBuilder{
				client:    client,
				method:    "GET",
				tableName: "test",
				headers:   make(map[string]string),
				params:    make(map[string]string),
			}

			result := tt.build(fb)

			// Check that we got the expected params
			if len(result.params) != len(tt.expected) {
				t.Errorf("Expected %d params, got %d", len(tt.expected), len(result.params))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result.params[key]; !ok {
					t.Errorf("Expected param %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Param %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}

			// Check that no unexpected params exist
			for key := range result.params {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("Unexpected param %s found", key)
				}
			}
		})
	}
}
