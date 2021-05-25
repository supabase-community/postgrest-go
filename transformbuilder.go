package postgrest

import "fmt"

type TransformBuilder struct {
	client *Client
	method string
	body   []byte
}

func (t *TransformBuilder) Execute() (string, error) {
	return Execute(t.client, t.method, t.body)
}

func (t *TransformBuilder) Limit(count int, foreignTable string) *TransformBuilder {
	query := t.client.clientTransport.baseURL.Query()
	if foreignTable != "" {
		query.Add(foreignTable+".limit", fmt.Sprint(count))
	} else {
		query.Add("limit", fmt.Sprint(count))
	}
	t.client.clientTransport.baseURL.RawQuery = query.Encode()
	return t
}

func (t *TransformBuilder) Order(column, foreignTable string, ascending, nullsFirst bool) *TransformBuilder {
	var key string
	if foreignTable != "" {
		key = foreignTable + ".order"
	} else {
		key = "order"
	}
	query := t.client.clientTransport.baseURL.Query()
	existingOrder := query.Get(key)

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

	if existingOrder != "" {
		query.Set(key, existingOrder+","+column+"."+ascendingString+"."+nullsString)
	} else {
		query.Add(key, column+"."+ascendingString+"."+nullsString)
	}
	t.client.clientTransport.baseURL.RawQuery = query.Encode()
	return t
}

func (t *TransformBuilder) Range(from, to int, foreignTable string) *TransformBuilder {
	query := t.client.clientTransport.baseURL.Query()
	if foreignTable != "" {
		query.Add(foreignTable+".offset", fmt.Sprint(from))
		query.Add(foreignTable+".limit", fmt.Sprint(to-from+1))
	} else {
		query.Add("offset", fmt.Sprint(from))
		query.Add("limit", fmt.Sprint(to-from+1))
	}
	return t
}

func (t *TransformBuilder) Single() *TransformBuilder {
	t.client.clientTransport.header.Set("Accept", "application/vnd.pgrst.object+json")
	return t
}
