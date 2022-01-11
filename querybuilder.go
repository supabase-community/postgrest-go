package postgrest

import (
	"encoding/json"
	"fmt"
	"strings"
)

type QueryBuilder struct {
	client    *Client
	method    string
	body      []byte
	tableName string
	headers   map[string]string
	params    map[string]string
}

func (q *QueryBuilder) ExecuteString() (string, countType, error) {
	return executeString(q.client, q.method, q.body, []string{q.tableName}, q.headers, q.params)
}

func (q *QueryBuilder) Execute() ([]byte, countType, error) {
	return execute(q.client, q.method, q.body, []string{q.tableName}, q.headers, q.params)
}

func (q *QueryBuilder) ExecuteTo(to interface{}) error {
	return executeTo(q.client, q.method, q.body, to, []string{q.tableName}, q.headers, q.params)
}

func (q *QueryBuilder) Select(columns, count string, head bool) *FilterBuilder {
	if head {
		q.method = "HEAD"
	} else {
		q.method = "GET"
	}

	if columns == "" {
		q.params["select"] = "*"
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
		q.params["select"] = result
	}

	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		currentValue, ok := q.headers["Prefer"]
		if ok && currentValue != "" {
			q.headers["Prefer"] = fmt.Sprintf("%s,count=%s", currentValue, count)
		} else {
			q.headers["Prefer"] = fmt.Sprintf("count=%s", count)
		}
	}
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params}
}

func (q *QueryBuilder) Insert(value interface{}, upsert bool, onConflict, returning, count string) *FilterBuilder {
	q.method = "POST"

	if onConflict != "" && upsert {
		q.params["on_conflict"] = onConflict
	}

	headerList := []string{}
	if upsert {
		headerList = append(headerList, "resolution=merge-duplicates")
	}
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")

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
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params}
}
func (q *QueryBuilder) Upsert(value interface{}, onConflict, returning, count string) *FilterBuilder {
	q.method = "POST"

	if onConflict != "" {
		q.params["on_conflict"] = onConflict
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
	q.headers["Prefer"] = strings.Join(headerList, ",")

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
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params}
}

func (q *QueryBuilder) Delete(returning, count string) *FilterBuilder {
	q.method = "DELETE"

	var headerList []string
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params}
}

func (q *QueryBuilder) Update(value interface{}, returning, count string) *FilterBuilder {
	q.method = "PATCH"

	var headerList []string
	if returning == "" {
		returning = "representation"
	}
	if returning == "minimal" || returning == "representation" {
		headerList = append(headerList, "return="+returning)
	}
	if count != "" && (count == `exact` || count == `planned` || count == `estimated`) {
		headerList = append(headerList, "count="+count)
	}
	q.headers["Prefer"] = strings.Join(headerList, ",")

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
	return &FilterBuilder{client: q.client, method: q.method, body: q.body, tableName: q.tableName, headers: q.headers, params: q.params}
}
