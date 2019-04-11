package pbdesc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/jhump/protoreflect/desc"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/api/annotations"
)

var ErrBodyNotFound = xerrors.New("body not found in HTTP Rule annotation")

type ProtoDescriptor struct{}

func (f *ProtoDescriptor) BodyMsgTypeNameByHTTPRule(inputType *desc.MessageDescriptor, rule *annotations.HttpRule) (*desc.MessageDescriptor, error) {
	body := rule.GetBody()
	if body == "" {
		return nil, ErrBodyNotFound
	}

	if body == "*" {
		return inputType, nil
	}

	for _, field := range inputType.GetFields() {
		if field.GetName() == body {
			return field.GetMessageType(), nil
		}
	}

	return nil, xerrors.Errorf("field name(%s) not found", body)
}

func (f *ProtoDescriptor) JSONBody(bodyMsgType *desc.MessageDescriptor) (string, error) {
	body, err := f.BodyStruct(bodyMsgType)
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

func (f *ProtoDescriptor) BodyStruct(msg *desc.MessageDescriptor) (interface{}, error) {
	fields := make([]reflect.StructField, 0, len(msg.GetFields()))

	for _, field := range msg.GetFields() {
		var t reflect.Type
		if field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			body, err := f.BodyStruct(field.GetMessageType())
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
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, field.GetJSONName())),
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
