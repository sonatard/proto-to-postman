package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sonatard/proto-to-postman/postman"

	"golang.org/x/xerrors"
)

type Option struct {
	importPaths []string
	configName  string
	baseURL     string
	headers     []*postman.HeaderParam
}

func parseOption() (*Option, []string, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage of %s:
   %s [OPTIONS] [pb files...]
Options\n`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get current directory: %v", err)
	}

	configName := flag.String("n", "", `config name`)
	protoImportOpt := flag.String("i", dir, `pb files import directory`)
	baseURLOpt := flag.String("b", "", `request API Base Path e.g) -b https://example.com/`)
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

	postHeaderParams := make([]*postman.HeaderParam, 0, len(headers))
	for _, header := range headers {
		h := strings.Split(header, ":")
		if len(h) != 2 {
			return nil, nil, xerrors.New("header format is wrong. HeaderName:HeaderValue")
		}
		postHeaderParam := &postman.HeaderParam{
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
