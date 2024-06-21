//go:build integration

package integration_test

import (
	"github.com/supabase-community/postgrest-go"
	"testing"
)

func TestNewClient_Test(t *testing.T) {
	rawURL := "http://localhost:3000"
	schema := "public"
	headers := map[string]string{}

	client := postgrest.NewClient(rawURL, schema, headers, "prod")
	if client == nil {
		t.Error("client is nil")
	}

	execute, _, err := client.From("users").Select("username", "exact", false).Execute()
	if err != nil {
		return
	}

	t.Log(string(execute))
}
