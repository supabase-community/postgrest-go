package postgrest

import (
	"encoding/json"
	"strings"
)

type QueryBuilder struct {
	client *Client
	method string
	body   []byte
}

func (q *QueryBuilder) Execute() (string, error) {
	return Execute(q.client, q.method, q.body)
}

func (q *QueryBuilder) Select(columns, count string, head bool) *FilterBuilder {
	if head {
		q.method = "HEAD"
	} else {
		q.method = "GET"
	}

	if columns == "" {
		q.client.clientTransport.params.Add("select", "*")
	} else {
		quoted := false
		var resultArr = []string{}
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
		q.client.clientTransport.params.Add("select", result)
	}

	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		if q.client.clientTransport.header.Get("Prefer") == "" {
			q.client.clientTransport.header.Set("Prefer", "count="+count)
		} else {
			q.client.clientTransport.header.Set("Prefer", q.client.clientTransport.header.Get("Prefer")+",count="+count)
		}
	}
	return &FilterBuilder{client: q.client, method: q.method, body: q.body}
}

func (q *QueryBuilder) Upsert(value interface{}, onConflict, returning, count string) *FilterBuilder {
	q.method = "POST"

	if onConflict != "" {
		q.client.clientTransport.params.Add("on_conflict", onConflict)
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
	q.client.clientTransport.header.Set("Prefer", strings.Join(headerList, ","))

	// Get body if exist
	var byteBody []byte = nil
	if value != nil {
		jsonBody, err := json.Marshal(value)
		if err != nil {
			q.client.ClientError = err
			return &FilterBuilder{}
		}
		byteBody = jsonBody
	}
	q.body = byteBody
	return &FilterBuilder{client: q.client, method: q.method, body: q.body}
}

func (q *QueryBuilder) Delete(returning, count string) *FilterBuilder {
	q.method = "DELETE"

	headerList := []string{}
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.client.clientTransport.header.Set("Prefer", strings.Join(headerList, ","))
	return &FilterBuilder{client: q.client, method: q.method, body: q.body}
}

func (q *QueryBuilder) Update(value interface{}, returning, count string) *FilterBuilder {
	q.method = "PATCH"

	headerList := []string{}
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.client.clientTransport.header.Set("Prefer", strings.Join(headerList, ","))

	// Get body if exist
	var byteBody []byte = nil
	if value != nil {
		jsonBody, err := json.Marshal(value)
		if err != nil {
			q.client.ClientError = err
			return &FilterBuilder{}
		}
		byteBody = jsonBody
	}
	q.body = byteBody
	return &FilterBuilder{client: q.client, method: q.method, body: q.body}
}
