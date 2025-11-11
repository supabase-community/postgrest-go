package postgrest

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// FilterBuilder provides filtering methods for queries
// Similar to PostgrestFilterBuilder in postgrest-js
type FilterBuilder[T any] struct {
	*Builder[T]
}

var filterOperators = []string{"eq", "neq", "gt", "gte", "lt", "lte", "like", "ilike", "is", "in", "cs", "cd", "sl", "sr", "nxl", "nxr", "adj", "ov", "fts", "plfts", "phfts", "wfts"}

// appendFilter is a helper method that appends a filter to existing filters on a column
func (f *FilterBuilder[T]) appendFilter(column, filterValue string) *FilterBuilder[T] {
	query := f.url.Query()
	existing := query.Get(column)
	andValue := query.Get("and")

	// Check if there's already an 'and' param that contains filters for this column
	columnPrefix := column + "."
	if andValue != "" && strings.Contains(andValue, columnPrefix) {
		// Append to existing 'and' param
		andValue = strings.TrimSuffix(andValue, ")") + "," + column + "." + filterValue + ")"
		query.Set("and", andValue)
	} else if existing != "" {
		// If a filter already exists for this column, combine with 'and'
		if andValue != "" {
			andValue = strings.TrimSuffix(andValue, ")") + "," + column + "." + filterValue + ")"
		} else {
			andValue = fmt.Sprintf("(%s.%s,%s.%s)", column, existing, column, filterValue)
		}
		query.Set("and", andValue)
		query.Del(column)
	} else {
		query.Set(column, filterValue)
	}
	f.url.RawQuery = query.Encode()
	return f
}

func isOperator(value string) bool {
	for _, op := range filterOperators {
		if op == value {
			return true
		}
	}
	return false
}

// Filter adds a filtering operator to the query
func (f *FilterBuilder[T]) Filter(column, operator, value string) *FilterBuilder[T] {
	if !isOperator(operator) {
		return f
	}
	return f.appendFilter(column, fmt.Sprintf("%s.%s", operator, value))
}

// Eq matches only rows where column is equal to value
func (f *FilterBuilder[T]) Eq(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("eq.%v", value))
}

// Neq matches only rows where column is not equal to value
func (f *FilterBuilder[T]) Neq(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("neq.%v", value))
}

// Gt matches only rows where column is greater than value
func (f *FilterBuilder[T]) Gt(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("gt.%v", value))
}

// Gte matches only rows where column is greater than or equal to value
func (f *FilterBuilder[T]) Gte(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("gte.%v", value))
}

// Lt matches only rows where column is less than value
func (f *FilterBuilder[T]) Lt(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("lt.%v", value))
}

// Lte matches only rows where column is less than or equal to value
func (f *FilterBuilder[T]) Lte(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("lte.%v", value))
}

// Like matches only rows where column matches pattern case-sensitively
func (f *FilterBuilder[T]) Like(column, pattern string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("like.%s", pattern))
}

// LikeAllOf matches only rows where column matches all of patterns case-sensitively
func (f *FilterBuilder[T]) LikeAllOf(column string, patterns []string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("like(all).{%s}", strings.Join(patterns, ",")))
}

// LikeAnyOf matches only rows where column matches any of patterns case-sensitively
func (f *FilterBuilder[T]) LikeAnyOf(column string, patterns []string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("like(any).{%s}", strings.Join(patterns, ",")))
}

// Ilike matches only rows where column matches pattern case-insensitively
func (f *FilterBuilder[T]) Ilike(column, pattern string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("ilike.%s", pattern))
}

// IlikeAllOf matches only rows where column matches all of patterns case-insensitively
func (f *FilterBuilder[T]) IlikeAllOf(column string, patterns []string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("ilike(all).{%s}", strings.Join(patterns, ",")))
}

// IlikeAnyOf matches only rows where column matches any of patterns case-insensitively
func (f *FilterBuilder[T]) IlikeAnyOf(column string, patterns []string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("ilike(any).{%s}", strings.Join(patterns, ",")))
}

// Is matches only rows where column IS value
func (f *FilterBuilder[T]) Is(column string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("is.%v", value))
}

// In matches only rows where column is included in the values array
func (f *FilterBuilder[T]) In(column string, values []interface{}) *FilterBuilder[T] {
	postgrestReservedCharsRegexp := regexp.MustCompile(`[,()]`)
	var cleanedValues []string
	for _, v := range values {
		valStr := fmt.Sprintf("%v", v)
		if postgrestReservedCharsRegexp.MatchString(valStr) {
			cleanedValues = append(cleanedValues, fmt.Sprintf(`"%s"`, valStr))
		} else {
			cleanedValues = append(cleanedValues, valStr)
		}
	}
	return f.appendFilter(column, fmt.Sprintf("in.(%s)", strings.Join(cleanedValues, ",")))
}

// Contains matches only rows where column contains every element appearing in value
func (f *FilterBuilder[T]) Contains(column string, value interface{}) *FilterBuilder[T] {
	switch v := value.(type) {
	case string:
		// range types
		return f.appendFilter(column, fmt.Sprintf("cs.%s", v))
	case []interface{}:
		// array
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, fmt.Sprintf("%v", item))
		}
		return f.appendFilter(column, fmt.Sprintf("cs.{%s}", strings.Join(strValues, ",")))
	default:
		// json
		jsonBytes, _ := json.Marshal(value)
		return f.appendFilter(column, fmt.Sprintf("cs.%s", string(jsonBytes)))
	}
}

// ContainedBy matches only rows where every element appearing in column is contained by value
func (f *FilterBuilder[T]) ContainedBy(column string, value interface{}) *FilterBuilder[T] {
	switch v := value.(type) {
	case string:
		// range types
		return f.appendFilter(column, fmt.Sprintf("cd.%s", v))
	case []interface{}:
		// array
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, fmt.Sprintf("%v", item))
		}
		return f.appendFilter(column, fmt.Sprintf("cd.{%s}", strings.Join(strValues, ",")))
	default:
		// json
		jsonBytes, _ := json.Marshal(value)
		return f.appendFilter(column, fmt.Sprintf("cd.%s", string(jsonBytes)))
	}
}

// RangeGt matches only rows where every element in column is greater than any element in range
func (f *FilterBuilder[T]) RangeGt(column, rangeValue string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("sr.%s", rangeValue))
}

// RangeGte matches only rows where every element in column is either contained in range or greater than any element in range
func (f *FilterBuilder[T]) RangeGte(column, rangeValue string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("nxl.%s", rangeValue))
}

// RangeLt matches only rows where every element in column is less than any element in range
func (f *FilterBuilder[T]) RangeLt(column, rangeValue string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("sl.%s", rangeValue))
}

// RangeLte matches only rows where every element in column is either contained in range or less than any element in range
func (f *FilterBuilder[T]) RangeLte(column, rangeValue string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("nxr.%s", rangeValue))
}

// RangeAdjacent matches only rows where column is mutually exclusive to range
func (f *FilterBuilder[T]) RangeAdjacent(column, rangeValue string) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("adj.%s", rangeValue))
}

// Overlaps matches only rows where column and value have an element in common
func (f *FilterBuilder[T]) Overlaps(column string, value interface{}) *FilterBuilder[T] {
	switch v := value.(type) {
	case string:
		// range
		return f.appendFilter(column, fmt.Sprintf("ov.%s", v))
	case []interface{}:
		// array
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, fmt.Sprintf("%v", item))
		}
		return f.appendFilter(column, fmt.Sprintf("ov.{%s}", strings.Join(strValues, ",")))
	default:
		return f
	}
}

// TextSearchOptions contains options for text search
type TextSearchOptions struct {
	Config string
	Type   string // "plain", "phrase", or "websearch"
}

// TextSearch matches only rows where column matches the query string
func (f *FilterBuilder[T]) TextSearch(column, query string, opts *TextSearchOptions) *FilterBuilder[T] {
	var typePart string
	if opts != nil {
		switch opts.Type {
		case "plain":
			typePart = "pl"
		case "phrase":
			typePart = "ph"
		case "websearch":
			typePart = "w"
		}
	}

	configPart := ""
	if opts != nil && opts.Config != "" {
		configPart = fmt.Sprintf("(%s)", opts.Config)
	}

	return f.appendFilter(column, fmt.Sprintf("%sfts%s.%s", typePart, configPart, query))
}

// Match matches only rows where each column in query keys is equal to its associated value
func (f *FilterBuilder[T]) Match(query map[string]interface{}) *FilterBuilder[T] {
	for column, value := range query {
		f.appendFilter(column, fmt.Sprintf("eq.%v", value))
	}
	return f
}

// Not matches only rows which doesn't satisfy the filter
func (f *FilterBuilder[T]) Not(column, operator string, value interface{}) *FilterBuilder[T] {
	return f.appendFilter(column, fmt.Sprintf("not.%s.%v", operator, value))
}

// OrOptions contains options for Or
type OrOptions struct {
	ReferencedTable string
	// Deprecated: Use ReferencedTable instead
	ForeignTable string
}

// Or matches only rows which satisfy at least one of the filters
func (f *FilterBuilder[T]) Or(filters string, opts *OrOptions) *FilterBuilder[T] {
	if opts == nil {
		opts = &OrOptions{}
	}

	key := "or"
	if opts.ReferencedTable != "" {
		key = opts.ReferencedTable + ".or"
	}

	query := f.url.Query()
	query.Set(key, fmt.Sprintf("(%s)", filters))
	f.url.RawQuery = query.Encode()
	return f
}

// Embed TransformBuilder methods
func (f *FilterBuilder[T]) Select(columns string) *FilterBuilder[T] {
	tb := &TransformBuilder[T]{Builder: f.Builder}
	return tb.Select(columns)
}

func (f *FilterBuilder[T]) Order(column string, opts *OrderOptions) *TransformBuilder[T] {
	tb := &TransformBuilder[T]{Builder: f.Builder}
	return tb.Order(column, opts)
}

func (f *FilterBuilder[T]) Limit(count int, opts *LimitOptions) *TransformBuilder[T] {
	tb := &TransformBuilder[T]{Builder: f.Builder}
	return tb.Limit(count, opts)
}

func (f *FilterBuilder[T]) Range(from, to int, opts *RangeOptions) *TransformBuilder[T] {
	tb := &TransformBuilder[T]{Builder: f.Builder}
	return tb.Range(from, to, opts)
}

func (f *FilterBuilder[T]) Single() *Builder[T] {
	tb := &TransformBuilder[T]{Builder: f.Builder}
	return tb.Single()
}

func (f *FilterBuilder[T]) MaybeSingle() *Builder[T] {
	tb := &TransformBuilder[T]{Builder: f.Builder}
	return tb.MaybeSingle()
}

// Execute executes the query and returns the response
func (f *FilterBuilder[T]) Execute(ctx context.Context) (*PostgrestResponse[T], error) {
	return f.Builder.Execute(ctx)
}

// ExecuteTo executes the query and unmarshals the result into the provided interface
func (f *FilterBuilder[T]) ExecuteTo(ctx context.Context, to interface{}) (*int64, error) {
	return f.Builder.ExecuteTo(ctx, to)
}
