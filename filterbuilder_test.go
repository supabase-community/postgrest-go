package postgrest

import (
	"net/http"
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

	client := NewClient("http://localhost:3000", "", nil)
	count, err := client.From("users").Select("*", "exact", false).ExecuteTo(&res)
	if err == nil && count > 0 {
		// The value for res will contain all columns for all users, and count will
		// be the exact number of rows in the users table.
	}
}
