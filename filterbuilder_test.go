package postgrest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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

func TestBuilder_ExecuteTo_ReturnsExecuteError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users?select=*")
	builder := NewBuilder[map[string]interface{}](c, "GET", testURL, nil)

	signalCtx, cancel := context.WithCancel(context.Background())
	cancel()
	builder.signal = signalCtx

	var target map[string]interface{}
	count, err := builder.ExecuteTo(context.Background(), &target)

	assert.Nil(t, count)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestBuilder_ExecuteTo_ReturnsResponseError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		}),
	}

	var target []map[string]interface{}
	count, err := c.From("users").Select("*", nil).ExecuteTo(context.Background(), &target)

	assert.Nil(t, count)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FetchError:")
}

func TestBuilder_ExecuteTo_ReturnsMarshalResponseDataError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users")
	builder := NewBuilder[chan int](c, "HEAD", testURL, nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	var target interface{}
	count, err := builder.ExecuteTo(context.Background(), &target)

	assert.Nil(t, count)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error marshaling response data")
}

func TestBuilder_ExecuteTo_ReturnsUnmarshalTargetError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, []map[string]interface{}{{"id": 1}})
			return resp, nil
		}),
	}

	var target int
	count, err := c.From("users").Select("*", nil).ExecuteTo(context.Background(), &target)

	assert.Nil(t, count)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshaling to target")
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
		expected url.Values
	}{
		{
			name: "Single filter on column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Eq("age", "25")
			},
			expected: url.Values{
				"age": {"eq.25"},
			},
		},
		{
			name: "Multiple filters on same column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Gte("age", "25").Lte("age", "35")
			},
			expected: url.Values{
				"age": {"gte.25", "lte.35"},
			},
		},
		{
			name: "Three filters on same column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Gte("age", "25").Lte("age", "35").Neq("age", "30")
			},
			expected: url.Values{
				"age": {"gte.25", "lte.35", "neq.30"},
			},
		},
		{
			name: "Multiple columns with multiple filters",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Eq("status", "active").Gte("age", "25").Lte("age", "35")
			},
			expected: url.Values{
				"status": {"eq.active"},
				"age":    {"gte.25", "lte.35"},
			},
		},
		{
			name: "In filter followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.In("id", []interface{}{"1", "2", "3"}).Eq("id", "4")
			},
			expected: url.Values{
				"id": {"in.(1,2,3)", "eq.4"},
			},
		},
		{
			name: "In filter quotes values with reserved characters",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.In("id", []interface{}{"abc,def", "a(b)", "plain"})
			},
			expected: url.Values{
				"id": {`in.("abc,def","a(b)",plain)`},
			},
		},
		{
			name: "Contains filter followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Contains("tags", []interface{}{"golang", "postgres"}).Overlaps("tags", []interface{}{"javascript"})
			},
			expected: url.Values{
				"tags": {"cs.{golang,postgres}", "ov.{javascript}"},
			},
		},
		{
			name: "Contains range string followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Contains("period", "[2022-01-01,2022-12-31]").Eq("status", "active")
			},
			expected: url.Values{
				"period": {"cs.[2022-01-01,2022-12-31]"},
				"status": {"eq.active"},
			},
		},
		{
			name: "Contains JSON value followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Contains("metadata", map[string]interface{}{"role": "admin"}).Eq("status", "active")
			},
			expected: url.Values{
				"metadata": {`cs.{"role":"admin"}`},
				"status":   {"eq.active"},
			},
		},
		{
			name: "Overlaps range string followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Overlaps("period", "[2022-01-01,2022-12-31]").Eq("status", "active")
			},
			expected: url.Values{
				"period": {"ov.[2022-01-01,2022-12-31]"},
				"status": {"eq.active"},
			},
		},
		{
			name: "Overlaps unsupported type is ignored",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Overlaps("metadata", map[string]interface{}{"role": "admin"}).Eq("status", "active")
			},
			expected: url.Values{
				"status": {"eq.active"},
			},
		},
		{
			name: "ContainedBy range string followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.ContainedBy("period", "[2022-01-01,2022-12-31]").Eq("status", "active")
			},
			expected: url.Values{
				"period": {"cd.[2022-01-01,2022-12-31]"},
				"status": {"eq.active"},
			},
		},
		{
			name: "ContainedBy JSON value followed by another filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.ContainedBy("metadata", map[string]interface{}{"role": "admin"}).Eq("status", "active")
			},
			expected: url.Values{
				"metadata": {`cd.{"role":"admin"}`},
				"status":   {"eq.active"},
			},
		},
		{
			name: "Text search followed by Like filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.TextSearch("title", "golang", &TextSearchOptions{Type: "plain"}).Like("title", "%tutorial%")
			},
			expected: url.Values{
				"title": {"plfts.golang", "like.%tutorial%"},
			},
		},
		{
			name: "Phrase text search followed by Like filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.TextSearch("title", "golang tutorial", &TextSearchOptions{Type: "phrase"}).Like("title", "%tutorial%")
			},
			expected: url.Values{
				"title": {"phfts.golang tutorial", "like.%tutorial%"},
			},
		},
		{
			name: "Websearch text search followed by Like filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.TextSearch("title", "golang tutorial", &TextSearchOptions{Type: "websearch"}).Like("title", "%tutorial%")
			},
			expected: url.Values{
				"title": {"wfts.golang tutorial", "like.%tutorial%"},
			},
		},
		{
			name: "Text search with config followed by Like filter",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.TextSearch("title", "golang", &TextSearchOptions{Type: "plain", Config: "english"}).Like("title", "%tutorial%")
			},
			expected: url.Values{
				"title": {"plfts(english).golang", "like.%tutorial%"},
			},
		},
		{
			name: "Or filter with referenced table key",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.Or("status.eq.ONLINE,status.eq.OFFLINE", &OrOptions{ReferencedTable: "messages"}).Eq("id", "1")
			},
			expected: url.Values{
				"messages.or": {"(status.eq.ONLINE,status.eq.OFFLINE)"},
				"id":          {"eq.1"},
			},
		},
		{
			name: "Range filters on same column",
			build: func(fb *FilterBuilder[[]map[string]interface{}]) *FilterBuilder[[]map[string]interface{}] {
				return fb.RangeGt("period", "[2022-01-01,2022-12-31]").RangeLt("period", "[2023-01-01,2023-12-31]")
			},
			expected: url.Values{
				"period": {"sr.[2022-01-01,2022-12-31]", "sl.[2023-01-01,2023-12-31]"},
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
			queryParams := result.url.Query()

			for key, expectedValues := range tt.expected {
				actualValues, ok := queryParams[key]
				if !ok {
					t.Errorf("expected param %s not found", key)
					continue
				}
				if len(actualValues) != len(expectedValues) {
					t.Errorf("param %s: expected %d values, got %d (%v)", key, len(expectedValues), len(actualValues), actualValues)
					continue
				}
				for i, exp := range expectedValues {
					if actualValues[i] != exp {
						t.Errorf("param %s[%d]: expected %q, got %q", key, i, exp, actualValues[i])
					}
				}
			}

			for key := range queryParams {
				if _, ok := tt.expected[key]; !ok && key != "select" {
					t.Errorf("unexpected param %s found", key)
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

func TestBuilder_Execute_RequestError(t *testing.T) {
	c := createClient(t)
	transportErr := errors.New("network down")
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, transportErr
		}),
	}

	t.Run("returns fetch error response when throwOnError disabled", func(t *testing.T) {
		response, err := c.From("users").
			Select("*", nil).
			Execute(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotNil(t, response.Error)
		assert.Contains(t, response.Error.Message, "FetchError:")
		assert.Contains(t, response.Error.Message, "network down")
		assert.Contains(t, response.Error.Details, "network down")
		assert.Equal(t, 0, response.Status)
	})

	t.Run("returns transport error when throwOnError enabled", func(t *testing.T) {
		response, err := c.From("users").
			Select("*", nil).
			ThrowOnError().
			Execute(context.Background())

		assert.ErrorIs(t, err, transportErr)
		assert.Nil(t, response)
	})
}

func TestBuilder_Execute_UsesBackgroundWhenContextIsNil(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	requestExecuted := false
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			requestExecuted = true
			resp, _ := httpmock.NewJsonResponse(200, []map[string]interface{}{})
			return resp, nil
		}),
	}

	var nilCtx context.Context
	response, err := c.From("users").Select("*", nil).Execute(nilCtx)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, requestExecuted)
}

func TestBuilder_Execute_SignalOverridesPassedContext(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users?select=*")
	builder := NewBuilder[map[string]interface{}](c, "GET", testURL, nil)

	signalCtx, cancel := context.WithCancel(context.Background())
	cancel()
	builder.signal = signalCtx

	response, err := builder.Execute(context.Background())

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, response)
}

func TestBuilder_Execute_ReturnsMarshalBodyError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users")
	builder := NewBuilder[map[string]interface{}](c, "POST", testURL, &BuilderOptions{
		Body: map[string]interface{}{
			"invalid": make(chan int),
		},
	})

	response, err := builder.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error marshaling body")
}

func TestBuilder_Execute_ReturnsContextErrorBeforeRequestCreation(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users?select=*")
	builder := NewBuilder[map[string]interface{}](c, "GET", testURL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	response, err := builder.Execute(ctx)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, response)
}

func TestBuilder_Execute_ReturnsCreateRequestError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users")
	builder := NewBuilder[map[string]interface{}](c, "BAD METHOD", testURL, nil)

	response, err := builder.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error creating request")
}

func TestBuilder_Execute_404WithEmptyBodyReturnsNoContent(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Nil(t, response.Error)
	assert.Equal(t, 204, response.Status)
	assert.Equal(t, "No Content", response.StatusText)
}

func TestBuilder_Execute_404WithArrayBodyReturnsOKAndZeroData(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("null")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Nil(t, response.Error)
	assert.Equal(t, 200, response.Status)
	assert.Equal(t, "OK", response.StatusText)
	assert.Nil(t, response.Data)
}

func TestBuilder_Execute_MaybeSingleClearsZeroRowsError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 406,
				Status:     "406 Not Acceptable",
				Body: io.NopCloser(strings.NewReader(
					`{"message":"JSON object requested, multiple (or no) rows returned","details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","hint":"","code":"PGRST116"}`,
				)),
				Header: make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).MaybeSingle().Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Nil(t, response.Error)
	assert.Equal(t, 200, response.Status)
	assert.Equal(t, "OK", response.StatusText)
}

func TestBuilder_Execute_SingleObjectIntoSliceType(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{"id":1,"name":"sean"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).Single().Execute(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.Data, 1)
	assert.Equal(t, "sean", response.Data[0]["name"])
}

func TestBuilder_Execute_SingleObjectArrayUnmarshalError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{"id":`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).Single().Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error unmarshaling single object array")
}

func TestBuilder_Execute_SingleObjectUnmarshalErrorNonSlice(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users?select=*")
	builder := NewBuilder[map[string]interface{}](c, "GET", testURL, nil)
	builder.headers.Set("Accept", "application/vnd.pgrst.object+json")
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{"id":`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := builder.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error unmarshaling single object")
}

func TestBuilder_Execute_MaybeSingleItemUnmarshalError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users?select=*")
	builder := NewBuilder[int](c, "GET", testURL, nil)
	builder.isMaybeSingle = true
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`[{"id":1}]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := builder.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error unmarshaling maybeSingle item")
}

func TestBuilder_Execute_MaybeSingleObjectArrayUnmarshalError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{"id":`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).MaybeSingle().Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error unmarshaling single object array")
}

func TestBuilder_Execute_MaybeSingleObjectUnmarshalErrorNonSlice(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	testURL, _ := url.Parse("http://localhost:3000/users?select=*")
	builder := NewBuilder[map[string]interface{}](c, "GET", testURL, nil)
	builder.isMaybeSingle = true
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{"id":`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := builder.Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error unmarshaling response")
}

func TestBuilder_Execute_DefaultResponseUnmarshalError(t *testing.T) {
	c := NewClient("http://localhost:3000", "", nil)
	c.session = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader(`{"id":`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	response, err := c.From("users").Select("*", nil).Execute(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "error unmarshaling response")
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

func TestFilterBuilder_MaybeSingleWrapper(t *testing.T) {
	t.Run("GET uses json accept and marks maybeSingle", func(t *testing.T) {
		testURL, _ := url.Parse("http://localhost:3000/users")
		builder := NewBuilder[map[string]interface{}](NewClient("http://localhost:3000", "", nil), "GET", testURL, nil)
		fb := &FilterBuilder[map[string]interface{}]{Builder: builder}

		result := fb.MaybeSingle()

		assert.Same(t, builder, result)
		assert.True(t, result.isMaybeSingle)
		assert.Equal(t, "application/json", result.headers.Get("Accept"))
	})

	t.Run("non-GET uses object json accept and marks maybeSingle", func(t *testing.T) {
		testURL, _ := url.Parse("http://localhost:3000/users")
		builder := NewBuilder[map[string]interface{}](NewClient("http://localhost:3000", "", nil), "POST", testURL, nil)
		fb := &FilterBuilder[map[string]interface{}]{Builder: builder}

		result := fb.MaybeSingle()

		assert.Same(t, builder, result)
		assert.True(t, result.isMaybeSingle)
		assert.Equal(t, "application/vnd.pgrst.object+json", result.headers.Get("Accept"))
	})
}

func TestBuilder_Execute_MaybeSingle_ArrayBranches(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		assertResult func(t *testing.T, response *PostgrestResponse[map[string]interface{}], err error)
	}{
		{
			name: "returns 406 when array has multiple rows",
			body: `[{"id":1},{"id":2}]`,
			assertResult: func(t *testing.T, response *PostgrestResponse[map[string]interface{}], err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotNil(t, response.Error)
				assert.Equal(t, "PGRST116", response.Error.Code)
				assert.Equal(t, 406, response.Status)
				assert.Equal(t, "Not Acceptable", response.StatusText)
			},
		},
		{
			name: "unmarshals the only row when array has one row",
			body: `[{"id":1,"name":"sean","email":"sean@test.com"}]`,
			assertResult: func(t *testing.T, response *PostgrestResponse[map[string]interface{}], err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Nil(t, response.Error)
				assert.Equal(t, "sean", response.Data["name"])
				assert.Equal(t, "sean@test.com", response.Data["email"])
			},
		},
		{
			name: "returns null equivalent when array is empty",
			body: `[]`,
			assertResult: func(t *testing.T, response *PostgrestResponse[map[string]interface{}], err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Nil(t, response.Error)
				assert.Nil(t, response.Data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient("http://localhost:3000", "", nil)
			c.session = &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(tt.body)),
						Header:     make(http.Header),
					}, nil
				}),
			}

			testURL, _ := url.Parse("http://localhost:3000/users?select=*")
			builder := NewBuilder[map[string]interface{}](c, "GET", testURL, nil)
			builder.isMaybeSingle = true
			builder.headers.Set("Accept", "application/json")

			response, err := builder.Execute(context.Background())
			tt.assertResult(t, response, err)
		})
	}
}
