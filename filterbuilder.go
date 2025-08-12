package postgrest

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FilterBuilder describes a builder for a filtered result set.
type FilterBuilder struct {
	client    *Client
	method    string // One of "HEAD", "GET", "POST", "PUT", "DELETE"
	body      []byte
	tableName string
	headers   map[string]string
	params    map[string]string
}

// ExecuteString runs the PostgREST query, returning the result as a JSON
// string.
func (f *FilterBuilder) ExecuteString() (string, int64, error) {
	return executeString(context.Background(), f.client, f.method, f.body, []string{f.tableName}, f.headers, f.params)
}

// ExecuteStringWithContext runs the PostgREST query, returning the result as
// a JSON string.
func (f *FilterBuilder) ExecuteStringWithContext(ctx context.Context) (string, int64, error) {
	return executeString(ctx, f.client, f.method, f.body, []string{f.tableName}, f.headers, f.params)
}

// Execute runs the PostgREST query, returning the result as a byte slice.
func (f *FilterBuilder) Execute() ([]byte, int64, error) {
	return execute(context.Background(), f.client, f.method, f.body, []string{f.tableName}, f.headers, f.params)
}

// ExecuteWithContext runs the PostgREST query with the given context,
// returning the result as a byte slice.
func (f *FilterBuilder) ExecuteWithContext(ctx context.Context) ([]byte, int64, error) {
	return execute(ctx, f.client, f.method, f.body, []string{f.tableName}, f.headers, f.params)
}

// ExecuteTo runs the PostgREST query, encoding the result to the supplied
// interface. Note that the argument for the to parameter should always be a
// reference to a slice.
func (f *FilterBuilder) ExecuteTo(to interface{}) (countType, error) {
	return executeTo(context.Background(), f.client, f.method, f.body, to, []string{f.tableName}, f.headers, f.params)
}

// ExecuteToWithContext runs the PostgREST query with the given context,
// encoding the result to the supplied interface. Note that the argument for
// the to parameter should always be a reference to a slice.
func (f *FilterBuilder) ExecuteToWithContext(ctx context.Context, to interface{}) (countType, error) {
	return executeTo(ctx, f.client, f.method, f.body, to, []string{f.tableName}, f.headers, f.params)
}

var filterOperators = []string{"eq", "neq", "gt", "gte", "lt", "lte", "like", "ilike", "is", "in", "cs", "cd", "sl", "sr", "nxl", "nxr", "adj", "ov", "fts", "plfts", "phfts", "wfts"}

// appendFilter is a helper method that appends a filter to existing filters on a column
func (f *FilterBuilder) appendFilter(column, filterValue string) *FilterBuilder {
	if existing, ok := f.params[column]; ok && existing != "" {
		// If a filter already exists for this column, combine with 'and'
		f.params["and"] = fmt.Sprintf("(%s.%s,%s.%s)", column, existing, column, filterValue)
		delete(f.params, column)
	} else if existingAnd, ok := f.params["and"]; ok {
		// If an 'and' parameter already exists, append to it
		f.params["and"] = strings.TrimSuffix(existingAnd, ")") + "," + column + "." + filterValue + ")"
	} else {
		f.params[column] = filterValue
	}
	return f
}

func isOperator(value string) bool {
	for _, operator := range filterOperators {
		if value == operator {
			return true
		}
	}
	return false
}

// Filter adds a filtering operator to the query. For a list of available
// operators, see: https://postgrest.org/en/stable/api.html#operators
func (f *FilterBuilder) Filter(column, operator, value string) *FilterBuilder {
	if !isOperator(operator) {
		f.client.ClientError = fmt.Errorf("invalid filter operator")
		return f
	}
	return f.appendFilter(column, fmt.Sprintf("%s.%s", operator, value))
}

func (f *FilterBuilder) And(filters, foreignTable string) *FilterBuilder {
	if foreignTable != "" {
		f.params[foreignTable+".and"] = fmt.Sprintf("(%s)", filters)
	} else {
		f.params[foreignTable+"and"] = fmt.Sprintf("(%s)", filters)
	}
	return f
}

func (f *FilterBuilder) Or(filters, foreignTable string) *FilterBuilder {
	if foreignTable != "" {
		f.params[foreignTable+".or"] = fmt.Sprintf("(%s)", filters)
	} else {
		f.params[foreignTable+"or"] = fmt.Sprintf("(%s)", filters)
	}
	return f
}

func (f *FilterBuilder) Not(column, operator, value string) *FilterBuilder {
	if !isOperator(operator) {
		return f
	}
	return f.Filter(column, "not."+operator, value)
}

func (f *FilterBuilder) Match(userQuery map[string]string) *FilterBuilder {
	for key, value := range userQuery {
		f.Filter(key, "eq", value)
	}
	return f
}

func (f *FilterBuilder) Eq(column, value string) *FilterBuilder {
	return f.Filter(column, "eq", value)
}

func (f *FilterBuilder) Neq(column, value string) *FilterBuilder {
	return f.Filter(column, "neq", value)
}

func (f *FilterBuilder) Gt(column, value string) *FilterBuilder {
	return f.Filter(column, "gt", value)
}

func (f *FilterBuilder) Gte(column, value string) *FilterBuilder {
	return f.Filter(column, "gte", value)
}

func (f *FilterBuilder) Lt(column, value string) *FilterBuilder {
	return f.Filter(column, "lt", value)
}

func (f *FilterBuilder) Lte(column, value string) *FilterBuilder {
	return f.Filter(column, "lte", value)
}

func (f *FilterBuilder) Like(column, value string) *FilterBuilder {
	return f.Filter(column, "like", value)
}

func (f *FilterBuilder) Ilike(column, value string) *FilterBuilder {
	return f.Filter(column, "ilike", value)
}

func (f *FilterBuilder) Is(column, value string) *FilterBuilder {
	return f.Filter(column, "is", value)
}

func (f *FilterBuilder) In(column string, values []string) *FilterBuilder {
	var cleanedValues []string
	illegalChars := regexp.MustCompile("[,()]")
	for _, value := range values {
		exp := illegalChars.MatchString(value)
		if exp {
			cleanedValues = append(cleanedValues, fmt.Sprintf("\"%s\"", value))
		} else {
			cleanedValues = append(cleanedValues, value)
		}
	}
	return f.appendFilter(column, fmt.Sprintf("in.(%s)", strings.Join(cleanedValues, ",")))
}

func (f *FilterBuilder) Contains(column string, value []string) *FilterBuilder {
	newValue := []string{}
	for _, v := range value {
		newValue = append(newValue, fmt.Sprintf("%#v", v))
	}

	valueString := fmt.Sprintf("{%s}", strings.Join(newValue, ","))

	return f.appendFilter(column, "cs."+valueString)
}

func (f *FilterBuilder) ContainedBy(column string, value []string) *FilterBuilder {
	newValue := []string{}
	for _, v := range value {
		newValue = append(newValue, fmt.Sprintf("%#v", v))
	}

	valueString := fmt.Sprintf("{%s}", strings.Join(newValue, ","))

	return f.appendFilter(column, "cd."+valueString)
}

func (f *FilterBuilder) ContainsObject(column string, value interface{}) *FilterBuilder {
	sum, err := json.Marshal(value)
	if err != nil {
		f.client.ClientError = err
		return f
	}
	return f.appendFilter(column, "cs."+string(sum))
}

func (f *FilterBuilder) ContainedByObject(column string, value interface{}) *FilterBuilder {
	sum, err := json.Marshal(value)
	if err != nil {
		f.client.ClientError = err
		return f
	}
	return f.appendFilter(column, "cd."+string(sum))
}

func (f *FilterBuilder) RangeLt(column, value string) *FilterBuilder {
	return f.appendFilter(column, "sl."+value)
}

func (f *FilterBuilder) RangeGt(column, value string) *FilterBuilder {
	return f.appendFilter(column, "sr."+value)
}

func (f *FilterBuilder) RangeGte(column, value string) *FilterBuilder {
	return f.appendFilter(column, "nxl."+value)
}

func (f *FilterBuilder) RangeLte(column, value string) *FilterBuilder {
	return f.appendFilter(column, "nxr."+value)
}

func (f *FilterBuilder) RangeAdjacent(column, value string) *FilterBuilder {
	return f.appendFilter(column, "adj."+value)
}

func (f *FilterBuilder) Overlaps(column string, value []string) *FilterBuilder {
	newValue := []string{}
	for _, v := range value {
		newValue = append(newValue, fmt.Sprintf("%#v", v))
	}

	valueString := fmt.Sprintf("{%s}", strings.Join(newValue, ","))
	return f.appendFilter(column, "ov."+valueString)
}

// TextSearch performs a full-text search filter. For more information, see
// https://postgrest.org/en/stable/api.html#fts.
func (f *FilterBuilder) TextSearch(column, userQuery, config, tsType string) *FilterBuilder {
	var typePart, configPart string
	if tsType == "plain" {
		typePart = "pl"
	} else if tsType == "phrase" {
		typePart = "ph"
	} else if tsType == "websearch" {
		typePart = "w"
	} else if tsType == "" {
		typePart = ""
	} else {
		f.client.ClientError = fmt.Errorf("invalid text search type")
		return f
	}
	if config != "" {
		configPart = fmt.Sprintf("(%s)", config)
	}
	return f.appendFilter(column, typePart+"fts"+configPart+"."+userQuery)
}

// OrderOpts describes the options to be provided to Order.
type OrderOpts struct {
	Ascending    bool
	NullsFirst   bool
	ForeignTable string
}

// DefaultOrderOpts is the default set of options used by Order.
var DefaultOrderOpts = OrderOpts{
	Ascending:    false,
	NullsFirst:   false,
	ForeignTable: "",
}

// Limit the result to the specified count.
func (f *FilterBuilder) Limit(count int, foreignTable string) *FilterBuilder {
	if foreignTable != "" {
		f.params[foreignTable+".limit"] = strconv.Itoa(count)
	} else {
		f.params["limit"] = strconv.Itoa(count)
	}

	return f
}

// Order the result with the specified column. A pointer to an OrderOpts
// object can be supplied to specify ordering options.
func (f *FilterBuilder) Order(column string, opts *OrderOpts) *FilterBuilder {
	if opts == nil {
		opts = &DefaultOrderOpts
	}

	key := "order"
	if opts.ForeignTable != "" {
		key = opts.ForeignTable + ".order"
	}

	ascendingString := "desc"
	if opts.Ascending {
		ascendingString = "asc"
	}

	nullsString := "nullslast"
	if opts.NullsFirst {
		nullsString = "nullsfirst"
	}

	existingOrder, ok := f.params[key]
	if ok && existingOrder != "" {
		f.params[key] = fmt.Sprintf("%s,%s.%s.%s", existingOrder, column, ascendingString, nullsString)
	} else {
		f.params[key] = fmt.Sprintf("%s.%s.%s", column, ascendingString, nullsString)
	}

	return f
}

// Range Limits the result to rows within the specified range, inclusive.
func (f *FilterBuilder) Range(from, to int, foreignTable string) *FilterBuilder {
	if foreignTable != "" {
		f.params[foreignTable+".offset"] = strconv.Itoa(from)
		f.params[foreignTable+".limit"] = strconv.Itoa(to - from + 1)
	} else {
		f.params["offset"] = strconv.Itoa(from)
		f.params["limit"] = strconv.Itoa(to - from + 1)
	}
	return f
}

// Single Retrieves only one row from the result. The total result set must be one row
// (e.g., by using Limit). Otherwise, this will result in an error.
func (f *FilterBuilder) Single() *FilterBuilder {
	f.headers["Accept"] = "application/vnd.pgrst.object+json"
	return f
}
