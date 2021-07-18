package postgrest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ExecuteErrorResponse is the error response format from postgrest. We really
// only use Code and Message, but we'll keep it as a struct for now.

type ExecuteErrorResponse struct {
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

	respbody, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		return nil, rerr
	}

	// If we didnt get 20x response, unmarshal the error body to be able to format
	// an informative error message. Mayb this should be >= 400, since I doubt redirect
	// errors are returned.. but to be safe
	if resp.StatusCode >= 300 {
		var errmsg *ExecuteErrorResponse
		json.Unmarshal(respbody, &errmsg)
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
