package postgrest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
)

type countType = int64

// ExecuteError is the error response format from postgrest. We really
// only use Code and Message, but we'll keep it as a struct for now.

type ExecuteError struct {
	Hint    string `json:"hint"`
	Details string `json:"details"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func executeHelper(client *Client, method string, body []byte, urlFragments []string, headers map[string]string, params map[string]string) ([]byte, countType, error) {
	if client.ClientError != nil {
		return nil, 0, client.ClientError
	}

	readerBody := bytes.NewBuffer(body)
	baseUrl := path.Join(append([]string{client.clientTransport.baseURL.Path}, urlFragments...)...)
	req, err := http.NewRequest(method, baseUrl, readerBody)
	if err != nil {
		return nil, 0, err
	}

	for key, val := range headers {
		req.Header.Add(key, val)
	}
	q := req.URL.Query()
	for key, val := range params {
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := client.session.Do(req)
	if err != nil {
		return nil, 0, err
	}

	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	// https://postgrest.org/en/stable/api.html#errors-and-http-status-codes
	if resp.StatusCode >= 400 {
		var errmsg *ExecuteError
		err := json.Unmarshal(respbody, &errmsg)
		if err != nil {
			return nil, 0, err
		}
		return nil, 0, fmt.Errorf("(%s) %s", errmsg.Code, errmsg.Message)
	}

	var count countType

	contentRange := resp.Header.Get("Content-Range")
	if contentRange != "" {
		split := strings.Split(contentRange, "/")
		if len(split) > 1 && split[1] != "*" {
			count, err = strconv.ParseInt(split[1], 0, 64)
			if err != nil {
				return nil, 0, err
			}
		}
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, 0, err
	}

	return respbody, count, nil
}

func executeString(client *Client, method string, body []byte, urlFragments []string, headers map[string]string, params map[string]string) (string, countType, error) {
	resp, count, err := executeHelper(client, method, body, urlFragments, headers, params)
	return string(resp), count, err
}

func execute(client *Client, method string, body []byte, urlFragments []string, headers map[string]string, params map[string]string) ([]byte, countType, error) {
	return executeHelper(client, method, body, urlFragments, headers, params)
}

func executeTo(client *Client, method string, body []byte, to interface{}, urlFragments []string, headers map[string]string, params map[string]string) error {
	resp, _, err := executeHelper(client, method, body, urlFragments, headers, params)

	if err != nil {
		return err
	}

	readableRes := bytes.NewBuffer(resp)

	err = json.NewDecoder(readableRes).Decode(&to)
	return err
}
