// This is basic example for postgrest-go library usage.
// For now this example is represent wanted syntax and bindings for library.
// After core development this test files will be used for CI tests.

package main

import (
	"fmt"

	"github.com/supabase/postgrest-go"
)

var (
	RestUrl = `http://localhost:3000`
	headers = map[string]string{}
	schema  = "public"
)

func main() {
	client := postgrest.NewClient(RestUrl, schema, headers)

	res, _, err := client.From("actor").Select("actor_id,first_name", "", false).ExecuteString()
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
}
