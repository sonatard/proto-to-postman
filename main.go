package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	parser "github.com/gogo/pbparser"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
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
					inputType, err := findInputType(*rpc.InputType, fds)
					if err != nil {
						return nil, xerrors.Errorf(": %w", err)
					}

					body := createBody(inputType)

					apiParam := &PostmanAPIParam{
						BaseURL: baseURL,
						Service: *service.Name,
						Method:  *rpc.Name,
						Body:    body,
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
				fullMessage := strings.Join([]string{"", *protoFile.Package, *message.Name}, ".")
				if fullMessage == inputTypeName {
					return message, nil
				}
			}
		}
	}

	return nil, xerrors.Errorf("input type(%s) not found", inputTypeName)
}

// TODO: Fix dirty code
//  1. Create body struct using reflect package and DescriptorProto.
//  2. Convert body struct to body json.
func createBody(msg *descriptor.DescriptorProto) string {
	var jsonBody string
	jsonBody += "{\n"
	for _, field := range msg.Field {
		jsonBody += fmt.Sprintf("\t\"%s\" : ", *field.Name)
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_STRING {
			jsonBody += fmt.Sprintf("\"\"")
		} else {
			jsonBody += fmt.Sprintf("0")
		}
		jsonBody += ",\n"
	}

	if jsonBody[len(jsonBody)-2:] == ",\n" {
		jsonBody = jsonBody[:len(jsonBody)-2] + "\n"
	}
	jsonBody += "}\n"

	if jsonBody == "{\n}\n" {
		jsonBody = "{}\n"
	}

	return jsonBody
}
