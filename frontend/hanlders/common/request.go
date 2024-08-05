package common

import (
	"bytes"
	"encoding/json"
	"expense-tracker/config"
	"net/http"
)

func MakeBackendHTTPRequest(method string, path string, accessToken string, payload any) (*http.Response, error) {
	apiUrl := "http://" + config.Envs.BackendURL + config.Envs.APIPath

	var req *http.Request
	var err error
	if payload != nil {
		marshalled, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(method, apiUrl+path, bytes.NewBuffer(marshalled))
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequest(method, apiUrl+path, nil)
		if err != nil {
			return nil, err
		}
	}

	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
