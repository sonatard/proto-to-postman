package main

import (
	"net/http"
	"path"
)

type Postman struct {
	Info PostmanInfo   `json:"info"`
	Item []PostmanItem `json:"item"`
}

type PostmanInfo struct {
	PostmanID string `json:"_postman_id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}

type PostmanItem struct {
	Name                    string                         `json:"name"`
	Request                 PostmanRequest                 `json:"request"`
	Response                []interface{}                  `json:"response"`
	ProtocolProfileBehavior PostmanProtocolProfileBehavior `json:"protocolProfileBehavior,omitempty"`
}

type PostmanRequest struct {
	Method string          `json:"method"`
	Header []PostmanHeader `json:"header"`
	Body   PostmanBody     `json:"body"`
	URL    PostmanURL      `json:"url"`
}

type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
}

type PostmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

type PostmanURL struct {
	Raw  string   `json:"raw"`
	Host []string `json:"host"`
	Path []string `json:"path"`
}

type PostmanProtocolProfileBehavior struct {
	DisableBodyPruning bool `json:"disableBodyPruning"`
}

type PostmanAPIParam struct {
	BaseURL string
	Service string
	Method  string
	Body    string
	Headers []*PostmanHeaderParam
}

type PostmanHeaderParam struct {
	Key   string
	Value string
}

func BuildPostman(configName string, apis []*PostmanAPIParam) *Postman {
	configID := ""

	var postmanItems []PostmanItem
	for _, api := range apis {
		postmanItem := BuildPostmanItem(api)
		postmanItems = append(postmanItems, postmanItem)
	}

	return NewPostman(configID, configName, postmanItems)
}

func BuildPostmanItem(api *PostmanAPIParam) PostmanItem {
	apiName := path.Join(api.Service, api.Method)
	httpMethod := http.MethodPost
	var headers []PostmanHeader
	for _, h := range api.Headers {
		header := NewPostmanHeader(h.Key, h.Value)
		headers = append(headers, header)
	}

	body := NewPostmanBody(api.Body)
	url := NewPostmanURL(api.BaseURL, []string{api.Service, api.Method})
	return NewPostmanItem(apiName, httpMethod, headers, body, url)
}

func NewPostmanHeader(key string, value string) PostmanHeader {
	return PostmanHeader{
		Key:   key,
		Value: value,
		Type:  "text",
		Name:  key,
	}
}

func NewPostmanBody(value string) PostmanBody {
	return PostmanBody{
		Mode: "raw",
		Raw:  value,
	}
}

func NewPostmanURL(host string, paths []string) PostmanURL {
	all := append([]string{host}, paths...)
	return PostmanURL{
		Raw:  path.Join(all...),
		Host: []string{host},
		Path: paths,
	}
}
func NewPostmanItem(apiName string, httpMethod string, header []PostmanHeader, body PostmanBody, url PostmanURL) PostmanItem {
	return PostmanItem{
		Name: apiName,
		Request: PostmanRequest{
			Method: httpMethod,
			Header: header,
			Body:   body,
			URL:    url,
		},
		Response: nil,
		ProtocolProfileBehavior: PostmanProtocolProfileBehavior{
			DisableBodyPruning: true,
		},
	}

}

func NewPostman(configID, configName string, items []PostmanItem) *Postman {
	return &Postman{
		Info: PostmanInfo{
			PostmanID: configID,
			Name:      configName,
			// TODO: choose Schema
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: items,
	}
}
