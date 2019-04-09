package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/sonatard/proto-to-postman/postman"
	"github.com/sonatard/proto-to-postman/proto"
	"golang.org/x/xerrors"
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
					apiParam, err := a.build(method, service)
					if err != nil {
						return nil, xerrors.Errorf(": %w", err)
					}

					apiParams = append(apiParams, apiParam)
				}
			}
		}
	}

	return apiParams, nil
}

func (a *apiParamsBuilder) build(method *descriptor.MethodDescriptorProto, service *descriptor.ServiceDescriptorProto) (*postman.APIParam, error) {
	inputType, err := proto.InputTypeFromName(method.GetInputType(), a.fds)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	body := proto.BodyStruct(inputType)
	b, err := json.Marshal(body)
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	var out bytes.Buffer
	err = json.Indent(&out, b, "", "\t")
	if err != nil {
		return nil, xerrors.Errorf(": %w", err)
	}

	apiParam := &postman.APIParam{
		BaseURL:    a.baseURL,
		HTTPMethod: http.MethodPost,
		Service:    service.GetName(),
		Method:     method.GetName(),
		Body:       out.String(),
		Headers:    a.headers,
	}

	return apiParam, nil
}
