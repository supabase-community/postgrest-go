package main

import (
	"fmt"

	"github.com/muratmirgun/postgrest-go"
)

var (
	REST_URL = `http://localhost:3000`
)

func main() {
	client := postgrest.NewClient(REST_URL, "", nil)
	if client.ClientError != nil {
		panic(client.ClientError)
	}

	result := client.Rpc("add_them", "", map[string]int{"a": 9, "b": 3})
	if client.ClientError != nil {
		panic(client.ClientError)
	}

	fmt.Println(result)
}
