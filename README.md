# Postgrest GO

[![golangci-lint](https://github.com/supabase/postgrest-go/actions/workflows/golangci.yml/badge.svg)](https://github.com/supabase/postgrest-go/actions/workflows/golangci.yml) [![CodeFactor](https://www.codefactor.io/repository/github/supabase/postgrest-go/badge/main?s=101cab44de33934fd85cadcd9a9b535a05791670)](https://www.codefactor.io/repository/github/supabase/postgrest-go/overview/main)

Golang client for [PostgREST](https://postgrest.org). The goal of this library is to make an "ORM-like" restful interface.

## Documentation

Full documentation can be found on our [website](https://supabase.io/docs/postgrest/client/postgrest-client).

### Quick start

Install

```bash
go get github.com/supabase/postgrest-go
```

Usage

```go
package main
​
import (
"fmt"
​
"github.com/supabase/postgrest-go"
)
​
func main() {
	client := postgrest.NewClient("http://localhost:3000", "", nil)
	if client.ClientError != nil {
		panic(client.ClientError)
	}
	​
	result := client.Rpc("add_them", "", map[string]int{"a": 12, "b": 3})
	if client.ClientError != nil {
		panic(client.ClientError)
	}
	​
	fmt.Println(result)
}
```

- select(): https://supabase.io/docs/postgrest/client/select
- insert(): https://supabase.io/docs/postgrest/client/insert
- update(): https://supabase.io/docs/postgrest/client/update
- delete(): https://supabase.io/docs/postgrest/client/delete

## License

This repo is liscenced under Apache License.

## Sponsors

We are building the features of Firebase using enterprise-grade, open source products. We support existing communities wherever possible, and if the products don’t exist we build them and open source them ourselves. Thanks to these sponsors who are making the OSS ecosystem better for everyone.

[![New Sponsor](https://user-images.githubusercontent.com/10214025/90518111-e74bbb00-e198-11ea-8f88-c9e3c1aa4b5b.png)](https://github.com/sponsors/supabase)

![Watch this repo](https://gitcdn.xyz/repo/supabase/monorepo/master/web/static/watch-repo.gif "Watch this repo")
