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


## Support proto file format

### No annotation

Create POST method`{URL}/{SERVICE_NAME}/{METHOD_NAME}`.

```proto
import "google/api/annotations.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse){};
}

message GetUserRequest {
    User user = 1;
}

message GetUserResponse {}

message User {
  string user_id= 1;
  string user_name = 2;

}
```

### google.api.http annotation

Possible to create multi APIs `{URL}/{AnnotationValue}`.

```proto
import "google/api/annotations.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse){
    option (google.api.http) = {
       post: "/UserService/GetUser"
       body: "user"
       additional_bindings {
         post: "/UserService/GetUser2"
         body: "*"
       }
       additional_bindings {
         get: "/UserService/GetUser"
       }
    };
  };
}

message GetUserRequest {
    User user = 1;
}

message GetUserResponse {}

message User {
  string user_id= 1;
  string user_name = 2;

}
```

google.api.http spec is here.

https://cloud.google.com/endpoints/docs/grpc-service-config/reference/rpc/google.api#google.api.HttpRule

## Limitation

- Only supported for Postman v2.1.0 scheme

