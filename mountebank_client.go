package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type MountebankClient struct {
	getResponseURL string
}

func NewMountebankClient(getResponseURL string) *MountebankClient {
	return &MountebankClient{
		getResponseURL: getResponseURL,
	}
}

type getResponseRequest struct {
	Request any `json:"request"`
}

type getResponseProxy struct {
	To string `json:"to"`
}

type getResponseResponse struct {
	// standard response
	Response json.RawMessage `json:"response"`

	// proxy response
	Proxy            *getResponseProxy `json:"proxy"`
	ProxyCallbackURL string            `json:"callbackURL"`
}

func (c *MountebankClient) GetResponse(ctx context.Context, request any) (*getResponseResponse, error) {
	data, err := json.Marshal(&getResponseRequest{Request: request})
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.getResponseURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode)
	}

	httpBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	log.Println("mb", string(httpBody))

	response := &getResponseResponse{}
	err = json.Unmarshal(httpBody, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

type saveProxyResponseRequest struct {
	Response any `json:"proxyResponse"`
}

func (c *MountebankClient) SaveProxyResponse(ctx context.Context, callbackURL string, response any) (*getResponseResponse, error) {
	data, err := json.Marshal(&saveProxyResponseRequest{Response: response})
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		callbackURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", httpResponse.StatusCode)
	}

	httpBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	getResponse := &getResponseResponse{}
	err = json.Unmarshal(httpBody, &getResponse.Response)
	if err != nil {
		return nil, err
	}

	return getResponse, nil
}
