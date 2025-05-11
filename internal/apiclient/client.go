package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const BaseURL = "https://api.mangaupdates.com/v1/"

var client = &http.Client{Timeout: 30 * time.Second}

func BuildURL(path string, queryParams map[string]string) (string, error) {
	baseURL, err := url.Parse(BaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	finalURL := baseURL.JoinPath(path)

	if queryParams != nil {
		q := finalURL.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		finalURL.RawQuery = q.Encode()
	}

	return finalURL.String(), nil
}

func DoRequest(method, fullURL string, bodyData interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	var req *http.Request
	var err error

	if bodyData != nil {
		jsonData, err := json.Marshal(bodyData)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshall request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err = http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	if bodyData != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json, application/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
