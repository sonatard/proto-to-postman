# proto-to-postman

proto-to-postman is a command line tool to create postman API import collection from `.proto`.

proto-to-postman uses `protoc` command.

## Install

```console
go get -u github.com/sonatard/proto-to-postman
```

## Usage

```console
proto-to-postman \
    -n xxx-api \
    -b "https://exmaple.com/" \
    -h "Content-Type:application/json,Accept:application/json"  \
    -i proto/ \
    api/v1/*.proto
```

## Limitation

- Only support POST Method
- Only support Postman v2.1.0 scheme
- Parse request body is dirty...

