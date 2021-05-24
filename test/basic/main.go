// This is basic example for postgrest-go library usage.
// For now this example is represent wanted syntax and bindings for library.
// After core development this test files will be used for CI tests.

package main

// import "fmt"

// var (
// 	REST_URL = `http://localhost:3000`
// 	headers  = map[string]string{
// 		"apikey": "postgrest-go",
// 	}
// 	schema = "public"
// )

// type DataModel struct {
// 	ID   int
// 	Task string
// 	Done bool
// }

// func main() {
// 	client := postgrest.NewClient(REST_URL, headers, schema)

// 	var data = []DataModel{}
// 	err := client.From("todos").Select("task").Eq("task", "finish writing tests").Unmarshal(&data)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println(data)
// }
