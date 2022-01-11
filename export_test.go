package postgrest

import (
	"os"
	"regexp"
	"testing"
)

const urlEnv = "POSTGREST_URL"
const apiKeyEnv = "API_KEY"

// If false, mock responses with httpmock. If true, use POSTGREST_URL (and
// optionally, API_KEY for Supabase), to run tests against an actual Postgres
// instance.
var mockResponses bool = false

var mockPath *regexp.Regexp

// A mock table/result set.
var users = []map[string]interface{}{
	{
		"id":    float64(1), // numeric types are returned as float64s
		"name":  "sean",
		"email": "sean@test.com",
	},
	{
		"id":    float64(2),
		"name":  "patti",
		"email": "patti@test.com",
	},
}

func createClient(t *testing.T) *Client {
	// If a POSTGREST_URL environment variable is specified, we'll use that
	// to test against real endpoints.
	url := os.Getenv(urlEnv)
	if url == "" {
		url = "http://mock.xyz"
		mockResponses = true

		var err error
		mockPath, err = regexp.Compile(regexp.QuoteMeta(url) + "?.*")
		if err != nil {
			t.Fatal(err)
		}
	}

	headers := make(map[string]string)
	if apiKeyEnv != "" {
		// If the API_KEY env is specified, we'll use it to auth with Supabase.
		headers["apikey"] = os.Getenv(apiKeyEnv)
	}

	return NewClient(url, "", headers)
}
