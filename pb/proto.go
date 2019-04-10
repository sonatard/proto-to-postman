package pb

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"golang.org/x/xerrors"
)

func MessageFromName(msg string, fds []*descriptor.FileDescriptorSet) (*descriptor.DescriptorProto, error) {
	for _, fd := range fds {
		for _, protoFile := range fd.GetFile() {
			for _, message := range protoFile.GetMessageType() {
				fullMessage := strings.Join([]string{"", protoFile.GetPackage(), message.GetName()}, ".")
				if fullMessage == msg {
					return message, nil
				}
			}
		}
	}

	return nil, xerrors.Errorf("input type(%s) not found", msg)
}

func BodyStruct(msg *descriptor.DescriptorProto, fds []*descriptor.FileDescriptorSet) (interface{}, error) {
	fields := make([]reflect.StructField, 0, len(msg.Field))

	for _, field := range msg.GetField() {
		var t reflect.Type
		if field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			msg, err := MessageFromName(field.GetTypeName(), fds)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			body, err := BodyStruct(msg, fds)
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
