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
		for _, protoFile := range fd.File {
			for _, service := range protoFile.Service {
				for _, method := range service.Method {
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
	jsonBody, err := a.jsonBody(method.GetInputType())
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	opts := method.GetOptions()
	if !proto.HasExtension(opts, annotations.E_Http) {
		return []*postman.APIParam{
			a.apiParamByMethod(method, service, jsonBody),
		}, nil
	}

	ext, err := proto.GetExtension(opts, annotations.E_Http)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	rule, ok := ext.(*annotations.HttpRule)
	if !ok {
		return nil, xerrors.New("annotation extension assertion error")
	}

	return a.apiParamByHTTPRule(rule, jsonBody), nil
}

func (a *apiParamsBuilder) jsonBody(inputTypeString string) (string, error) {
	inputType, err := pb.InputTypeFromName(inputTypeString, a.fds)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	body := pb.BodyStruct(inputType)
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

func (a *apiParamsBuilder) apiParamByMethod(method *descriptor.MethodDescriptorProto, service *descriptor.ServiceDescriptorProto, out string) *postman.APIParam {
	return &postman.APIParam{
		BaseURL:    a.baseURL,
		HTTPMethod: http.MethodPost,
		Path:       "/" + path.Join(service.GetName(), method.GetName()),
		Body:       out,
		Headers:    a.headers,
	}
}

func (a *apiParamsBuilder) apiParamByHTTPRule(rule *annotations.HttpRule, out string) []*postman.APIParam {
	var apiParams []*postman.APIParam

	if endpoint := newEndpoint(rule); endpoint != nil {
		apiParam := &postman.APIParam{
			BaseURL:    a.baseURL,
			HTTPMethod: endpoint.method,
			Path:       endpoint.url,
			Body:       out,
			Headers:    a.headers,
		}
		apiParams = append(apiParams, apiParam)
	}

	for _, r := range rule.AdditionalBindings {
		if endpoint := newEndpoint(r); endpoint != nil {
			apiParam := &postman.APIParam{
				BaseURL:    a.baseURL,
				HTTPMethod: endpoint.method,
				Path:       endpoint.url,
				Body:       out,
				Headers:    a.headers,
			}

			apiParams = append(apiParams, apiParam)
		}
	}

	return apiParams
}

type endpoint struct {
	method string
	url    string
}

func newEndpoint(opts *annotations.HttpRule) *endpoint {
	if opts == nil {
		return nil
	}
	switch opt := opts.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return &endpoint{"GET", opt.Get}
	case *annotations.HttpRule_Put:
		return &endpoint{"PUT", opt.Put}
	case *annotations.HttpRule_Post:
		return &endpoint{"POST", opt.Post}
	case *annotations.HttpRule_Delete:
		return &endpoint{"DELETE", opt.Delete}
	case *annotations.HttpRule_Patch:
		return &endpoint{"PATCH", opt.Patch}
	}
	return nil
}
