package postgrest

import (
	"context"
	"net/http"
	"net/url"
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

		count, err := c.From("users").Select("id, name, email", nil).Eq("name", "sean").ExecuteTo(context.Background(), &got)
		assert.NoError(err)
		assert.EqualValues(want, got)
		if count != nil {
			assert.Equal(int64(0), *count)
		}
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

		opts := &SelectOptions{Count: "exact"}
		count, err := c.From("users").Select("id, name, email", opts).Eq("name", "sean").ExecuteTo(context.Background(), &got)
		assert.NoError(err)
		assert.EqualValues(want, got)
		if count != nil {
			assert.Equal(int64(1), *count)
		}
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
	opts := &SelectOptions{Count: "exact"}
	count, err := client.From("users").Select("*", opts).ExecuteTo(context.Background(), &res)
	if err == nil && count != nil && *count > 0 {
		// The value for res will contain all columns for all users, and count will
		// be the exact number of rows in the users table.
	}
	_ = count
}

func TestFilterBuilder_Limit(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	want := []map[string]interface{}{users[0]}
	var got []map[string]interface{}

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, want)
			resp.Header.Add("Content-Range", "*/2")
			return resp, nil
		})
	}

	opts := &SelectOptions{Count: "exact"}
	response, err := c.From("users").Select("id, name, email", opts).Limit(1, nil).Execute(context.Background())
	assert.NoError(err)
	assert.Nil(response.Error)

	got = response.Data
	assert.EqualValues(want, got)

	// Matching supabase-js, the count returned is not the number of transformed
	// rows, but the number of filtered rows.
	if response.Count != nil {
		assert.Equal(int64(len(users)), *response.Count, "expected count to be %v", len(users))
	}
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

	opts := &SelectOptions{Count: "exact"}
	_, err := c.From("users").Select("id, name, email", opts).Limit(1, nil).Execute(ctx)
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

	var got []map[string]interface{}

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, want)
			resp.Header.Add("Content-Range", "*/2")
			return resp, nil
		})
	}

	opts := &SelectOptions{Count: "exact"}
	orderOpts := &OrderOptions{Ascending: true}
	response, err := c.
		From("users").
		Select("id, name, email", opts).
		Order("name", orderOpts).
		Execute(context.Background())
	assert.NoError(err)
	assert.Nil(response.Error)

	got = response.Data
	assert.EqualValues(want, got)
	if response.Count != nil {
		assert.Equal(int64(len(users)), *response.Count)
	}
}

func TestFilterBuilder_Range(t *testing.T) {
	c := createClient(t)
	assert := assert.New(t)

	want := users
	var got []map[string]interface{}

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, want)
			resp.Header.Add("Content-Range", "*/2")
			return resp, nil
		})
	}

	opts := &SelectOptions{Count: "exact"}
	response, err := c.
		From("users").
		Select("id, name, email", opts).
		Range(0, 1, nil).
		Execute(context.Background())
	assert.NoError(err)
	assert.Nil(response.Error)

	got = response.Data
	assert.EqualValues(want, got)
	if response.Count != nil {
		assert.Equal(int64(len(users)), *response.Count)
	}
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

		opts := &SelectOptions{Count: "exact"}
		response, err := c.
			From("users").
			Select("id, name, email", opts).
			Limit(1, nil).
			Single().
			Execute(context.Background())
		assert.NoError(err)
		assert.Nil(response.Error)

		// response.Data should be []map[string]interface{} with one element
		// Extract the first element for comparison
		if len(response.Data) > 0 {
			got = response.Data[0]
		}
		assert.EqualValues(want, got)
		if response.Count != nil {
			assert.Equal(int64(len(users)), *response.Count)
		}
	})

	// An error will be returned from PostgREST if the total count of the result
	// set > 1, so Single can pretty easily err.
	t.Run("Error", func(t *testing.T) {
		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
				resp, _ := httpmock.NewJsonResponse(500, PostgrestError{
					Message: "error message",
				})

				resp.Header.Add("Content-Range", "*/2")
				return resp, nil
			})
		}

		_, err := c.From("users").Select("*", nil).Single().Execute(context.Background())
		assert.Error(err)
	})
}

func TestFilterAppend(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}]
		expected map[string]string
	}{
		{
			name: "Single filter on column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Eq("age", "25")
			},
			expected: map[string]string{
				"age": "eq.25",
			},
		},
		{
			name: "Multiple filters on same column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Gte("age", "25").Lte("age", "35")
			},
			expected: map[string]string{
				"and": "(age.gte.25,age.lte.35)",
			},
		},
		{
			name: "Three filters on same column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Gte("age", "25").Lte("age", "35").Neq("age", "30")
			},
			expected: map[string]string{
				"and": "(age.gte.25,age.lte.35,age.neq.30)",
			},
		},
		{
			name: "Multiple columns with multiple filters",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Eq("status", "active").Gte("age", "25").Lte("age", "35")
			},
			expected: map[string]string{
				"status": "eq.active",
				"and":    "(age.gte.25,age.lte.35)",
			},
		},
		{
			name: "In filter followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.In("id", []interface{}{"1", "2", "3"}).Eq("id", "4")
			},
			expected: map[string]string{
				"and": "(id.in.(1,2,3),id.eq.4)",
			},
		},
		{
			name: "Contains filter followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Contains("tags", []interface{}{"golang", "postgres"}).Overlaps("tags", []interface{}{"javascript"})
			},
			expected: map[string]string{
				"and": "(tags.cs.{golang,postgres},tags.ov.{javascript})",
			},
		},
		{
			name: "Text search followed by Like filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.TextSearch("title", "golang", &TextSearchOptions{Type: "plain"}).Like("title", "%tutorial%")
			},
			expected: map[string]string{
				"and": "(title.plfts.golang,title.like.%tutorial%)",
			},
		},
		{
			name: "Range filters on same column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
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
			testURL, _ := url.Parse("http://localhost:3000/test")
			builder := NewBuilder[[]map[string]interface{}](client, "GET", testURL, nil)
			fb := &FilterBuilder[[]map[string]interface{}]{
				Builder: builder,
			}

			result := tt.build(fb)

			// Check that we got the expected params
			queryParams := result.url.Query()
			if len(queryParams) != len(tt.expected) {
				t.Errorf("Expected %d params, got %d", len(tt.expected), len(queryParams))
			}

			for key, expectedValue := range tt.expected {
				actualValue := queryParams.Get(key)
				if actualValue == "" {
					t.Errorf("Expected param %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Param %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}

			// Check that no unexpected params exist
			for key := range queryParams {
				if _, ok := tt.expected[key]; !ok && key != "select" {
					t.Errorf("Unexpected param %s found", key)
				}
			}
		})
	}
}

func TestBuilder_ThrowOnError(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(500, PostgrestError{
				Message: "test error",
			})
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		ThrowOnError().
		Execute(context.Background())

	if mockResponses {
		assert.Error(t, err)
		assert.Nil(t, response)
	}
}

func TestBuilder_SetHeader(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			// Verify custom header is set
			assert.Equal(t, "custom-value", req.Header.Get("X-Custom-Header"))
			resp, _ := httpmock.NewJsonResponse(200, users)
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		SetHeader("X-Custom-Header", "custom-value").
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Gt(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").Select("*", nil).Gt("id", 1).Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Lt(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").Select("*", nil).Lt("id", 10).Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Like(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").Select("*", nil).Like("name", "%sean%").Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Ilike(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").Select("*", nil).Ilike("name", "%SEAN%").Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Is(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").Select("*", nil).Is("deleted_at", nil).Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Match(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		Match(map[string]interface{}{
			"name":  "sean",
			"email": "sean@test.com",
		}).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Not(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").Select("*", nil).Not("status", "eq", "deleted").Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Or(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		Or("status.eq.ONLINE,status.eq.OFFLINE", nil).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestPostgrestError_Error(t *testing.T) {
	err := NewPostgrestError("test message", "details", "hint", "code")
	assert.Equal(t, "test message", err.Error())
}

func TestFilterBuilder_LikeAllOf(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		LikeAllOf("name", []string{"%sean%", "%test%"}).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_LikeAnyOf(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		LikeAnyOf("name", []string{"%sean%", "%patti%"}).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_IlikeAllOf(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		IlikeAllOf("name", []string{"%SEAN%", "%TEST%"}).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_IlikeAnyOf(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		IlikeAnyOf("name", []string{"%SEAN%", "%PATTI%"}).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_ContainedBy(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		ContainedBy("tags", []interface{}{"golang", "postgres"}).
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_RangeGte(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		RangeGte("period", "[2022-01-01,2022-12-31]").
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_RangeLte(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		RangeLte("period", "[2023-01-01,2023-12-31]").
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_RangeAdjacent(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		RangeAdjacent("period", "[2022-01-01,2022-12-31]").
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Filter(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("*", nil).
		Filter("age", "gte", "25").
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_Filter_InvalidOperator(t *testing.T) {
	c := createClient(t)

	// Filter with invalid operator should be ignored
	response, err := c.From("users").
		Select("*", nil).
		Filter("age", "invalid", "25").
		Execute(context.Background())

	// Should not error, just ignore invalid operator
	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestFilterBuilder_Select(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, "[]"))
	}

	response, err := c.From("users").
		Select("id, name", nil).
		Eq("id", 1).
		Select("id, name, email").
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}

func TestFilterBuilder_MaybeSingle(t *testing.T) {
	c := createClient(t)

	if mockResponses {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, users[0])
			return resp, nil
		})
	}

	response, err := c.From("users").
		Select("*", nil).
		Limit(1, nil).
		MaybeSingle().
		Execute(context.Background())

	if mockResponses {
		assert.NoError(t, err)
		assert.NotNil(t, response)
	}
}
