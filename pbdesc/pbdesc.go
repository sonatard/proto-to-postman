package pbdesc

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
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

	field := inputType.FindFieldByName(body)
	if field == nil {
		return nil, xerrors.Errorf("field name(%s) not found", body)
	}

	return field.GetMessageType(), nil
}

func (f *ProtoDescriptor) NewMessage(baseMsgType *desc.MessageDescriptor) (*dynamic.Message, error) {
	base := dynamic.NewMessage(baseMsgType)

	for _, field := range baseMsgType.GetFields() {
		if field.IsRepeated() {
			continue
		}

		if field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			fieldMsgType := field.GetMessageType()
			msg, err := f.NewMessage(fieldMsgType)
			if err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}

			if err := base.TrySetField(field, msg); err != nil {
				return nil, xerrors.Errorf(": %w", err)
			}
		}

	}

	return base, nil
}
