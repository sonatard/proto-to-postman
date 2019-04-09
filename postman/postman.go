package postman

import (
	"path"
)

type Postman struct {
	Info Info   `json:"info"`
	Item []Item `json:"item"`
}

type Info struct {
	PostmanID string `json:"_postman_id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}

type Item struct {
	Name                    string                  `json:"name"`
	Request                 Request                 `json:"request"`
	Response                []interface{}           `json:"response"`
	ProtocolProfileBehavior ProtocolProfileBehavior `json:"protocolProfileBehavior,omitempty"`
}

type Request struct {
	Method string   `json:"method"`
	Header []Header `json:"header"`
	Body   Body     `json:"body"`
	URL    URL      `json:"url"`
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
}

type Body struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

type URL struct {
	Raw  string   `json:"raw"`
	Host []string `json:"host"`
	Path []string `json:"path"`
}

type ProtocolProfileBehavior struct {
	DisableBodyPruning bool `json:"disableBodyPruning"`
}

type APIParam struct {
	BaseURL    string
	HTTPMethod string
	Service    string
	Method     string
	Body       string
	Headers    []*HeaderParam
}

type HeaderParam struct {
	Key   string
	Value string
}

func Build(configName string, apis []*APIParam) *Postman {
	configID := ""

	var postmanItems []Item
	for _, api := range apis {
		postmanItem := BuildItem(api)
		postmanItems = append(postmanItems, postmanItem)
	}

	return NewPostman(configID, configName, postmanItems)
}

func BuildItem(api *APIParam) Item {
	apiName := path.Join(api.Service, api.Method)
	var headers []Header
	for _, h := range api.Headers {
		header := NewHeader(h.Key, h.Value)
		headers = append(headers, header)
	}

	body := NewBody(api.Body)
	url := NewURL(api.BaseURL, []string{api.Service, api.Method})
	return NewItem(apiName, api.HTTPMethod, headers, body, url)
}

func NewHeader(key string, value string) Header {
	return Header{
		Key:   key,
		Value: value,
		Type:  "text",
		Name:  key,
	}
}

func NewBody(value string) Body {
	return Body{
		Mode: "raw",
		Raw:  value,
	}
}

func NewURL(host string, paths []string) URL {
	all := append([]string{host}, paths...)
	return URL{
		Raw:  path.Join(all...),
		Host: []string{host},
		Path: paths,
	}
}
func NewItem(apiName string, httpMethod string, header []Header, body Body, url URL) Item {
	return Item{
		Name: apiName,
		Request: Request{
			Method: httpMethod,
			Header: header,
			Body:   body,
			URL:    url,
		},
		Response: nil,
		ProtocolProfileBehavior: ProtocolProfileBehavior{
			DisableBodyPruning: true,
		},
	}

}

func NewPostman(configID, configName string, items []Item) *Postman {
	return &Postman{
		Info: Info{
			PostmanID: configID,
			Name:      configName,
			// TODO: choose Schema
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: items,
	}
}
