package postgrest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ExecuteError is the error response format from postgrest. We really
// only use Code and Message, but we'll keep it as a struct for now.

type ExecuteError struct {
	Hint    string `json:"hint"`
	Details string `json:"details"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func executeHelper(client *Client, method string, body []byte) ([]byte, error) {
	if client.ClientError != nil {
		return nil, client.ClientError
	}

	readerBody := bytes.NewBuffer(body)
	req, err := http.NewRequest(method, client.clientTransport.baseURL.Path, readerBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.session.Do(req)
	if err != nil {
		return nil, err
	}

	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// https://postgrest.org/en/stable/api.html#errors-and-http-status-codes
	if resp.StatusCode >= 400 {
		var errmsg *ExecuteError
		err := json.Unmarshal(respbody, &errmsg)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("(%s) %s", errmsg.Code, errmsg.Message)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return respbody, nil
}

func executeString(client *Client, method string, body []byte) (string, error) {
	resp, err := executeHelper(client, method, body)
	return string(resp), err
}

func execute(client *Client, method string, body []byte) ([]byte, error) {
	return executeHelper(client, method, body)
}

func executeTo(client *Client, method string, body []byte, to interface{}) error {
	resp, err := executeHelper(client, method, body)

	if err != nil {
		return err
	}

	readableRes := bytes.NewBuffer(resp)

	err = json.NewDecoder(readableRes).Decode(&to)
	return err
}
