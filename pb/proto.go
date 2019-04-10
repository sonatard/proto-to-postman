package pb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/api/annotations"
)

type FileDescriptors struct {
	Set []*descriptor.FileDescriptorSet
}

func (f *FileDescriptors) MessageFromName(msgName string) (*descriptor.DescriptorProto, error) {
	for _, fd := range f.Set {
		for _, protoFile := range fd.GetFile() {
			for _, message := range protoFile.GetMessageType() {
				fullMessage := strings.Join([]string{"", protoFile.GetPackage(), message.GetName()}, ".")
				if fullMessage == msgName {
					return message, nil
				}
			}
		}
	}

	return nil, xerrors.Errorf("input type(%s) not found", msgName)
}

func (f *FileDescriptors) BodyMsgTypeNameByHTTPRule(inputTypeName string, rule *annotations.HttpRule) (string, error) {
	body := rule.GetBody()
	if body == "" || body == "*" {
		return inputTypeName, nil
	}

	req, err := f.MessageFromName(inputTypeName)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	for _, field := range req.GetField() {
		if field.GetName() == body {
			return field.GetTypeName(), nil
		}
	}

	return "", xerrors.Errorf("field name(%s) not found", body)
}

func (f *FileDescriptors) JSONBody(bodyMsgType string) (string, error) {
	inputType, err := f.MessageFromName(bodyMsgType)
	if err != nil {
		return "", xerrors.Errorf(": %w", err)
	}

	body, err := f.BodyStruct(inputType)
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

func (f *FileDescriptors) BodyStruct(msg *descriptor.DescriptorProto) (interface{}, error) {
	fields := make([]reflect.StructField, 0, len(msg.Field))

	for _, field := range msg.GetField() {
		var t reflect.Type
		if field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			inputType, err := f.MessageFromName(field.GetTypeName())
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			body, err := f.BodyStruct(inputType)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			t = reflect.TypeOf(body)
		} else {
			t = reflectType(field.GetType())
		}

		f := reflect.StructField{
			Name: generator.CamelCase(field.GetName()),
			Type: t,
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field.GetJsonName())),
		}

		fields = append(fields, f)
	}

	bodyStruct := reflect.StructOf(fields)

	return reflect.New(bodyStruct).Elem().Interface(), nil
}

func reflectType(t descriptor.FieldDescriptorProto_Type) reflect.Type {
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
