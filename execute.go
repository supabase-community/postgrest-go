package postgrest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// countType is the integer type returned from execute functions when a count
// specifier is supplied to a builder.
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
		client.l.Error("Client error", zap.Error(client.ClientError))
		return nil, 0, client.ClientError
	}

	readerBody := bytes.NewBuffer(body)
	baseUrl := path.Join(append([]string{client.Transport.baseURL.Path}, urlFragments...)...)
	client.l.Debug("Creating request", zap.String("method", method), zap.String("url", baseUrl), zap.ByteString("body", body))
	req, err := http.NewRequest(method, baseUrl, readerBody)
	if err != nil {
		client.l.Error("Error creating request", zap.Error(err))
		return nil, 0, fmt.Errorf("error creating request: %s", err.Error())
	}

	for key, val := range headers {
		req.Header.Add(key, val)
	}
	q := req.URL.Query()
	for key, val := range params {
		q.Add(key, val)
	}
	req.URL.RawQuery = q.Encode()
	client.l.Debug("Final request", zap.String("url", req.URL.String()), zap.Any("headers", req.Header))

	resp, err := client.session.Do(req)
	if err != nil {
		client.l.Error("Error performing request", zap.Error(err))
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		client.l.Error("Error reading response body", zap.Error(err))
		return nil, 0, err
	}

	client.l.Debug("Received response", zap.Int("status_code", resp.StatusCode), zap.ByteString("body", respBody))

	// https://postgrest.org/en/stable/api.html#errors-and-http-status-codes
	if resp.StatusCode >= 400 {
		var errmsg *ExecuteError
		err := json.Unmarshal(respBody, &errmsg)
		if err != nil {
			client.l.Error("Error parsing error response", zap.Error(err))
			return nil, 0, fmt.Errorf("error parsing error response: %s", err.Error())
		}
		client.l.Error("PostgREST error", zap.String("code", errmsg.Code), zap.String("message", errmsg.Message))
		return nil, 0, fmt.Errorf("(%s) %s", errmsg.Code, errmsg.Message)
	}

	var count countType
	contentRange := resp.Header.Get("Content-Range")
	if contentRange != "" {
		split := strings.Split(contentRange, "/")
		if len(split) > 1 && split[1] != "*" {
			count, err = strconv.ParseInt(split[1], 0, 64)
			if err != nil {
				client.l.Error("Error parsing count from Content-Range header", zap.Error(err))
				return nil, 0, fmt.Errorf("error parsing count from Content-Range header: %s", err.Error())
			}
		}
	}

	return respBody, count, nil
}
func executeString(client *Client, method string, body []byte, urlFragments []string, headers map[string]string, params map[string]string) (string, countType, error) {
	resp, count, err := executeHelper(client, method, body, urlFragments, headers, params)
	return string(resp), count, err
}

func execute(client *Client, method string, body []byte, urlFragments []string, headers map[string]string, params map[string]string) ([]byte, countType, error) {
	return executeHelper(client, method, body, urlFragments, headers, params)
}

func executeTo(client *Client, method string, body []byte, to interface{}, urlFragments []string, headers map[string]string, params map[string]string) (countType, error) {
	resp, count, err := executeHelper(client, method, body, urlFragments, headers, params)

	if err != nil {
		client.l.Error("Error in executeTo", zap.Error(err))
		return count, err
	}

	readableRes := bytes.NewBuffer(resp)
	err = json.NewDecoder(readableRes).Decode(&to)
	if err != nil {
		client.l.Error("Error decoding response body", zap.Error(err))
	}
	return count, err
}
