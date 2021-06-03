package postgrest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type FilterBuilder struct {
	client *Client
	method string
	body   []byte
}

func (f *FilterBuilder) Execute() (string, error) {
	return Execute(f.client, f.method, f.body)
}

func (f *FilterBuilder) ExecuteReturnByteArray() ([]byte, error) {
	return ExecuteReturnByteArray(f.client, f.method, f.body)
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

func (f *FilterBuilder) Filter(column, operator, value string) *FilterBuilder {
	if !isOperator(operator) {
		f.client.ClientError = fmt.Errorf("invalid filter operator")
		return f
	}
	f.client.clientTransport.params.Add(column, operator+"."+value)
	return f
}

func (f *FilterBuilder) Or(filters, foreignTable string) *FilterBuilder {
	if foreignTable != "" {
		f.client.clientTransport.params.Add(foreignTable+".or", fmt.Sprintf("(%s)", filters))
	} else {
		f.client.clientTransport.params.Add("or", fmt.Sprintf("(%s)", filters))
	}
	return f
}

func (f *FilterBuilder) Not(column, operator, value string) *FilterBuilder {
	if !isOperator(operator) {
		return f
	}
	f.client.clientTransport.params.Add(column, "not."+operator+"."+value)
	return f
}

func (f *FilterBuilder) Match(userQuery map[string]string) *FilterBuilder {
	for key, value := range userQuery {
		f.client.clientTransport.params.Add(key, "eq."+value)
	}
	return f
}

func (f *FilterBuilder) Eq(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "eq."+value)
	return f
}

func (f *FilterBuilder) Neq(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "neq."+value)
	return f
}

func (f *FilterBuilder) Gt(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "gt."+value)
	return f
}

func (f *FilterBuilder) Gte(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "gte."+value)
	return f
}

func (f *FilterBuilder) Lt(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "lt."+value)
	return f
}

func (f *FilterBuilder) Lte(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "lte."+value)
	return f
}

func (f *FilterBuilder) Like(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "like."+value)
	return f
}

func (f *FilterBuilder) Ilike(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "ilike."+value)
	return f
}

func (f *FilterBuilder) Is(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "is."+value)
	return f
}

func (f *FilterBuilder) In(column string, value []string) *FilterBuilder {
	cleanedValues := value
	var values []string
	for _, cleanValue := range cleanedValues {
		exp, _ := regexp.MatchString(cleanValue, "[,()]")
		if exp {
			values = append(values, "\"cleanValue\"")
		}
	}
	f.client.clientTransport.params.Add(column, "in."+strings.Join(values, ","))
	return f
}

func (f *FilterBuilder) Contains(column string, value []string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "cs."+strings.Join(value, ","))
	return f
}

func (f *FilterBuilder) ContainedBy(column string, value []string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "cd."+strings.Join(value, ","))
	return f
}

func (f *FilterBuilder) ContainsObject(column string, value interface{}) *FilterBuilder {
	sum , err := json.Marshal(value)
	if err != nil {
		f.client.ClientError = err
	}
	f.client.clientTransport.params.Add(column, "cs."+string(sum))
	return f
}

func (f *FilterBuilder) ContainedByObject(column string, value interface{}) *FilterBuilder{
	sum , err := json.Marshal(value)
	if err != nil {
		f.client.ClientError = err
	}
	f.client.clientTransport.params.Add(column, "cs."+string(sum))
	return f
}

func (f *FilterBuilder) RangeLt(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "sl."+value)
	return f
}

func (f *FilterBuilder) RangeGt(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "sr."+value)
	return f
}

func (f *FilterBuilder) RangeGte(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "nxl."+value)
	return f
}

func (f *FilterBuilder) RangeLte(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "nxr."+value)
	return f
}

func (f *FilterBuilder) RangeAdjacent(column, value string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "adj."+value)
	return f
}

func (f *FilterBuilder) Overlaps(column string, value []string) *FilterBuilder {
	f.client.clientTransport.params.Add(column, "ov."+strings.Join(value, ","))
	return f
}

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
	f.client.clientTransport.params.Add(column, typePart+"fts"+configPart+"."+userQuery)
	return f
}
