package pkg

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Server struct {
	Service *protogen.Service
	Paths   []APIPath
}

// APIPath each rpc
type APIPath struct {
	Method      *protogen.Method
	Tags        []string
	Roles       []string
	Features    []string
	Description string
	Summary     string
	GoPath      string
	OpenAPIPath string
	HTTPMethod  string
	Parameters  []Parameter
}

// BuildParameters builds parameters
func (r *APIPath) BuildParameters(pathKeys map[string]string) {
	r.Parameters = parseParameters(r.Method.Input, pathKeys, "", "")
	pathCount := countPathParameters(r.Parameters)
	if pathCount != len(pathKeys) {
		panic("unmatched path keys (some of the path keys do not match body parameters)")
	}
}

func parseParameters(
	msg *protogen.Message,
	pathKeys map[string]string,
	keypref string,
	reqkeypref string,
) []Parameter {
	finalParams := []Parameter{}
	for _, field := range msg.Fields {
		kind := field.Desc.Kind()

		key := keypref + field.GoName
		isPath := false
		requestedKey := reqkeypref + field.Desc.JSONName()
		if val, ok := pathKeys[requestedKey]; ok {
			isPath = true
			requestedKey = val
		}

		ismsg := kind == protoreflect.MessageKind &&
			(field.Message.Desc.FullName() != "google.protobuf.Timestamp" &&
				field.Message.Desc.FullName() != "google.protobuf.ListValue" &&
				field.Message.Desc.FullName() != "google.protobuf.Struct")

		switch ismsg {
		case true:
			p := Parameter{
				field,
				requestedKey,
				key,
				field.GoName,
				"struct",
				field.Desc.HasOptionalKeyword(),
				field.Desc.IsList(),
				isPath,
				parseParameters(field.Message, pathKeys, key+".", requestedKey+"."),
			}
			finalParams = append(finalParams, p)
		default:
			_, rawType, _ := getGolangType(field)
			p := Parameter{
				field,
				requestedKey,
				key,
				field.GoName,
				rawType,
				field.Desc.HasOptionalKeyword(),
				field.Desc.IsList(),
				isPath,
				[]Parameter{},
			}
			finalParams = append(finalParams, p)
		}
	}

	return finalParams
}

func countPathParameters(
	prms []Parameter,
) int {
	count := 0
	for idx := range prms {
		if len(prms[idx].Holding) != 0 {
			count += countPathParameters(prms[idx].Holding)
		} else if prms[idx].IsPath {
			count++
		}
	}
	return count
}

func getGolangType(f *protogen.Field) (fullType, rawType string, notAType bool) {
	if f.Desc.HasOptionalKeyword() {
		fullType = "*"
	}
	if f.Desc.IsList() {
		fullType = fullType + "[]"
	}

	kind := f.Desc.Kind()
	switch kind {
	case protoreflect.BoolKind:
		fullType = fullType + BoolType
		rawType = BoolType
	case protoreflect.StringKind:
		fullType = fullType + StringType
		rawType = StringType
	case protoreflect.Int32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Sint32Kind:
		fullType = fullType + Int32Type
		rawType = Int32Type
	case protoreflect.Fixed32Kind, protoreflect.Uint32Kind:
		fullType = fullType + UInt32Type
		rawType = UInt32Type
	case protoreflect.Int64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Sint64Kind:
		fullType = fullType + Int64Type
		rawType = Int64Type
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		fullType = fullType + UInt64Type
		rawType = UInt64Type
	case protoreflect.FloatKind:
		fullType = fullType + Float32Type
		rawType = Float32Type
	case protoreflect.DoubleKind:
		fullType = fullType + Float64Type
		rawType = Float64Type
	case protoreflect.BytesKind:
		fullType = fullType + BytesType
		rawType = BytesType
	case protoreflect.EnumKind:
		fullType = fullType + StringType
		notAType = true
		rawType = EnumType
	case protoreflect.MessageKind:
		if f.Message.Desc.FullName() == "google.protobuf.Timestamp" {
			fullType = fullType + TimeType
			notAType = true
			rawType = TimeType
		} else if f.Message.Desc.FullName() == "google.protobuf.Struct" {
			fullType = fullType + AnyType
			notAType = true
			rawType = AnyType
		} else if f.Message.Desc.FullName() == "google.protobuf.ListValue" {
			fullType = fullType + AnySliceType
			notAType = true
			rawType = AnySliceType
		} else {
			fullType = fullType + StructType
			notAType = true
			rawType = StructType
		}
	case protoreflect.GroupKind:
		fullType = fullType + "group"
		notAType = true
		rawType = "group"
	}
	return fullType, rawType, notAType
}

const (
	Int32Type    = "int32"
	UInt32Type   = "uint32"
	Int64Type    = "int64"
	UInt64Type   = "uint64"
	Float32Type  = "float32"
	Float64Type  = "float64"
	BytesType    = "[]byte"
	EnumType     = "enum"
	StringType   = "string"
	BoolType     = "bool"
	StructType   = "struct"
	TimeType     = "Time"
	AnyType      = "any"
	AnySliceType = "[]any"
)

type Parameter struct {
	Field         *protogen.Field
	RequestedKey  string
	FullParameter string
	PropertyName  string
	Type          string
	IsOptional    bool
	IsList        bool
	IsPath        bool
	Holding       []Parameter
	// resolve Pointer to Input
}
