package main

import (

	// "google.golang.org/genproto/googleapis/api/annotations"
	// "google.golang.org/genproto/googleapis/api/serviceconfig"
	// "bytes"
	// "go/ast"
	// "go/build"
	// "go/printer"
	// "go/token"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	// "google.golang.org/protobuf/proto"
	// "google.golang.org/protobuf/reflect/protoreflect"
	// "google.golang.org/protobuf/types/descriptorpb"
	"github.com/BetaLixT/golang-tooling/protoc-gen-goconstrgen/pkg"
	// "google.golang.org/protobuf/proto"
	// "google.golang.org/protobuf/runtime/protoimpl"
	// "google.golang.org/protobuf/types/descriptorpb"
	// "google.golang.org/protobuf/proto"
	// "google.golang.org/protobuf/types/descriptorpb"
)

const (
	timePackage     = protogen.GoImportPath("time")
	fmtPackage      = protogen.GoImportPath("fmt")
	pbTimePackage   = protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb")
	pbStructPackage = protogen.GoImportPath("google.golang.org/protobuf/types/known/structpb")
)

func main() {
	protogen.Options{}.Run(func(p *protogen.Plugin) error {
		for _, f := range p.Files {
			if f.Generate {
				if err := GenerateFile(p, f); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// GenerateFile generating file
func GenerateFile(
	plugin *protogen.Plugin,
	file *protogen.File,
) error {
	// isGenerated := false

	plugin.SupportedFeatures = 1
	// protojsonPackage := protogen.GoImportPath("google.golang.org/protobuf/encoding/protojson")
	gofilename := file.GeneratedFilenamePrefix + ".con.go"
	f := plugin.NewGeneratedFile(gofilename, file.GoImportPath)

	f.P("// Code generated by protoc-gen-goconsgen. DO NOT EDIT.")
	f.P("// source: ", file.Desc.Path())
	f.P()
	f.P("package ", file.GoPackageName)

	for _, msg := range file.Messages {
		errorable := false
		f.P("func New", msg.GoIdent.GoName, "(")
		for _, field := range msg.Fields {
			// f.P(pkg.ToPrivateName(field.GoIdent.GoName), " ", field.Desc.Kind().String(), ",")
			if GenerateParameter(f, field) {
				errorable = true
			}
		}
		errsuf := ""
		if errorable {
			errsuf = ", error"
		}
		f.P(") (*", msg.GoIdent, errsuf, ") {")

		keys := make([]string, len(msg.Fields))
		vals := make([]any, len(msg.Fields))
		for idx, field := range msg.Fields {
			k, v := ResolveAssignment(f, field)
			keys[idx] = k
			vals[idx] = v
		}
		f.P("return &", msg.GoIdent, "{")
		for idx := range msg.Fields {
			f.P(keys[idx], ": ", vals[idx], ",")
		}
		if errorable {
			f.P("}, nil}")
		} else {
			f.P("}}")
		}
		f.P("")
	}
	return nil
}

// GenerateParameter generates function parameter
func GenerateParameter(
	f *protogen.GeneratedFile,
	field *protogen.Field,
) bool {
	if field.Desc.IsMap() {
		var key *protogen.Field
		var value *protogen.Field
		for _, f := range field.Message.Fields {
			if f.Desc.JSONName() == "value" {
				value = f
			} else if f.Desc.JSONName() == "key" {
				key = f
			}
		}

		if key.Desc.Kind() == protoreflect.MessageKind {
			panic("unsupported key type for map")
		}

		p1, p2, errorable := GetParameterType(value, false)
		f.P(pkg.ToPrivateName(field.GoName), " map[", key.Desc.Kind().String(), "]", p1, p2, ",")
		return errorable
	}

	p1, p2, errorable := GetParameterType(field, field.Desc.HasOptionalKeyword())
	pref := ""
	if field.Desc.IsList() {
		pref = " []"
	}

	f.P(pkg.ToPrivateName(field.GoName), " ", pref, p1, p2, ",")
	return errorable
}

// GetParameterType resolves type
func GetParameterType(field *protogen.Field, optional bool) (any, any, bool) {
	kind := field.Desc.Kind()
	switch kind {
	case protoreflect.MessageKind:
		if field.Message.Desc.FullName() == "google.protobuf.Timestamp" {
			pref := ""
			if optional {
				pref = "*"
			}
			return pref, timePackage.Ident("Time"), false
		}
		if field.Message.Desc.FullName() == "google.protobuf.Struct" {
			return "", "map[string]interface{}", true
		}
		return "*", field.Message.GoIdent, false
	case protoreflect.EnumKind:
		pref := ""
		if optional {
			pref = "*"
		}
		return pref, "string", true
	case protoreflect.BytesKind:
		pref := ""
		return pref, "[]byte", false
	default:
		pref := ""
		if optional {
			pref = "*"
		}

		_, rt, _ := getGolangType(field)
		return pref, rt, false
	}
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
	Int32Type   = "int32"
	UInt32Type  = "uint32"
	Int64Type   = "int64"
	UInt64Type  = "uint64"
	Float32Type = "float32"
	Float64Type = "float64"
	BytesType   = "[]byte"
	EnumType    = "enum"
	StringType  = "string"
	BoolType    = "bool"
	StructType  = "struct"
	TimeType    = "Time"
	AnyType     = "any"
)

// ResolveAssignment resolves assignments to object
func ResolveAssignment(
	f *protogen.GeneratedFile,
	field *protogen.Field,
) (string, any) {
	kind := field.Desc.Kind()
	switch kind {
	case protoreflect.MessageKind:
		if field.Message.Desc.FullName() == "google.protobuf.Timestamp" {
			if field.Desc.IsList() {
				org := pkg.ToPrivateName(field.GoName)
				pname := "res" + field.GoName
				f.P(pname, " := make([]*", pbTimePackage.Ident("Timestamp"), ", len(", org, "))")
				f.P("for idx := range ", org, " {")
				f.P(pname, "[idx] = ", pbTimePackage.Ident("New("+org+"[idx])"))
				f.P("}")
				return field.GoName, pname
			}

			pname := pkg.ToPrivateName(field.GoName)
			if field.Desc.HasOptionalKeyword() {
				ptrPname := "resp" + field.GoName
				f.P(ptrPname, ":= (*", pbTimePackage.Ident("Timestamp"), ")(nil)")
				f.P("if ", pname, " != nil {")
				f.P(ptrPname, " = ", pbTimePackage.Ident("New(*"+pname+")"))
				f.P("}")
				return field.GoName, ptrPname
			} else {
				return field.GoName, pbTimePackage.Ident("New(" + pname + ")")
			}

		}
		if field.Message.Desc.FullName() == "google.protobuf.Struct" {
			pname := pkg.ToPrivateName(field.GoName)

			if field.Desc.IsList() {

				f.P("res", field.GoName, " := ([]*", pbStructPackage.Ident("Struct"), ")(nil)")

				f.P("if ", pname, " != nil {")
				f.P(
					"res",
					field.GoName,
					" = make([]*",
					pbStructPackage.Ident("Struct"),
					", 0, len(",
					pname,
					"))",
				)
				f.P("for idx := range ", pname, " {")
				f.P("temp, err := ", pbStructPackage.Ident("NewStruct("+pname+"[idx])"))
				f.P("if err != nil {")
				f.P("return nil, err")
				f.P("}")
				f.P("res", field.GoName, " = append(res", field.GoName, ", temp)")
				f.P("}")

				if !field.Desc.HasOptionalKeyword() {
					f.P("} else {")
					f.P("res", field.GoName, " = []*", pbStructPackage.Ident("Struct"), "{}")
				}
				f.P("}")
			} else {
				if field.Desc.HasOptionalKeyword() {
					f.P("res", field.GoName, " := (*", pbStructPackage.Ident("Struct"), ")(nil)")

					f.P("if ", pname, " != nil {")
					f.P("var err error")
					f.P("res", field.GoName, ", err = ", pbStructPackage.Ident("NewStruct("+pname+")"))
					f.P("if err != nil {")
					f.P("return nil, err")
					f.P("}")
					f.P("}")
				} else {
					f.P("res", field.GoName, ", err := ", pbStructPackage.Ident("NewStruct("+pname+")"))
					f.P("if err != nil {")
					f.P("return nil, err")
					f.P("}")
				}
			}
			return field.GoName, "res" + field.GoName
		}
		return field.GoName, pkg.ToPrivateName(field.GoName)

	case protoreflect.EnumKind:
		if field.Desc.IsList() {
			org := pkg.ToPrivateName(field.GoName)
			pname := "res" + field.GoName
			f.P(pname, " := make([]", field.Enum.GoIdent, ", len(", org, "))")
			f.P("for idx := range ", org, " {")
			f.P("temp, ok := ", field.Enum.GoIdent, "_value[", org, "[idx]]")
			f.P("if !ok {")
			f.P("return nil, ", fmtPackage.Ident("Errorf(\"invalid enum value\")"))
			f.P("}")
			f.P(pname, "[idx] = ", field.Enum.GoIdent, "(temp)")
			f.P("}")
			return field.GoName, pname
		}

		pname := pkg.ToPrivateName(field.GoName)

		pref := "resp"
		keypref := ""
		if field.Desc.HasOptionalKeyword() {
			keypref = "*"
			f.P(pref, field.GoName, ":= (*", field.Enum.GoIdent, ")(nil)")
			f.P("if ", pname, " != nil {")
		}
		f.P("res", field.GoName, ", ok := ", field.Enum.GoIdent, "_value[", keypref, pname, "]")
		f.P("if !ok {")
		f.P("return nil, ", fmtPackage.Ident("Errorf(\"invalid enum value\")"))
		f.P("}")
		if field.Desc.HasOptionalKeyword() {
			f.P("t", " := ", field.Enum.GoIdent, "(res", field.GoName, ")")
			f.P("resp", field.GoName, " = &t")
			f.P("}")
		} else {
			f.P("resp", field.GoName, " := ", field.Enum.GoIdent, "(res", field.GoName, ")")
		}
		return field.GoName, pref + field.GoName
	default:
		return field.GoName, pkg.ToPrivateName(field.GoName)
	}
}

// func fullType(pkg *build.Package, e ast.Expr) string {
// 	ast.Inspect(e, func(n ast.Node) bool {
// 		switch n := n.(type) {
// 		case *ast.Ident:
// 			// Using typeSpec instead of IsExported here would be
// 			// more accurate, but it'd be crazy expensive, and if
// 			// the type isn't exported, there's no point trying
// 			// to implement it anyway.
// 			if n.IsExported() {
// 				n.Name = pkg.Name + "." + n.Name
// 			}
// 		case *ast.SelectorExpr:
// 			return false
// 		}
// 		return true
// 	})
// 	var buf bytes.Buffer
// 	printer.Fprint(&buf, token.NewFileSet(), e)
// 	return buf.String()
// }
