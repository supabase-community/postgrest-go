package postgrest

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func Execute(client *Client, method string, body []byte) (string, error) {
	readerBody := bytes.NewBuffer(body)
	req, err := http.NewRequest(method, client.clientTransport.baseURL.Path, readerBody)
	if err != nil {
		return "", err
	}
	resp, err := client.session.Do(req)
	if err != nil {
		return "", err
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := string(respbody)

	err = resp.Body.Close()
	if err != nil {
		return "", err
	}
	return result, nil
}
