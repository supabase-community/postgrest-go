package postgrest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

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
