package postgrest

import (
	"fmt"
	"strings"
)

type FilterBuilder struct {
	client *Client
	method string
	body   []byte
}

func (f *FilterBuilder) Execute() {
	_ = f.body
	_ = f.client
	_ = f.method
}

var filterOperators = []string{"eq", "neq", "gt", "gte", "lt", "lte", "like", "ilike", "is", "in", "cs", "cd", "sl", "sr", "nxl", "nxr", "adj", "ov", "fts", "plfts", "phfts", "wfts"}

func isOperator(value string) bool {
	for _, operator := range filterOperators {
		if value == operator {
			return true
		}
	}
	return false
}

func (f *FilterBuilder) Filter(column, operator, value string) {
	if !isOperator(operator) {
		f.client.ClientError = fmt.Errorf("invalid filter operator")
		return
	}
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, operator+"."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Or(filters, foreignTable string) {
	query := f.client.clientTransport.baseURL.Query()
	if foreignTable != "" {
		query.Add(foreignTable+".or", fmt.Sprintf("(%s)", filters))
	} else {
		query.Add("or", fmt.Sprintf("(%s)", filters))
	}
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Not(column, operator, value string) {
	if !isOperator(operator) {
		return
	}
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "not."+operator+"."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Match(userQuery map[string]string) {
	query := f.client.clientTransport.baseURL.Query()
	for key, value := range userQuery {
		query.Add(key, "eq."+value)
	}
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Eq(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "eq."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Neq(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "neq."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Gt(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "gt."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Gte(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "gte."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Lt(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "lt."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Lte(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "lte."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Like(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "like."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Ilike(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "ilike."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Is(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "is."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) In() {}
func (f *FilterBuilder) Contains(column string, value []string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "cs."+strings.Join(value, ","))
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) ContainedBy(column string, value []string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "cd."+strings.Join(value, ","))
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) ContainsObject()    {}
func (f *FilterBuilder) ContainedByObject() {}

func (f *FilterBuilder) RangeLt(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "sl."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) RangeGt(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "sr."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) RangeGte(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "nxl."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) RangeLte(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "nxr."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) RangeAdjacent(column, value string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "adj."+value)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) Overlaps(column string, value []string) {
	query := f.client.clientTransport.baseURL.Query()
	query.Add(column, "ov."+strings.Join(value, ","))
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}

func (f *FilterBuilder) TextSearch(column, userQuery, config, tsType string) {
	query := f.client.clientTransport.baseURL.Query()
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
		return
	}
	if config != "" {
		configPart = fmt.Sprintf("(%s)", config)
	}
	query.Add(column, typePart+"fts"+configPart+"."+userQuery)
	f.client.clientTransport.baseURL.RawQuery = query.Encode()
}
