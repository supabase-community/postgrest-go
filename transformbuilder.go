package postgrest

import (
	"fmt"
	"strconv"
)

type TransformBuilder struct {
	client    *Client
	method    string
	body      []byte
	headers   map[string]string
	params    map[string]string
}

func (t *TransformBuilder) ExecuteString() (string, error) {
	return executeString(t.client, t.method, t.body, []string{}, t.headers, t.params)
}

func (t *TransformBuilder) Execute() ([]byte, error) {
	return execute(t.client, t.method, t.body, []string{}, t.headers, t.params)
}

func (t *TransformBuilder) ExecuteTo(to interface{}) error {
	return executeTo(t.client, t.method, t.body, to, []string{}, t.headers, t.params)
}

func (t *TransformBuilder) Limit(count int, foreignTable string) *TransformBuilder {
	if foreignTable != "" {
		t.params[foreignTable + ".limit"] = strconv.Itoa(count)
	} else {
		t.params["limit"] = strconv.Itoa(count)
	}

	return t
}

func (t *TransformBuilder) Order(column, foreignTable string, ascending, nullsFirst bool) *TransformBuilder {
	var key string
	if foreignTable != "" {
		key = foreignTable + ".order"
	} else {
		key = "order"
	}

	existingOrder, ok := t.params[key]

	var ascendingString string
	if ascending {
		ascendingString = "asc"
	} else {
		ascendingString = "desc"
	}

	var nullsString string
	if nullsFirst {
		nullsString = "nullsfirst"
	} else {
		nullsString = "nullslast"
	}

	if ok && existingOrder != "" {
		t.params[key] = fmt.Sprintf("%s,%s.%s.%s", existingOrder, column, ascendingString, nullsString)
	} else {
		t.params[key] = fmt.Sprintf("%s.%s.%s", column, ascendingString, nullsString)
	}

	return t
}

func (t *TransformBuilder) Range(from, to int, foreignTable string) *TransformBuilder {
	if foreignTable != "" {
		t.params[foreignTable + ".offset"] = strconv.Itoa(from)
		t.params[foreignTable + ".limit"] = strconv.Itoa(to - from + 1)
	} else {
		t.params["offset"] = strconv.Itoa(from)
		t.params["limit"] = strconv.Itoa(to - from + 1)
	}
	return t
}

func (t *TransformBuilder) Single() *TransformBuilder {
	t.headers["Accept"] = "application/vnd.pgrst.object+json"
	return t
}
