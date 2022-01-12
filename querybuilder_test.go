package postgrest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	assert.NotNil(t, NewClient("", "", nil))
}

func TestSelect(t *testing.T) {
	assert := assert.New(t)
	c := createClient(t)

	t.Run("ValidResult", func(t *testing.T) {
		got := []map[string]interface{}{}

		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			responder, _ := httpmock.NewJsonResponder(200, users)
			httpmock.RegisterRegexpResponder("GET", mockPath, responder)
		}

		bs, count, err := c.From("users").Select("id, name, email", "", false).Execute()
		assert.NoError(err)

		err = json.Unmarshal(bs, &got)
		assert.NoError(err)
		assert.EqualValues(users, got)
		assert.Equal(countType(0), count)
	})

	t.Run("WithCount", func(t *testing.T) {
		got := []map[string]interface{}{}

		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterRegexpResponder("GET", mockPath, func(req *http.Request) (*http.Response, error) {
				resp, _ := httpmock.NewJsonResponse(200, users)

				resp.Header.Add("Content-Range", "0-1/2")
				return resp, nil
			})
		}

		bs, count, err := c.From("users").Select("id, name, email", "exact", false).Execute()
		assert.NoError(err)

		err = json.Unmarshal(bs, &got)
		assert.NoError(err)
		assert.EqualValues(users, got)
		assert.Equal(countType(2), count)
	})
}

func TestFilter(t *testing.T) {
	assert := assert.New(t)
	c := createClient(t)

	t.Run("Eq", func(t *testing.T) {
		want := "[{\"email\":\"patti@test.com\"}]"

		if mockResponses {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterRegexpResponder("GET", mockPath, httpmock.NewStringResponder(200, want))
		}

		got, _, err := c.From("users").Select("email", "", false).Eq("email", "patti@test.com").ExecuteString()
		assert.NoError(err)
		assert.Equal(want, got)
	})
}
