package postgrest

import (
	"encoding/json"
	"net/http"
	"sort"
	"testing"

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

	client := NewClient("http://localhost:3000", "", nil, "dev")
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
