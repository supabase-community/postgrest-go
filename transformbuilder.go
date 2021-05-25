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

	if foreignTable != "" {
		t.client.clientTransport.params.Add(foreignTable+".limit", fmt.Sprint(count))
	} else {
		t.client.clientTransport.params.Add("limit", fmt.Sprint(count))
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

	existingOrder := t.client.clientTransport.params.Get(key)

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
		t.client.clientTransport.params.Set(key, existingOrder+","+column+"."+ascendingString+"."+nullsString)
	} else {
		t.client.clientTransport.params.Add(key, column+"."+ascendingString+"."+nullsString)
	}

	return t
}

func (t *TransformBuilder) Range(from, to int, foreignTable string) *TransformBuilder {

	if foreignTable != "" {
		t.client.clientTransport.params.Add(foreignTable+".offset", fmt.Sprint(from))
		t.client.clientTransport.params.Add(foreignTable+".limit", fmt.Sprint(to-from+1))
	} else {
		t.client.clientTransport.params.Add("offset", fmt.Sprint(from))
		t.client.clientTransport.params.Add("limit", fmt.Sprint(to-from+1))
	}
	return t
}

func (t *TransformBuilder) Single() *TransformBuilder {
	t.client.clientTransport.header.Set("Accept", "application/vnd.pgrst.object+json")
	return t
}
