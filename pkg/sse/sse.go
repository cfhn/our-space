package sse

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tmaxmax/go-sse"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	ErrCardinalityMismatch = errors.New("multiple values provided for non-repeated field")
	ErrInvalidEnumValue    = errors.New("invalid enum value")
	ErrNestedMapping       = errors.New("nested mapping not implemented")
	ErrUnknownProtoKind    = errors.New("unknown proto kind")
	ErrPathParamRepeated   = errors.New("path params can't be repeated fields")
)

func GrpcProxy[Request proto.Message, Response proto.Message](
	cc *grpc.ClientConn, desc grpc.StreamDesc, fullMethod string,
) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		grpcRequest := (*new(Request)).ProtoReflect().New().Interface().(Request)

		err := mapQueryValuesToMessage(r.URL.Query(), grpcRequest.ProtoReflect())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = mapPathParamsToMessage(pathParams, grpcRequest.ProtoReflect())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cs, err := cc.NewStream(r.Context(), &desc, fullMethod)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		err = cs.SendMsg(grpcRequest)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		session, err := sse.Upgrade(w, r)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		err = session.Flush()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		for {
			grpcResponse := (*new(Response)).ProtoReflect().New().Interface().(Response)

			err = cs.RecvMsg(grpcResponse)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return
			}

			jsonBytes, err := protojson.Marshal(grpcResponse)
			if err != nil {
				return
			}

			sseMessage := &sse.Message{
				Type: sse.Type("data"),
			}

			sseMessage.AppendData(string(jsonBytes))

			err = session.Send(sseMessage)
			if err != nil {
				return
			}
			err = session.Flush()
			if err != nil {
				return
			}
		}
	}
}

func mapQueryValuesToMessage(values url.Values, msg protoreflect.Message) error {
	for param, queryValues := range values {
		field := msg.Type().Descriptor().Fields().ByJSONName(param)
		if field == nil {
			field = msg.Type().Descriptor().Fields().ByName(protoreflect.Name(param))
			if field == nil {
				continue
			}
		}

		if field.Cardinality() == protoreflect.Repeated {
			value, err := toProtoValueList(msg, field, queryValues)
			if err != nil {
				return err
			}

			msg.Set(field, value)
		} else if len(queryValues) == 1 {
			value, err := toProtoValue(field, queryValues[0])
			if err != nil {
				return err
			}

			msg.Set(field, value)
		} else {
			return ErrCardinalityMismatch
		}
	}

	return nil
}

func mapPathParamsToMessage(values map[string]string, msg protoreflect.Message) error {
	for k, v := range values {
		field := msg.Type().Descriptor().Fields().ByJSONName(k)
		if field == nil {
			field = msg.Type().Descriptor().Fields().ByName(protoreflect.Name(k))
			if field == nil {
				continue
			}
		}

		if field.Cardinality() == protoreflect.Repeated {
			return ErrPathParamRepeated
		}

		value, err := toProtoValue(field, v)
		if err != nil {
			return err
		}

		msg.Set(field, value)
	}

	return nil
}

func toProtoValue(field protoreflect.FieldDescriptor, value string) (protoreflect.Value, error) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfBool(v), nil
	case protoreflect.EnumKind:
		var enumValue protoreflect.EnumValueDescriptor
		index, err := strconv.Atoi(value)
		if err != nil {
			enumName := protoreflect.Name(value)
			if !enumName.IsValid() {
				return protoreflect.Value{}, ErrInvalidEnumValue
			}

			enumValue = field.Enum().Values().ByName(enumName)
		} else {
			enumValue = field.Enum().Values().ByNumber(protoreflect.EnumNumber(index))
		}

		if enumValue == nil {
			return protoreflect.Value{}, ErrInvalidEnumValue
		}

		return protoreflect.ValueOfEnum(enumValue.Number()), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(value), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(value)), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return protoreflect.Value{}, ErrNestedMapping
	default:
		return protoreflect.Value{}, ErrUnknownProtoKind
	}
}

func toProtoValueList(
	msg protoreflect.Message, field protoreflect.FieldDescriptor, values []string,
) (protoreflect.Value, error) {
	list := msg.NewField(field)

	for _, value := range values {
		protoValue, err := toProtoValue(field, value)
		if err != nil {
			return protoreflect.Value{}, err
		}

		list.List().Append(protoValue)
	}

	return list, nil
}
