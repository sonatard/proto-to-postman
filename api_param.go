package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/sonatard/proto-to-postman/pb"
	"github.com/sonatard/proto-to-postman/postman"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/api/annotations"
)

type apiParamsBuilder struct {
	baseURL string
	headers []*postman.HeaderParam
	fds     []*descriptor.FileDescriptorSet
}

func NewAPIParamsBuilder(baseURL string, headers []*postman.HeaderParam, fds []*descriptor.FileDescriptorSet) *apiParamsBuilder {
	return &apiParamsBuilder{
		baseURL: baseURL,
		headers: headers,
		fds:     fds,
	}
}

func (a *apiParamsBuilder) Build() ([]*postman.APIParam, error) {
	var apiParams []*postman.APIParam
	for _, fd := range a.fds {
		for _, protoFile := range fd.GetFile() {
			for _, service := range protoFile.GetService() {
				for _, method := range service.GetMethod() {
					params, err := a.build(method, service)
					if err != nil {
						return nil, xerrors.Errorf(": %w", err)
					}

					apiParams = append(apiParams, params...)
				}
			}
		}
	}

	return apiParams, nil
}

func (a *apiParamsBuilder) build(method *descriptor.MethodDescriptorProto, service *descriptor.ServiceDescriptorProto) ([]*postman.APIParam, error) {
	opts := method.GetOptions()
	if !proto.HasExtension(opts, annotations.E_Http) {
		apiParam, err := a.apiParamByMethod(method, service)
		if err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}

		return []*postman.APIParam{apiParam}, nil
	}

	ext, err := proto.GetExtension(opts, annotations.E_Http)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	rule, ok := ext.(*annotations.HttpRule)
	if !ok {
		return nil, xerrors.New("annotation extension assertion error")
	}

	apiParams, err := a.apiParamByHTTPRule(rule, method.GetInputType())
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	return apiParams, nil
}

func (a *apiParamsBuilder) jsonBody(bodyMsgType string) (string, error) {
	inputType, err := pb.MessageFromName(bodyMsgType, a.fds)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	body, err := pb.BodyStruct(inputType, a.fds)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	var out bytes.Buffer
	err = json.Indent(&out, b, "", "\t")
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	return out.String(), nil
}

func (a *apiParamsBuilder) apiParamByMethod(method *descriptor.MethodDescriptorProto, service *descriptor.ServiceDescriptorProto) (*postman.APIParam, error) {
	jsonBody, err := a.jsonBody(method.GetInputType())
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	return &postman.APIParam{
		BaseURL:    a.baseURL,
		HTTPMethod: http.MethodPost,
		Path:       "/" + path.Join(service.GetName(), method.GetName()),
		Body:       jsonBody,
		Headers:    a.headers,
	}, nil
}

func (a *apiParamsBuilder) apiParamByHTTPRule(rule *annotations.HttpRule, inputType string) ([]*postman.APIParam, error) {
	var apiParams []*postman.APIParam

	bodyMsgType, err := a.bodyMsgTypeByHTTPRule(rule, inputType)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	if endpoint := newEndpoint(rule); endpoint != nil {
		jsonBody, err := a.jsonBody(bodyMsgType)
		if err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}

		apiParam := &postman.APIParam{
			BaseURL:    a.baseURL,
			HTTPMethod: endpoint.method,
			Path:       endpoint.url,
			Body:       jsonBody,
			Headers:    a.headers,
		}
		apiParams = append(apiParams, apiParam)
	}

	for _, r := range rule.GetAdditionalBindings() {
		if endpoint := newEndpoint(r); endpoint != nil {
			bodyMsgType, err := a.bodyMsgTypeByHTTPRule(rule, inputType)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			jsonBody, err := a.jsonBody(bodyMsgType)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			apiParam := &postman.APIParam{
				BaseURL:    a.baseURL,
				HTTPMethod: endpoint.method,
				Path:       endpoint.url,
				Body:       jsonBody,
				Headers:    a.headers,
			}

			apiParams = append(apiParams, apiParam)
		}
	}

	return apiParams, nil
}

func (a *apiParamsBuilder) bodyMsgTypeByHTTPRule(rule *annotations.HttpRule, inputType string) (string, error) {
	body := rule.GetBody()
	if body == "" || body == "*" {
		return inputType, nil
	}

	req, err := pb.MessageFromName(inputType, a.fds)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	var bodyMsgType string
	for _, field := range req.GetField() {
		if field.GetName() == body {
			bodyMsgType = field.GetTypeName()
		}
	}

	return bodyMsgType, nil
}

type endpoint struct {
	method string
	url    string
}

func newEndpoint(rule *annotations.HttpRule) *endpoint {
	if rule == nil {
		return nil
	}

	var e *endpoint
	switch opt := rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		e = &endpoint{"GET", opt.Get}
	case *annotations.HttpRule_Put:
		e = &endpoint{"PUT", opt.Put}
	case *annotations.HttpRule_Post:
		e = &endpoint{"POST", opt.Post}
	case *annotations.HttpRule_Delete:
		e = &endpoint{"DELETE", opt.Delete}
	case *annotations.HttpRule_Patch:
		e = &endpoint{"PATCH", opt.Patch}
	}

	return e
}
