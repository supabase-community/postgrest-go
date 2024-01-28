# Postgrest GO

[![golangci-lint](https://github.com/supabase-community/postgrest-go/actions/workflows/golangci.yml/badge.svg)](https://github.com/supabase-community/postgrest-go/actions/workflows/golangci.yml) [![CodeFactor](https://www.codefactor.io/repository/github/supabase-community/postgrest-go/badge/main?s=101cab44de33934fd85cadcd9a9b535a05791670)](https://www.codefactor.io/repository/github/supabase-community/postgrest-go/overview/main)

Golang client for [PostgREST](https://postgrest.org). The goal of this library is to make an "ORM-like" restful interface.

## Documentation

Full documentation can be found [here](https://pkg.go.dev/github.com/supabase-community/postgrest-go).

### Quick start

Install

```bash
go get github.com/supabase-community/postgrest-go
```

Usage

```go
package main

import (
	"fmt"

	"github.com/supabase-community/postgrest-go"
)

func main() {
	client := postgrest.NewClient("http://localhost:3000/rest/v1", "", nil)
	if client.ClientError != nil {
		panic(client.ClientError)
	}

	result := client.Rpc("add_them", "", map[string]int{"a": 12, "b": 3})
	if client.ClientError != nil {
		panic(client.ClientError)
	}

	fmt.Println(result)
}
```

- select(): https://supabase.com/docs/reference/javascript/select
- insert(): https://supabase.com/docs/reference/javascript/insert
- update(): https://supabase.com/docs/reference/javascript/update
- upsert(): https://supabase.com/docs/reference/javascript/upsert
- delete(): https://supabase.com/docs/reference/javascript/delete

## Testing

Some tests are implemented to run against mocked Postgrest endpoints. Optionally, tests can be run against an actual Postgrest instance by setting a `POSTGREST_URL` environment variable to the fully-qualified URL to a Postgrest instance, and, optionally, an `API_KEY` environment variable (if, for example, testing against a local Supabase instance).

A [script](test/seed.sql) is included in the test directory that can be used to seed the test database.

To run all tests:

```bash
go test ./...
```

## License

This repo is licensed under the [Apache License](LICENSE).

## Sponsors

We are building the features of Firebase using enterprise-grade, open source products. We support existing communities wherever possible, and if the products don’t exist we build them and open source them ourselves. Thanks to these sponsors who are making the OSS ecosystem better for everyone.

[![New Sponsor](https://user-images.githubusercontent.com/10214025/90518111-e74bbb00-e198-11ea-8f88-c9e3c1aa4b5b.png)](https://github.com/sponsors/supabase)
