package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/sonatard/proto-to-postman/parser"
	"github.com/sonatard/proto-to-postman/postman"
	"golang.org/x/xerrors"
)

func main() {
	opt, paths, err := parseOption()
	if err != nil || len(paths) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	w := os.Stdout
	if err := run(paths, opt.importPaths, opt.configName, opt.baseURL, opt.headers, w); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}

func run(files []string, importPaths []string, configName, baseURL string, headers []*postman.HeaderParam, w io.Writer) error {
	fds := make([]*descriptor.FileDescriptorSet, 0, len(files))
	for _, file := range files {
		fd, err := parser.ParseFile(file, importPaths...)
		if err != nil {
			return xerrors.Errorf("Unable to parse pb file: %v \n", err)
		}

		fds = append(fds, fd)
	}

	apiParamBuilder := NewAPIParamsBuilder(baseURL, headers, fds)
	apiParams, err := apiParamBuilder.Build()
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	postman := postman.Build(configName, apiParams)
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
