package postgrest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// QueryBuilder describes a builder for a query.
type QueryBuilder struct {
	client    *Client
	method    string
	body      []byte
	tableName string
	headers   map[string]string
	params    map[string]string
	err       error
}

// ExecuteString runs the PostgREST query, returning the result as a JSON
// string.
func (q *QueryBuilder) ExecuteString() (string, int64, error) {
	return executeString(context.Background(), q.client, q.method, q.body, []string{q.tableName}, q.headers, q.params, q.err)
}

// ExecuteStringWithContext runs the PostgREST query, returning the result as
// a JSON string.
func (q *QueryBuilder) ExecuteStringWithContext(ctx context.Context) (string, int64, error) {
	return executeString(ctx, q.client, q.method, q.body, []string{q.tableName}, q.headers, q.params, q.err)
}

// Execute runs the Postgrest query, returning the result as a byte slice.
func (q *QueryBuilder) Execute() ([]byte, int64, error) {
	return execute(context.Background(), q.client, q.method, q.body, []string{q.tableName}, q.headers, q.params, q.err)
}

// ExecuteWithContext runs the PostgREST query with the given context,
// returning the result as a byte slice.
func (q *QueryBuilder) ExecuteWithContext(ctx context.Context) ([]byte, int64, error) {
	return execute(ctx, q.client, q.method, q.body, []string{q.tableName}, q.headers, q.params, q.err)
}

// ExecuteTo runs the PostgREST query, encoding the result to the supplied
// interface. Note that the argument for the to parameter should always be a
// reference to a slice unless the filter method Single is specified.
func (q *QueryBuilder) ExecuteTo(to interface{}) (int64, error) {
	return executeTo(context.Background(), q.client, q.method, q.body, to, []string{q.tableName}, q.headers, q.params, q.err)
}

// ExecuteToWithContext runs the PostgREST query with the given context,
// encoding the result to the supplied interface. Note that the argument for
// the to parameter should always be a reference to a slice unless the filter method Single is specified.
func (q *QueryBuilder) ExecuteToWithContext(ctx context.Context, to interface{}) (int64, error) {
	return executeTo(ctx, q.client, q.method, q.body, to, []string{q.tableName}, q.headers, q.params, q.err)
}

// Select performs vertical filtering.
func (q *QueryBuilder) Select(columns, count string, head bool) *FilterBuilder {
	if head {
		q.method = "HEAD"
	} else {
		q.method = "GET"
	}

	if columns == "" {
		q.params["select"] = "*"
	} else {
		quoted := false
		var resultArr []string
		for _, char := range strings.Split(columns, "") {
			if char == `"` {
				quoted = !quoted
			}
			if char == " " {
				char = ""
			}
			resultArr = append(resultArr, char)
		}
		result := strings.Join(resultArr, "")
		q.params["select"] = result
	}

	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		currentValue, ok := q.headers["Prefer"]
		if ok && currentValue != "" {
			q.headers["Prefer"] = fmt.Sprintf("%s,count=%s", currentValue, count)
		} else {
			q.headers["Prefer"] = fmt.Sprintf("count=%s", count)
		}
	}
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params, err: q.err}
}

// Insert performs an insertion into the table.
func (q *QueryBuilder) Insert(value interface{}, upsert bool, onConflict, returning, count string) *FilterBuilder {
	q.method = "POST"

	if onConflict != "" && upsert {
		q.params["on_conflict"] = onConflict
	}

	var headerList []string
	if upsert {
		headerList = append(headerList, "resolution=merge-duplicates")
	}
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")

	// Get body if exist
	var byteBody []byte = nil
	if value != nil {
		jsonBody, err := json.Marshal(value)
		if err != nil {
			q.client.ClientError = err
			return &FilterBuilder{err: errors.Join(q.err, err)}
		}
		byteBody = jsonBody
	}
	q.body = byteBody
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params, err: q.err}
}

// Upsert performs an upsert into the table.
func (q *QueryBuilder) Upsert(value interface{}, onConflict, returning, count string) *FilterBuilder {
	q.method = "POST"

	if onConflict != "" {
		q.params["on_conflict"] = onConflict
	}

	headerList := []string{"resolution=merge-duplicates"}
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")

	// Get body if exist
	var byteBody []byte = nil
	if value != nil {
		jsonBody, err := json.Marshal(value)
		if err != nil {
			q.client.ClientError = err
			return &FilterBuilder{err: errors.Join(q.err, err)}
		}
		byteBody = jsonBody
	}
	q.body = byteBody
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params, err: q.err}
}

// Delete performs a deletion from the table.
func (q *QueryBuilder) Delete(returning, count string) *FilterBuilder {
	q.method = "DELETE"

	var headerList []string
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params, err: q.err}
}

// Update performs an update on the table.
func (q *QueryBuilder) Update(value interface{}, returning, count string) *FilterBuilder {
	q.method = "PATCH"

	var headerList []string
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")

	// Get body if it exists
	var byteBody []byte = nil
	if value != nil {
		jsonBody, err := json.Marshal(value)
		if err != nil {
			q.client.ClientError = err
			return &FilterBuilder{err: errors.Join(q.err, err)}
		}
		byteBody = jsonBody
	}
	q.body = byteBody
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params, err: q.err}
}
