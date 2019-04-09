package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/sonatard/proto-to-postman/parser"
	"golang.org/x/xerrors"
)

func main() {
	opt, paths, err := parseOption()
	if err != nil || len(paths) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := run(paths, opt.importPaths, os.Stdout, opt.configName, opt.baseURL, opt.headers); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

type Option struct {
	importPaths []string
	configName  string
	baseURL     string
	headers     []*PostmanHeaderParam
}

func parseOption() (*Option, []string, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage of %s:
   %s [OPTIONS] [proto files...]
Options\n`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get current directory: %v", err)
	}

	configName := flag.String("n", "", `config name`)
	protoImportOpt := flag.String("i", dir, `proto files import directory`)
	baseURLOpt := flag.String("b", "", `request API Base URL e.g) -b https://example.com/`)
	headerOpts := flag.String("h", "", `request headerOpts e.g) -h Content-Type:application/json,XXXX:ABC`)
	flag.Parse()

	protoImportPaths := strings.Split(*protoImportOpt, ",")
	for i := range protoImportPaths {
		protoImportPath, err := filepath.Abs(protoImportPaths[i])
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to get absolute path: %v", err)
		}
		protoImportPaths[i] = protoImportPath
	}

	headers := strings.Split(*headerOpts, ",")

	postHeaderParams := make([]*PostmanHeaderParam, 0, len(headers))
	for _, header := range headers {
		h := strings.Split(header, ":")
		if len(h) != 2 {
			return nil, nil, xerrors.New("header format is wrong. HeaderName:HeaderValue")
		}
		postHeaderParam := &PostmanHeaderParam{
			Key:   h[0],
			Value: h[1],
		}

		postHeaderParams = append(postHeaderParams, postHeaderParam)
	}

	return &Option{
		importPaths: protoImportPaths,
		configName:  *configName,
		baseURL:     *baseURLOpt,
		headers:     postHeaderParams,
	}, flag.Args(), nil
}

func run(files []string, importPaths []string, w io.Writer, configName, baseURL string, postHeaderParam []*PostmanHeaderParam) error {
	fds := make([]*descriptor.FileDescriptorSet, 0, len(files))
	for _, file := range files {
		fd, err := parser.ParseFile(file, importPaths...)
		if err != nil {
			return xerrors.Errorf("Unable to parse proto file: %v \n", err)
		}

		fds = append(fds, fd)
	}

	apiParams, err := createAPIParam(baseURL, fds, postHeaderParam)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	postman := BuildPostman(configName, apiParams)
	body, err := json.Marshal(postman)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	_, err = fmt.Fprintf(w, "%s\n", body)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func createAPIParam(baseURL string, fds []*descriptor.FileDescriptorSet, headers []*PostmanHeaderParam) ([]*PostmanAPIParam, error) {
	var apiParams []*PostmanAPIParam
	for _, fd := range fds {
		for _, protoFile := range fd.File {
			for _, service := range protoFile.Service {
				for _, rpc := range service.Method {
					inputType, err := findInputType(rpc.GetInputType(), fds)
					if err != nil {
						return nil, xerrors.Errorf(": %w", err)
					}

					bodyStruct := createBodyStruct(inputType)
					v := reflect.New(bodyStruct).Interface()
					b, err := json.Marshal(v)
					if err != nil {
						return nil, xerrors.Errorf(": %w", err)
					}
					var out bytes.Buffer
					json.Indent(&out, b, "", "\t")

					apiParam := &PostmanAPIParam{
						BaseURL: baseURL,
						Service: service.GetName(),
						Method:  rpc.GetName(),
						Body:    out.String(),
						Headers: headers,
					}

					apiParams = append(apiParams, apiParam)
				}
			}
		}
	}

	return apiParams, nil
}

func findInputType(inputTypeName string, fds []*descriptor.FileDescriptorSet) (*descriptor.DescriptorProto, error) {
	for _, fd := range fds {
		for _, protoFile := range fd.File {
			for _, message := range protoFile.MessageType {
				fullMessage := strings.Join([]string{"", protoFile.GetPackage(), message.GetName()}, ".")
				if fullMessage == inputTypeName {
					return message, nil
				}
			}
		}
	}

	return nil, xerrors.Errorf("input type(%s) not found", inputTypeName)
}

func createBodyStruct(msg *descriptor.DescriptorProto) reflect.Type {
	fields := make([]reflect.StructField, 0, len(msg.Field))

	for _, field := range msg.Field {
		f := reflect.StructField{
			Name: generator.CamelCase(field.GetName()),
			Type: createReflectType(field.GetType()),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field.GetJsonName())),
		}

		fields = append(fields, f)
	}

	return reflect.StructOf(fields)
}

func createReflectType(t descriptor.FieldDescriptorProto_Type) reflect.Type {
	switch t {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		var v float64
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		var v float32
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_INT64, descriptor.FieldDescriptorProto_TYPE_SINT64, descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		var v int64
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_UINT64, descriptor.FieldDescriptorProto_TYPE_FIXED64:
		var v uint64
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_TYPE_SINT32, descriptor.FieldDescriptorProto_TYPE_SFIXED32, descriptor.FieldDescriptorProto_TYPE_ENUM:
		var v int32
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_UINT32, descriptor.FieldDescriptorProto_TYPE_FIXED32:
		var v uint32
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		var v bool
		return reflect.TypeOf(v)
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		var v string
		return reflect.TypeOf(v)

	}

	var v string
	return reflect.TypeOf(v)
}
