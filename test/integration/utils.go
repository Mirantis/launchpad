package test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
)

type MKEConnect struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// UnAuthorizedRequest is a helper function to make an unauthorized request
func UnAuthHttpRequest(url string, reqMethod string, requestBody []byte, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest(reqMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("Error on building request.\n[ERRO] - %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := doRequest(req, client)
	if err != nil {
		fmt.Printf("Error on response.\n[ERRO] - %s", err)
		return nil, err
	}

	return resp, nil
}

// AuthorizedRequest is a helper function to make an authorized request
func AuthHttpRequest(url string, token string, reqMethod string, requestBody []byte, client *http.Client) (*http.Response, error) {
	var bearer = "Bearer " + token
	req, err := http.NewRequest(reqMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("Error on building request.\n[ERRO] - %s", err)
	}
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := doRequest(req, client)
	if err != nil {
		fmt.Printf("Error on response.\n[ERRO] - %s", err)
		return nil, err
	}

	return resp, nil
}

func doRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetHTTPClient is a helper function to create an HTTP client
func GetHTTPClient(tlsConfig *tls.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{Transport: transport}
}
