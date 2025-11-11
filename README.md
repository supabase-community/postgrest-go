# Postgrest GO

[![golangci-lint](https://github.com/supabase-community/postgrest-go/actions/workflows/golangci.yml/badge.svg)](https://github.com/supabase-community/postgrest-go/actions/workflows/golangci.yml) [![CodeFactor](https://www.codefactor.io/repository/github/supabase-community/postgrest-go/badge/main?s=101cab44de33934fd85cadcd9a9b535a05791670)](https://www.codefactor.io/repository/github/supabase-community/postgrest-go/overview/main)
[![Go Coverage](https://github.com/supabase-community/postgrest-go/wiki/coverage.svg)](https://raw.githack.com/wiki/supabase-community/postgrest-go/coverage.html)

Golang client for [PostgREST](https://postgrest.org). The goal of this library is to make an "ORM-like" restful interface.

## Documentation

Full documentation can be found [here](https://pkg.go.dev/github.com/supabase-community/postgrest-go).

## Installation

```bash
go get github.com/supabase-community/postgrest-go
```

## Quick Start

### Basic Usage

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supabase-community/postgrest-go"
)

func main() {
	// Create a new client
	client := postgrest.NewClient("http://localhost:3000/rest/v1", "public", nil)
	if client.ClientError != nil {
		panic(client.ClientError)
	}

	// Select data
	response, err := client.From("users").Select("*", nil).Execute(context.Background())
	if err != nil {
		panic(err)
	}

	if response.Error != nil {
		panic(response.Error)
	}

	// Print results
	data, _ := json.Marshal(response.Data)
	fmt.Println(string(data))
}
```

### Select with Filters

```go
// Select with filters
opts := &postgrest.SelectOptions{Count: "exact"}
response, err := client.
	From("users").
	Select("id, name, email", opts).
	Eq("status", "active").
	Limit(10, nil).
	Order("name", &postgrest.OrderOptions{Ascending: true}).
	Execute(context.Background())
```

### Insert Data

```go
// Insert a single row
insertOpts := &postgrest.InsertOptions{Count: "exact"}
response, err := client.
	From("users").
	Insert(map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}, insertOpts).
	Execute(context.Background())
```

### Update Data

```go
// Update rows
updateOpts := &postgrest.UpdateOptions{Count: "exact"}
response, err := client.
	From("users").
	Update(map[string]interface{}{
		"status": "inactive",
	}, updateOpts).
	Eq("id", 1).
	Execute(context.Background())
```

### Delete Data

```go
// Delete rows
deleteOpts := &postgrest.DeleteOptions{Count: "exact"}
response, err := client.
	From("users").
	Delete(deleteOpts).
	Eq("status", "inactive").
	Execute(context.Background())
```

### Upsert Data

```go
// Upsert (insert or update)
upsertOpts := &postgrest.UpsertOptions{
	OnConflict:      "email",
	IgnoreDuplicates: false,
	Count:           "exact",
}
response, err := client.
	From("users").
	Upsert(map[string]interface{}{
		"email": "john@example.com",
		"name":  "John Doe",
	}, upsertOpts).
	Execute(context.Background())
```

### RPC (Remote Procedure Call)

```go
// Call a PostgreSQL function
rpcOpts := &postgrest.RpcOptions{Count: "exact"}
response, err := client.
	Rpc("get_user_by_email", map[string]interface{}{
		"email": "john@example.com",
	}, rpcOpts).
	Execute(context.Background())
```

### Advanced Filtering

```go
// Multiple filters
response, err := client.
	From("users").
	Select("*", nil).
	Eq("status", "active").
	Gte("age", 18).
	Lte("age", 65).
	In("role", []interface{}{"admin", "user"}).
	Like("name", "%John%").
	Execute(context.Background())

// Text search
textSearchOpts := &postgrest.TextSearchOptions{
	Type:   "websearch",
	Config: "english",
}
response, err := client.
	From("posts").
	Select("*", nil).
	TextSearch("content", "golang tutorial", textSearchOpts).
	Execute(context.Background())
```

### Single Result

```go
// Get a single row
response, err := client.
	From("users").
	Select("*", nil).
	Eq("id", 1).
	Limit(1, nil).
	Single().
	Execute(context.Background())

// Get a single row or null (maybeSingle)
response, err := client.
	From("users").
	Select("*", nil).
	Eq("id", 999).
	MaybeSingle().
	Execute(context.Background())
```

### ExecuteTo (Unmarshal directly)

```go
// Unmarshal directly into a struct
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users []User
count, err := client.
	From("users").
	Select("*", nil).
	Eq("status", "active").
	ExecuteTo(context.Background(), &users)
```

### Schema Selection

```go
// Switch to a different schema
client = client.Schema("private")
response, err := client.From("sensitive_data").Select("*", nil).Execute(context.Background())
```

### Error Handling

```go
response, err := client.From("users").Select("*", nil).Execute(context.Background())
if err != nil {
	// Network or parsing error
	panic(err)
}

if response.Error != nil {
	// PostgREST error
	fmt.Printf("Error: %s (Code: %s)\n", response.Error.Message, response.Error.Code)
	fmt.Printf("Details: %s\n", response.Error.Details)
	fmt.Printf("Hint: %s\n", response.Error.Hint)
}
```

### Throw on Error

```go
// Throw errors instead of returning them in response
response, err := client.
	From("users").
	Select("*", nil).
	ThrowOnError().
	Execute(context.Background())
// If there's an error, it will be thrown here
```

## API Reference

### Client Methods

- `NewClient(url, schema, headers)` - Create a new client
- `From(table)` - Start a query on a table
- `Rpc(function, args, opts)` - Call a PostgreSQL function
- `Schema(schema)` - Switch to a different schema
- `SetApiKey(key)` - Set API key header
- `SetAuthToken(token)` - Set authorization token
- `ChangeSchema(schema)` - Change schema for subsequent requests

### QueryBuilder Methods

- `Select(columns, opts)` - Select columns
- `Insert(values, opts)` - Insert rows
- `Update(values, opts)` - Update rows
- `Upsert(values, opts)` - Upsert rows
- `Delete(opts)` - Delete rows

### FilterBuilder Methods

- `Eq(column, value)` - Equal
- `Neq(column, value)` - Not equal
- `Gt(column, value)` - Greater than
- `Gte(column, value)` - Greater than or equal
- `Lt(column, value)` - Less than
- `Lte(column, value)` - Less than or equal
- `Like(column, pattern)` - Case-sensitive pattern match
- `Ilike(column, pattern)` - Case-insensitive pattern match
- `Is(column, value)` - IS operator (for NULL checks)
- `In(column, values)` - IN operator
- `Contains(column, value)` - Contains (for arrays/jsonb)
- `ContainedBy(column, value)` - Contained by
- `TextSearch(column, query, opts)` - Full-text search
- `Match(query)` - Match multiple columns
- `Not(column, operator, value)` - Negate operator
- `Or(filters, opts)` - OR condition

### TransformBuilder Methods

- `Order(column, opts)` - Order results
- `Limit(count, opts)` - Limit results
- `Range(from, to, opts)` - Range results
- `Single()` - Get single result
- `MaybeSingle()` - Get single result or null
- `Select(columns)` - Select after insert/update/delete
- `CSV()` - Return CSV format
- `GeoJSON()` - Return GeoJSON format
- `Explain(opts)` - Get query plan
- `Rollback()` - Rollback transaction
- `MaxAffected(value)` - Set max affected rows

### Response Structure

```go
type PostgrestResponse[T any] struct {
	Error      *PostgrestError `json:"error,omitempty"`
	Data       T               `json:"data,omitempty"`
	Count      *int64          `json:"count,omitempty"`
	Status     int             `json:"status"`
	StatusText string          `json:"statusText"`
}
```

## Testing

### Unit Tests

Some tests are implemented to run against mocked Postgrest endpoints using `httpmock`. To run unit tests:

```bash
go test ./...
```

### Integration Tests

Integration tests use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) to spin up real PostgreSQL and PostgREST containers for testing. This ensures tests run against actual PostgREST instances.

#### Running Integration Tests

**Option 1: Using Testcontainers (Default)**

Integration tests will automatically use testcontainers to start PostgreSQL and PostgREST containers:

```bash
go test -v -run TestIntegration
```

**Option 2: Using Existing Services**

If you have a running PostgREST instance, you can use it instead:

```bash
export POSTGREST_URL=http://localhost:3000
export USE_TESTCONTAINERS=false
go test -v -run TestIntegration
```

**Option 3: Using Docker Compose**

You can also use the provided `docker-compose.yaml` to start services manually:

```bash
docker-compose up -d
export POSTGREST_URL=http://localhost:3000
export USE_TESTCONTAINERS=false
go test -v -run TestIntegration
docker-compose down
```

#### Integration Test Coverage

The integration tests cover:

- ✅ Basic SELECT queries
- ✅ Filtering (Eq, In, Like, etc.)
- ✅ Ordering and limiting
- ✅ Single and MaybeSingle results
- ✅ INSERT operations
- ✅ UPDATE operations
- ✅ DELETE operations
- ✅ RPC (Remote Procedure Calls)
- ✅ Schema switching
- ✅ Complex queries with multiple filters
- ✅ Error handling
- ✅ Relationships and joins

#### Test Database Schema

The integration tests use the schema defined in `test/00-schema.sql` and seed data from `test/01-dummy-data.sql`. These files define:

- `users` table with various data types
- `channels` and `messages` tables for relationship testing
- `kitchen_sink` table for comprehensive type testing
- `movie`, `person`, `profile` tables for foreign key testing
- PostgreSQL functions for RPC testing
- Multiple schemas (`public` and `personal`) for schema switching tests

## Package made possible through the efforts of: 
<a href="https://github.com/supabase-community/postgrest-go/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=supabase-community/postgrest-go" />
</a>

Made with [contrib.rocks](https://contrib.rocks).

## License

This repo is licensed under the [Apache License](LICENSE).

## Sponsors

We are building the features of Firebase using enterprise-grade, open source products. We support existing communities wherever possible, and if the products don't exist we build them and open source them ourselves. Thanks to these sponsors who are making the OSS ecosystem better for everyone.

[![New Sponsor](https://user-images.githubusercontent.com/10214025/90518111-e74bbb00-e198-11ea-8f88-c9e3c1aa4b5b.png)](https://github.com/sponsors/supabase)
