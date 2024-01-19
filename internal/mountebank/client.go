package mountebank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type Client struct {
	getResponseURL string
}

func NewClient(getResponseURL string) *Client {
	return &Client{
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
	Response *rpcResponse `json:"response"`

	// proxy response
	Proxy            *getResponseProxy `json:"proxy"`
	ProxyCallbackURL string            `json:"callbackURL"`
}

type GetResponseResponse struct {
	// standard response
	Response *RpcResponse

	// proxy response
	Proxy            *getResponseProxy
	ProxyCallbackURL string
}

func (c *Client) GetResponse(
	ctx context.Context,
	request *RpcData,
	desc protoreflect.MessageDescriptor,
) (*GetResponseResponse, error) {
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

	{
		log.Println("mb", string(data), string(httpBody))
	}

	temp := &getResponseResponse{}
	err = json.Unmarshal(httpBody, temp)
	if err != nil {
		return nil, err
	}

	resp, err := convert(temp.Response, desc)
	if err != nil {
		return nil, err
	}

	return &GetResponseResponse{
		Response:         resp,
		Proxy:            temp.Proxy,
		ProxyCallbackURL: temp.ProxyCallbackURL,
	}, nil
}

type saveProxyResponseRequest struct {
	Response any `json:"proxyResponse"`
}

func (c *Client) SaveProxyResponse(
	ctx context.Context,
	callbackURL string,
	response *RpcResponse,
	desc protoreflect.MessageDescriptor,
) (*GetResponseResponse, error) {
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

	temp := &getResponseResponse{}
	err = json.Unmarshal(httpBody, &temp.Response)
	if err != nil {
		return nil, err
	}

	resp, err := convert(temp.Response, desc)
	if err != nil {
		return nil, err
	}

	return &GetResponseResponse{Response: resp}, nil
}
