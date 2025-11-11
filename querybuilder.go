package postgrest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// QueryBuilder provides query building methods
// Similar to PostgrestQueryBuilder in postgrest-js
type QueryBuilder[T any] struct {
	url     *url.URL
	headers http.Header
	schema  string
	client  *Client
}

// NewQueryBuilder creates a new QueryBuilder instance
func NewQueryBuilder[T any](client *Client, relation string) *QueryBuilder[T] {
	baseURL := client.Transport.baseURL
	queryURL := baseURL.JoinPath(relation)

	headers := make(http.Header)
	if client.Transport != nil {
		client.Transport.mu.RLock()
		for key, values := range client.Transport.header {
			for _, val := range values {
				headers.Add(key, val)
			}
		}
		client.Transport.mu.RUnlock()
	}

	return &QueryBuilder[T]{
		url:     queryURL,
		headers: headers,
		schema:  client.schemaName,
		client:  client,
	}
}

// SelectOptions contains options for Select
type SelectOptions struct {
	Head  bool
	Count string // "exact", "planned", or "estimated"
}

// Select performs a SELECT query on the table or view
func (q *QueryBuilder[T]) Select(columns string, opts *SelectOptions) *FilterBuilder[[]T] {
	if opts == nil {
		opts = &SelectOptions{}
	}

	method := "GET"
	if opts.Head {
		method = "HEAD"
	}

	// Remove whitespaces except when quoted
	quoted := false
	var cleanedColumns strings.Builder
	if columns == "" {
		cleanedColumns.WriteString("*")
	} else {
		for _, char := range columns {
			if char == '"' {
				quoted = !quoted
			}
			if char == ' ' && !quoted {
				continue
			}
			cleanedColumns.WriteRune(char)
		}
	}

	query := q.url.Query()
	query.Set("select", cleanedColumns.String())
	q.url.RawQuery = query.Encode()

	if opts.Count != "" && (opts.Count == "exact" || opts.Count == "planned" || opts.Count == "estimated") {
		q.headers.Add("Prefer", fmt.Sprintf("count=%s", opts.Count))
	}

	builder := NewBuilder[[]T](q.client, method, q.url, &BuilderOptions{
		Headers: q.headers,
		Schema:  q.schema,
	})

	return &FilterBuilder[[]T]{Builder: builder}
}

// InsertOptions contains options for Insert
type InsertOptions struct {
	Count         string // "exact", "planned", or "estimated"
	DefaultToNull bool
}

// Insert performs an INSERT into the table or view
func (q *QueryBuilder[T]) Insert(values interface{}, opts *InsertOptions) *FilterBuilder[interface{}] {
	if opts == nil {
		opts = &InsertOptions{DefaultToNull: true}
	}

	method := "POST"

	headers := make(http.Header)
	for key, values := range q.headers {
		for _, val := range values {
			headers.Add(key, val)
		}
	}

	if opts.Count != "" && (opts.Count == "exact" || opts.Count == "planned" || opts.Count == "estimated") {
		headers.Add("Prefer", fmt.Sprintf("count=%s", opts.Count))
	}
	if !opts.DefaultToNull {
		headers.Add("Prefer", "missing=default")
	}

	// Handle array values to set columns parameter
	valuesBytes, _ := json.Marshal(values)
	var valuesArray []map[string]interface{}
	if json.Unmarshal(valuesBytes, &valuesArray) == nil && len(valuesArray) > 0 {
		columns := make(map[string]bool)
		for _, row := range valuesArray {
			for key := range row {
				columns[key] = true
			}
		}
		var uniqueColumns []string
		for col := range columns {
			uniqueColumns = append(uniqueColumns, fmt.Sprintf(`"%s"`, col))
		}
		if len(uniqueColumns) > 0 {
			query := q.url.Query()
			query.Set("columns", strings.Join(uniqueColumns, ","))
			q.url.RawQuery = query.Encode()
		}
	}

	builder := NewBuilder[interface{}](q.client, method, q.url, &BuilderOptions{
		Headers: headers,
		Schema:  q.schema,
		Body:    values,
	})

	return &FilterBuilder[interface{}]{Builder: builder}
}

// UpsertOptions contains options for Upsert
type UpsertOptions struct {
	OnConflict       string
	IgnoreDuplicates bool
	Count            string // "exact", "planned", or "estimated"
	DefaultToNull    bool
}

// Upsert performs an UPSERT on the table or view
func (q *QueryBuilder[T]) Upsert(values interface{}, opts *UpsertOptions) *FilterBuilder[interface{}] {
	if opts == nil {
		opts = &UpsertOptions{IgnoreDuplicates: false, DefaultToNull: true}
	}

	method := "POST"

	headers := make(http.Header)
	for key, values := range q.headers {
		for _, val := range values {
			headers.Add(key, val)
		}
	}

	resolution := "merge-duplicates"
	if opts.IgnoreDuplicates {
		resolution = "ignore-duplicates"
	}
	headers.Add("Prefer", fmt.Sprintf("resolution=%s", resolution))

	if opts.OnConflict != "" {
		query := q.url.Query()
		query.Set("on_conflict", opts.OnConflict)
		q.url.RawQuery = query.Encode()
	}
	if opts.Count != "" && (opts.Count == "exact" || opts.Count == "planned" || opts.Count == "estimated") {
		headers.Add("Prefer", fmt.Sprintf("count=%s", opts.Count))
	}
	if !opts.DefaultToNull {
		headers.Add("Prefer", "missing=default")
	}

	// Handle array values to set columns parameter
	valuesBytes, _ := json.Marshal(values)
	var valuesArray []map[string]interface{}
	if json.Unmarshal(valuesBytes, &valuesArray) == nil && len(valuesArray) > 0 {
		columns := make(map[string]bool)
		for _, row := range valuesArray {
			for key := range row {
				columns[key] = true
			}
		}
		var uniqueColumns []string
		for col := range columns {
			uniqueColumns = append(uniqueColumns, fmt.Sprintf(`"%s"`, col))
		}
		if len(uniqueColumns) > 0 {
			query := q.url.Query()
			query.Set("columns", strings.Join(uniqueColumns, ","))
			q.url.RawQuery = query.Encode()
		}
	}

	builder := NewBuilder[interface{}](q.client, method, q.url, &BuilderOptions{
		Headers: headers,
		Schema:  q.schema,
		Body:    values,
	})

	return &FilterBuilder[interface{}]{Builder: builder}
}

// UpdateOptions contains options for Update
type UpdateOptions struct {
	Count string // "exact", "planned", or "estimated"
}

// Update performs an UPDATE on the table or view
func (q *QueryBuilder[T]) Update(values interface{}, opts *UpdateOptions) *FilterBuilder[interface{}] {
	if opts == nil {
		opts = &UpdateOptions{}
	}

	method := "PATCH"

	headers := make(http.Header)
	for key, values := range q.headers {
		for _, val := range values {
			headers.Add(key, val)
		}
	}

	if opts.Count != "" && (opts.Count == "exact" || opts.Count == "planned" || opts.Count == "estimated") {
		headers.Add("Prefer", fmt.Sprintf("count=%s", opts.Count))
	}

	builder := NewBuilder[interface{}](q.client, method, q.url, &BuilderOptions{
		Headers: headers,
		Schema:  q.schema,
		Body:    values,
	})

	return &FilterBuilder[interface{}]{Builder: builder}
}

// DeleteOptions contains options for Delete
type DeleteOptions struct {
	Count string // "exact", "planned", or "estimated"
}

// Delete performs a DELETE on the table or view
func (q *QueryBuilder[T]) Delete(opts *DeleteOptions) *FilterBuilder[interface{}] {
	if opts == nil {
		opts = &DeleteOptions{}
	}

	method := "DELETE"

	headers := make(http.Header)
	for key, values := range q.headers {
		for _, val := range values {
			headers.Add(key, val)
		}
	}

	if opts.Count != "" && (opts.Count == "exact" || opts.Count == "planned" || opts.Count == "estimated") {
		headers.Add("Prefer", fmt.Sprintf("count=%s", opts.Count))
	}

	builder := NewBuilder[interface{}](q.client, method, q.url, &BuilderOptions{
		Headers: headers,
		Schema:  q.schema,
	})

	return &FilterBuilder[interface{}]{Builder: builder}
}
