package main

import (
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/sonatard/proto-to-postman/pbdesc"
	"github.com/sonatard/proto-to-postman/postman"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/api/annotations"
)

type apiParamsBuilder struct {
	baseURL string
	headers []*postman.HeaderParam
	pbdesc  *pbdesc.ProtoDescriptor
}

func NewAPIParamsBuilder(baseURL string, headers []*postman.HeaderParam) *apiParamsBuilder {
	return &apiParamsBuilder{
		baseURL: baseURL,
		headers: headers,
		pbdesc:  &pbdesc.ProtoDescriptor{},
	}
}

func (a *apiParamsBuilder) Build(fds []*desc.FileDescriptor) ([]*postman.APIParam, error) {
	var apiParams []*postman.APIParam
	for _, fd := range fds {
		for _, service := range fd.GetServices() {
			for _, method := range service.GetMethods() {
				params, err := a.build(method, service)
				if err != nil {
					return nil, xerrors.Errorf(": %w", err)
				}

				apiParams = append(apiParams, params...)
			}
		}
	}

	return apiParams, nil
}

func (a *apiParamsBuilder) build(method *desc.MethodDescriptor, service *desc.ServiceDescriptor) ([]*postman.APIParam, error) {
	opts := method.GetOptions()

	if !proto.HasExtension(opts, annotations.E_Http) {
		return []*postman.APIParam{}, nil
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

func (a *apiParamsBuilder) apiParamByHTTPRule(rule *annotations.HttpRule, inputType *desc.MessageDescriptor) ([]*postman.APIParam, error) {
	var apiParams []*postman.APIParam

	if endpoint := newEndpoint(rule); endpoint != nil {
		bodyMsgTypeName, err := a.pbdesc.BodyMsgTypeNameByHTTPRule(inputType, rule)
		if err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}

		jsonBody, err := a.pbdesc.JSONBody(bodyMsgTypeName)
		if err != nil {
			return nil, xerrors.Errorf(": %w", err)
		}

		apiParam := &postman.APIParam{
			BaseURL:    a.baseURL,
			HTTPMethod: endpoint.method,
			Path:       endpoint.path,
			Body:       jsonBody,
			Headers:    a.headers,
		}

		apiParams = append(apiParams, apiParam)
	}

	for _, r := range rule.GetAdditionalBindings() {
		if endpoint := newEndpoint(r); endpoint != nil {
			bodyMsgTypeName, err := a.pbdesc.BodyMsgTypeNameByHTTPRule(inputType, r)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			jsonBody, err := a.pbdesc.JSONBody(bodyMsgTypeName)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			apiParam := &postman.APIParam{
				BaseURL:    a.baseURL,
				HTTPMethod: endpoint.method,
				Path:       endpoint.path,
				Body:       jsonBody,
				Headers:    a.headers,
			}

			apiParams = append(apiParams, apiParam)
		}
	}

	return apiParams, nil
}

type endpoint struct {
	method string
	path   string
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
