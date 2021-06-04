package postgrest

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func ExecuteHelper(client *Client, method string, body []byte) ([]byte, error) {
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

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return respbody, nil
}

func Execute(client *Client, method string, body []byte) (string, error) {
	resp, err := ExecuteHelper(client, method, body)
	return string(resp), err
}

func ExecuteReturnByteArray(client *Client, method string, body []byte) ([]byte, error) {
	return ExecuteHelper(client, method, body)
}
