package postgrest

import (
	"fmt"
	"strconv"
	"strings"
)

// TransformBuilder provides transformation methods for queries
// Similar to PostgrestTransformBuilder in postgrest-js
type TransformBuilder[T any] struct {
	*Builder[T]
}

// Select performs a SELECT on the query result
func (t *TransformBuilder[T]) Select(columns string) *FilterBuilder[T] {
	// Remove whitespaces except when quoted
	quoted := false
	var cleanedColumns strings.Builder
	for _, char := range columns {
		if char == '"' {
			quoted = !quoted
		}
		if char == ' ' && !quoted {
			continue
		}
		cleanedColumns.WriteRune(char)
	}

	cleaned := cleanedColumns.String()
	if cleaned == "" {
		cleaned = "*"
	}

	query := t.url.Query()
	query.Set("select", cleaned)
	t.url.RawQuery = query.Encode()
	t.headers.Add("Prefer", "return=representation")

	return &FilterBuilder[T]{Builder: t.Builder}
}

// Order orders the query result by column
func (t *TransformBuilder[T]) Order(column string, opts *OrderOptions) *TransformBuilder[T] {
	if opts == nil {
		opts = &OrderOptions{Ascending: true}
	}

	key := "order"
	if opts.ReferencedTable != "" {
		key = opts.ReferencedTable + ".order"
	}

	ascendingStr := "desc"
	if opts.Ascending {
		ascendingStr = "asc"
	}

	nullsStr := ""
	if opts.NullsFirst != nil {
		if *opts.NullsFirst {
			nullsStr = ".nullsfirst"
		} else {
			nullsStr = ".nullslast"
		}
	}

	query := t.url.Query()
	existingOrder := query.Get(key)
	orderValue := fmt.Sprintf("%s.%s%s", column, ascendingStr, nullsStr)
	if existingOrder != "" {
		orderValue = existingOrder + "," + orderValue
	}

	query.Set(key, orderValue)
	t.url.RawQuery = query.Encode()
	return t
}

// OrderOptions contains options for ordering
type OrderOptions struct {
	Ascending       bool
	NullsFirst      *bool
	ReferencedTable string
	// Deprecated: Use ReferencedTable instead
	ForeignTable string
}

// Limit limits the query result by count
func (t *TransformBuilder[T]) Limit(count int, opts *LimitOptions) *TransformBuilder[T] {
	if opts == nil {
		opts = &LimitOptions{}
	}

	key := "limit"
	if opts.ReferencedTable != "" {
		key = opts.ReferencedTable + ".limit"
	}

	query := t.url.Query()
	query.Set(key, strconv.Itoa(count))
	t.url.RawQuery = query.Encode()
	return t
}

// LimitOptions contains options for limiting
type LimitOptions struct {
	ReferencedTable string
	// Deprecated: Use ReferencedTable instead
	ForeignTable string
}

// Range limits the query result by starting at an offset from and ending at to
func (t *TransformBuilder[T]) Range(from, to int, opts *RangeOptions) *TransformBuilder[T] {
	if opts == nil {
		opts = &RangeOptions{}
	}

	offsetKey := "offset"
	limitKey := "limit"
	if opts.ReferencedTable != "" {
		offsetKey = opts.ReferencedTable + ".offset"
		limitKey = opts.ReferencedTable + ".limit"
	}

	query := t.url.Query()
	query.Set(offsetKey, strconv.Itoa(from))
	// Range is inclusive, so add 1
	query.Set(limitKey, strconv.Itoa(to-from+1))
	t.url.RawQuery = query.Encode()
	return t
}

// RangeOptions contains options for range
type RangeOptions struct {
	ReferencedTable string
	// Deprecated: Use ReferencedTable instead
	ForeignTable string
}

// AbortSignal sets the AbortSignal for the fetch request
func (t *TransformBuilder[T]) AbortSignal(ctx interface{}) *TransformBuilder[T] {
	// In Go, we use context.Context instead of AbortSignal
	// This is a placeholder for API compatibility
	return t
}

// Single returns data as a single object instead of an array
func (t *TransformBuilder[T]) Single() *Builder[T] {
	t.headers.Set("Accept", "application/vnd.pgrst.object+json")
	return t.Builder
}

// MaybeSingle returns data as a single object or null
func (t *TransformBuilder[T]) MaybeSingle() *Builder[T] {
	if t.method == "GET" {
		t.headers.Set("Accept", "application/json")
	} else {
		t.headers.Set("Accept", "application/vnd.pgrst.object+json")
	}
	t.isMaybeSingle = true
	return t.Builder
}

// CSV returns data as a string in CSV format
func (t *TransformBuilder[T]) CSV() *Builder[string] {
	t.headers.Set("Accept", "text/csv")
	return &Builder[string]{
		method:             t.method,
		url:                t.url,
		headers:            t.headers,
		schema:             t.schema,
		body:               t.body,
		shouldThrowOnError: t.shouldThrowOnError,
		signal:             t.signal,
		client:             t.client,
		isMaybeSingle:      t.isMaybeSingle,
	}
}

// GeoJSON returns data as an object in GeoJSON format
func (t *TransformBuilder[T]) GeoJSON() *Builder[map[string]interface{}] {
	t.headers.Set("Accept", "application/geo+json")
	return &Builder[map[string]interface{}]{
		method:             t.method,
		url:                t.url,
		headers:            t.headers,
		schema:             t.schema,
		body:               t.body,
		shouldThrowOnError: t.shouldThrowOnError,
		signal:             t.signal,
		client:             t.client,
		isMaybeSingle:      t.isMaybeSingle,
	}
}

// ExplainOptions contains options for explain
type ExplainOptions struct {
	Analyze  bool
	Verbose  bool
	Settings bool
	Buffers  bool
	WAL      bool
	Format   string // "text" or "json"
}

// Explain returns data as the EXPLAIN plan for the query
func (t *TransformBuilder[T]) Explain(opts *ExplainOptions) *Builder[interface{}] {
	if opts == nil {
		opts = &ExplainOptions{Format: "text"}
	}

	var options []string
	if opts.Analyze {
		options = append(options, "analyze")
	}
	if opts.Verbose {
		options = append(options, "verbose")
	}
	if opts.Settings {
		options = append(options, "settings")
	}
	if opts.Buffers {
		options = append(options, "buffers")
	}
	if opts.WAL {
		options = append(options, "wal")
	}

	optionsStr := strings.Join(options, "|")

	forMediatype := t.headers.Get("Accept")
	if forMediatype == "" {
		forMediatype = "application/json"
	}

	acceptValue := fmt.Sprintf("application/vnd.pgrst.plan+%s; for=\"%s\"; options=%s;", opts.Format, forMediatype, optionsStr)
	t.headers.Set("Accept", acceptValue)

	return &Builder[interface{}]{
		method:             t.method,
		url:                t.url,
		headers:            t.headers,
		schema:             t.schema,
		body:               t.body,
		shouldThrowOnError: t.shouldThrowOnError,
		signal:             t.signal,
		client:             t.client,
		isMaybeSingle:      t.isMaybeSingle,
	}
}

// Rollback rolls back the query
func (t *TransformBuilder[T]) Rollback() *TransformBuilder[T] {
	t.headers.Add("Prefer", "tx=rollback")
	return t
}

// MaxAffected sets the maximum number of rows that can be affected by the query
func (t *TransformBuilder[T]) MaxAffected(value int) *TransformBuilder[T] {
	t.headers.Add("Prefer", "handling=strict")
	t.headers.Add("Prefer", fmt.Sprintf("max-affected=%d", value))
	return t
}
