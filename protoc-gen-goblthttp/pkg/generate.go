// Package pkg package
package pkg

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

const (
	contextPackage   = protogen.GoImportPath("context")
	ginPackage       = protogen.GoImportPath("github.com/gin-gonic/gin")
	embedPackage     = protogen.GoImportPath("embed")
	protojsonPackage = protogen.GoImportPath("google.golang.org/protobuf/encoding/protojson")
	ioutilPackage    = protogen.GoImportPath("io/ioutil")
	fmtPackage       = protogen.GoImportPath("fmt")
	gorrPackage      = protogen.GoImportPath("github.com/betalixt/gorr")
	strconvPackage   = protogen.GoImportPath("strconv")
	timePackage      = protogen.GoImportPath("time")
	timepbPackage    = protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb")
	stringsPackage   = protogen.GoImportPath("strings")
	grpcPackage      = protogen.GoImportPath("google.golang.org/grpc")
)

// GenerateHTTPServers generates http servers
func GenerateHTTPServers(
	srvs []Server,
	g *protogen.GeneratedFile,
	_ *protogen.File,
) error {
	g.P(
		"func newMissingRequiredParametersError(parameter string) *",
		gorrPackage.Ident("Error"),
		"{",
	)
	g.P("return ", gorrPackage.Ident("NewError"), "(")
	g.P(gorrPackage.Ident("ErrorCode"), "{")
	g.P("		Code:    400,")
	g.P("		Message: \"MissingRequiredParametersError\",")
	g.P("	},")
	g.P("	400,")
	g.P("	\"missing field(s): \"+parameter,")
	g.P(")")
	g.P("}")
	g.P("")
	g.P("func newUnparsableParameterError(parameter string) *", gorrPackage.Ident("Error"), "{")
	g.P("return ", gorrPackage.Ident("NewError"), "(")
	g.P(gorrPackage.Ident("ErrorCode"), "{")
	g.P("		Code:    400,")
	g.P("		Message: \"UnparsableParametersError\",")
	g.P("	},")
	g.P("	400,")
	g.P("	\"failed to parsed or missing field(s): \"+parameter,")
	g.P(")")
	g.P("}")

	for _, srv := range srvs {
		intname := srv.Service.GoName + "HTTPServer"
		g.P(fmt.Sprintf("// %s", srv.Service.GoName))
		g.P("type ", intname, " interface {")
		for _, rpc := range srv.Paths {
			// inputStructName := rpc.Method.Input.GoIdent.GoName
			// if _, ok := imports[rpc.Method.Input.GoIdent.GoImportPath]; ok {
			// 	inputStructName = rpc.Method.Input.GoIdent.GoImportPath.Ident(inputStructName)
			// }
			g.Write([]byte(rpc.Method.Comments.Leading.String()))
			g.P(
				"\t",
				rpc.Method.GoName,
				"(",
				contextPackage.Ident("Context"),
				", *",
				rpc.Method.Input.GoIdent,
				") (*",
				rpc.Method.Output.GoIdent,
				", error)",
			)
			g.Write([]byte(rpc.Method.Comments.Trailing.String()))
		}
		g.P("}")

		// controllers
		// TODO: handle path and query parameter type :)
		ctrlName := ToPrivateName(srv.Service.GoName)
		g.P("type ", ctrlName, " struct {")
		g.P("app ", intname)
		g.P("}")

		for _, rpc := range srv.Paths {

			g.P("// ", rpc.Description)
			g.P(
				"func (p *",
				ctrlName,
				")",
				ToPrivateName(rpc.Method.GoName),
				"(ctx *",
				ginPackage.Ident("Context"),
				") {",
			)

			g.P("body := ", rpc.Method.Input.GoIdent, "{}")
			if rpc.HTTPMethod != "GET" && rpc.HTTPMethod != "DELETE" {
				// TODO if anything left in body
				g.P("raw, err :=", ioutilPackage.Ident("ReadAll"), "(ctx.Request.Body)")
				g.P("if err != nil {")
				g.P("	ctx.Error(err)")
				g.P("	return")
				g.P("}")
				g.P(protojsonPackage.Ident("Unmarshal"), "(raw, &body)")
			} else {
				renderQueryParameters(g, rpc.Parameters, []string{})
			}
			renderPathParameters(g, rpc.Parameters, []string{})

			// for _, qpm := range rpc.QueryParameters {
			// 	g.P("body.", qpm.ModelParameter, "= ctx.Query(\",", qpm.Key, "\")")
			// }
			// for _, pth := range rpc.PathParameters {
			// 	g.P("body.", pth.ModelParameter, "= ctx.Param(\",", pth.Key, "\")")
			// }

			g.P("var c ", contextPackage.Ident("Context"))
			g.P("if v, ok := ctx.Get(InternalContextKey); ok {")
			g.P("	c, _ = v.(", contextPackage.Ident("Context"), ")")
			g.P("}")
			g.P("if c == nil {")
			g.P("	c = ctx")
			g.P("}")

			g.P("res, err := p.app.", rpc.Method.GoName, "(")
			g.P("c,")
			g.P("&body,")
			g.P(")")
			g.P("if err != nil {")
			g.P("ctx.Error(err)")
			g.P("return")
			g.P("}")

			g.P("resraw, err := protomarsh.Marshal(res)")
			g.P("if err != nil {")
			g.P("	ctx.Error(err)")
			g.P("	return")
			g.P("}")
			g.P("ctx.Status(200)")
			g.P("ctx.Header(\"Content-Type\", \"application/json\")")
			g.P("_, err = ctx.Writer.Write(resraw)")
			g.P("if err != nil {")
			g.P("	ctx.Error(err)")
			g.P("	return")
			g.P("}")
			g.P("}")
		}

		g.P("func Register", srv.Service.GoName, "HTTPServer (")
		g.P("grp *", ginPackage.Ident("RouterGroup"), ",")
		g.P("srv ", intname, ",")
		g.P(") {")
		g.P("ctrl := ", ctrlName, "{app: srv}")
		for _, rpc := range srv.Paths {
			g.P(
				"grp.",
				rpc.HTTPMethod,
				"(\"",
				rpc.GoPath,
				"\", ",
				"ctrl.",
				ToPrivateName(rpc.Method.GoName),
				")",
			)
		}
		g.P("}")
	}

	return nil
}

func renderQueryParameters(
	g *protogen.GeneratedFile,
	prms []Parameter,
	filter []string,
) {
	for _, prm := range prms {
		found := false
		for idx := range filter {
			if prm.FullParameter == filter[idx] {
				found = true
				break
			}
		}

		if found || prm.IsPath {
			continue
		}
		if len(prm.Holding) != 0 {
			g.P("body.", prm.FullParameter, " = &", prm.Field.Message.GoIdent, "{}")
			renderQueryParameters(g, prm.Holding, filter)
		} else {
			if prm.IsList {
				switch prm.Type {
				case Int32Type:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]int32, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(vals[idx], 10, 32)")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = int32(p)")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case UInt32Type:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]uint32, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(vals[idx], 10, 32)")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = uint32(p)")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case Int64Type:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]int64, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(vals[idx], 10, 64)")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = p")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case UInt64Type:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]uint64, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(vals[idx], 10, 64)")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = p")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case Float32Type:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]float32, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(vals[idx], 32)")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = float32(p)")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case Float64Type:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]float64, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(vals[idx], 64)")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = p")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case BytesType:
					panic("bytes array currently not supported for query parameters")
				case EnumType:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]", prm.Field.Enum.GoIdent, ", len(vals))")
					g.P("for idx := range vals {")
					g.P("p, ok := ", prm.Field.Enum.GoIdent, "_value[vals[idx]]")
					g.P("if !ok {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = ", prm.Field.Enum.GoIdent, "(p)")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case StringType:
					g.P("body.", prm.FullParameter, "= ctx.QueryArray(\"", prm.RequestedKey, "\")")
				case BoolType:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")
					g.P("fin := make([]bool, len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", strconvPackage.Ident("ParseBool"), "(vals[idx])")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = p")
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				case TimeType:
					g.P("{")
					g.P("vals := ctx.QueryArray(\"", prm.RequestedKey, "\")")

					g.P("fin := make([]*", timepbPackage.Ident("Timestamp"), ", len(vals))")
					g.P("for idx := range vals {")
					g.P("p, err := ", timePackage.Ident("Parse"), "(", timePackage.Ident("RFC3339"), ", vals[idx])")
					g.P("if err != nil {")
					g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
					g.P("	return")
					g.P("}")
					g.P("fin[idx] = ", timepbPackage.Ident("New(p)"))
					g.P("}")
					g.P("body.", prm.FullParameter, "= fin")
					g.P("}")
				}
			} else {
				g.P("if val, ok := ctx.GetQuery(\"", prm.RequestedKey, "\"); ok {")

				if !prm.IsOptional {
					switch prm.Type {
					case Int32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= int32(p)")
					case UInt32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= uint32(p)")
					case Int64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case UInt64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case Float32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= float32(p)")
					case Float64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case BytesType:
						g.P("body.", prm.FullParameter, "= []byte(val)")
					case EnumType:
						g.P("p, ok := ", prm.Field.Enum.GoIdent, "_value[val]")
						g.P("if !ok {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, " = ", prm.Field.Enum.GoIdent, "(p)")
					case StringType:
						g.P("body.", prm.FullParameter, "= val")
					case BoolType:
						g.P("p, err := ", strconvPackage.Ident("ParseBool"), "(val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case TimeType:
						g.P("p, err := ", timePackage.Ident("Parse"), "(", timePackage.Ident("RFC3339"), ", val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= ", timepbPackage.Ident("New(p)"))
					}
				} else {
					switch prm.Type {
					case Int32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := int32(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case UInt32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := uint32(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case Int64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case UInt64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case Float32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := float32(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case Float64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case BytesType:
						g.P("body.", prm.FullParameter, "= []byte(val)")
					case EnumType:
						g.P("p, ok := ", prm.Field.Enum.GoIdent, "_value[val]")
						g.P("if !ok {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := ", prm.Field.Enum.GoIdent, "(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case StringType:
						g.P("body.", prm.FullParameter, "= &val")
					case BoolType:
						g.P("p, err := ", strconvPackage.Ident("ParseBool"), "(val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case TimeType:
						g.P("p, err := ", timePackage.Ident("Parse"), "(", timePackage.Ident("RFC3339"), ", val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= ", timepbPackage.Ident("New(p)"))
					}
				}

				g.P("} else {")
				if prm.IsOptional {
					g.P("body.", prm.FullParameter, " = nil")
				} else {
					g.P("ctx.Error(newMissingRequiredParametersError(\"", prm.RequestedKey, "\"))")
					g.P("return")
				}
				g.P("}")
			}
		}
	}
}

func renderPathParameters(
	g *protogen.GeneratedFile,
	prms []Parameter,
	filter []string,
) {
	for _, prm := range prms {
		found := false
		for idx := range filter {
			if prm.FullParameter == filter[idx] {
				found = true
				break
			}
		}

		if found {
			continue
		}
		if len(prm.Holding) != 0 {
			// g.P("body.", prm.FullParameter, " = &", prm.Field.Message.GoIdent, "{}") // TODO: issues may arise here
			renderPathParameters(g, prm.Holding, filter)
		} else {
			if !prm.IsPath {
				continue
			}

			if prm.IsList {
				panic("list not supported for path parameters")
			} else {
				g.P("if val := ctx.Param(\"", prm.RequestedKey, "\"); val != \"\" {")

				if !prm.IsOptional {
					switch prm.Type {
					case Int32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= int32(p)")
					case UInt32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= uint32(p)")
					case Int64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case UInt64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case Float32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= float32(p)")
					case Float64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case BytesType:
						g.P("body.", prm.FullParameter, "= []byte(val)")
					case EnumType:
						g.P("p, ok := ", prm.Field.Enum.GoIdent, "_value[val]")
						g.P("if !ok {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, " = ", prm.Field.Enum.GoIdent, "(p)")
					case StringType:
						g.P("body.", prm.FullParameter, "= val")
					case BoolType:
						g.P("p, err := ", strconvPackage.Ident("ParseBool"), "(val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= p")
					case TimeType:
						g.P("p, err := ", timePackage.Ident("Parse"), "(", timePackage.Ident("RFC3339"), ", val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= ", timepbPackage.Ident("New(p)"))
					}
				} else {
					switch prm.Type {
					case Int32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := int32(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case UInt32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := uint32(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case Int64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseInt"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case UInt64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseUint"), "(val, 10, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case Float32Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 32)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := float32(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case Float64Type:
						g.P("p, err := ", strconvPackage.Ident("ParseFloat"), "(val, 64)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case BytesType:
						g.P("body.", prm.FullParameter, "= []byte(val)")
					case EnumType:
						g.P("p, ok := ", prm.Field.Enum.GoIdent, "_value[val]")
						g.P("if !ok {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("x := ", prm.Field.Enum.GoIdent, "(p)")
						g.P("body.", prm.FullParameter, "= &x")
					case StringType:
						g.P("body.", prm.FullParameter, "= &val")
					case BoolType:
						g.P("p, err := ", strconvPackage.Ident("ParseBool"), "(val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= &p")
					case TimeType:
						g.P("p, err := ", timePackage.Ident("Parse"), "(", timePackage.Ident("RFC3339"), ", val)")
						g.P("if err != nil {")
						g.P("	ctx.Error(newUnparsableParameterError(\"", prm.RequestedKey, "\"))")
						g.P("	return")
						g.P("}")
						g.P("body.", prm.FullParameter, "= ", timepbPackage.Ident("New(p)"))
					}
				}

				g.P("} else {")
				if prm.IsOptional {
					g.P("body.", prm.FullParameter, " = nil")
				} else {
					g.P("ctx.Error(newMissingRequiredParametersError(\"", prm.RequestedKey, "\"))")
					g.P("return")
				}
				g.P("}")
			}
		}
	}
}

type info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type path struct{}

type openAPI3 struct {
	Info info `json:"info"`
}

// GenerateOpenAPI generates open api doc
func GenerateOpenAPI(
	srvs []Server,
	g *protogen.GeneratedFile,
	gjson *protogen.GeneratedFile,
	file *protogen.File,
) error {
	g.P("openapi: 3.0.3")
	g.P("info:")
	g.P("  title: ", file.Desc.Package())
	g.P("  version: ", "'1.0'") // TODO: better way to figure this out
	g.P("paths:")

	pathMap := map[string][]APIPath{}
	for _, svc := range srvs {
		for _, api := range svc.Paths {
			pathMap[api.OpenAPIPath] = append(pathMap[api.OpenAPIPath], api)
		}
	}

	for path, apis := range pathMap {
		g.P("  ", path, ":")
		for _, api := range apis {
			g.P("    ", strings.ToLower(api.HTTPMethod), ":")
			if len(api.Tags) != 0 {
				g.P("      tags:")
				for _, tag := range api.Tags {
					g.P("        - ", tag)
				}
			}
			g.P("      summary: ", api.Summary)         // TODO: escaping
			g.P("      description: ", api.Description) // TODO: escaping

			if api.HTTPMethod != "GET" && api.HTTPMethod != "DELETE" {
				g.P("      parameters:")
				renderParametersOpenAPI(g, api.Parameters, true, false)
				g.P("      requestBody:")
				g.P("        description: ", api.Method.Input.GoIdent.GoName)
				g.P("        content:")
				g.P("          application/json:")
				g.P("            schema:")
				g.P("              type: object")
				g.P("              properties:")
				// renderRequestBodyOpenAPI(g, api.Parameters, "")
				g.P(
					"              $ref: '#/components/schemas/",
					api.Method.Input.GoIdent.GoName,
					"'",
				)
				g.P("        required: true")
			} else {
				g.P("      parameters:")
				renderParametersOpenAPI(g, api.Parameters, false, false)
			}

			g.P("      responses:")
			g.P("        '200':")
			g.P("          description: ", api.Method.Output.GoIdent.GoName)
			g.P("          content: ")
			g.P("            application/json:")
			g.P("              schema:")
			g.P(
				"                $ref: '#/components/schemas/",
				api.Method.Output.GoIdent.GoName,
				"'",
			)

		}
	}

	g.P("components:")
	g.P("  schemas:")
	schemas := map[string]struct{}{}
	for _, svc := range srvs {
		for _, api := range svc.Paths {

			if err := generateOpenAPIComponentSchema(
				g,
				schemas,
				api.Method.Output,
				"",
				false,
			); err != nil {
				return err
			}

			generateOpenAPIComponentSchemaFromParameters(
				g,
				api.Method.Input.GoIdent.GoName,
				api.Parameters,
				false,
			)

			// if err := generateOpenAPIComponentSchema(
			// 	g,
			// 	schemas,
			// 	api.Method.Input,
			// ); err != nil {
			// 	return err
			// }

		}
	}

	bytes, err := g.Content()
	if err != nil {
		panic(err)
	}
	op := map[string]interface{}{}
	yaml.Unmarshal(bytes, &op)
	op = removeNulls(op)
	jsonraw, err := json.Marshal(op)
	if err != nil {
		panic(err)
	}
	gjson.P(string(jsonraw))
	return nil
}

func removeNulls(m map[string]interface{}) map[string]interface{} {
	for k := range m {
		if nm, ok := m[k].(map[string]interface{}); ok {
			m[k] = removeNulls(nm)
		}
		if m[k] == nil {
			delete(m, k)
		}
	}
	return m
}

func renderParametersOpenAPI(
	g *protogen.GeneratedFile,
	prms []Parameter,
	skipQP bool,
	skipUserContext bool,
) {
	for _, prm := range prms {
		if len(prm.Holding) != 0 {
			if skipUserContext && prm.RequestedKey == "userContext" {
				continue
			}
			renderParametersOpenAPI(g, prm.Holding, skipQP, false)
		} else {

			if prm.IsPath {
				g.P("        - in: path")
			} else {
				if skipQP {
					continue
				}
				g.P("        - in: query")
			}

			g.P("          name: ", prm.RequestedKey)
			if !prm.IsOptional {
				g.P("          required: true")
			} else {
				g.P("          required: false")
			}

			g.P("          schema:")
			prfx := ""
			if prm.IsList {
				g.P("            type: array")
				g.P("            items:")
				prfx = "  "
			}

			switch prm.Type {
			case Int32Type:
				g.P(prfx, "            type: integer")
				g.P(prfx, "            format: int32")
				g.P(prfx, "            example: 1")
			case UInt32Type:
				g.P(prfx, "            type: integer")
				g.P(prfx, "            format: int32")
				g.P(prfx, "            example: 1")
			case Int64Type:
				g.P(prfx, "            type: integer")
				g.P(prfx, "            format: int64")
				g.P(prfx, "            example: 1")
			case UInt64Type:
				g.P(prfx, "            type: integer")
				g.P(prfx, "            format: int64")
				g.P(prfx, "            example: 1")
			case Float32Type:
				g.P(prfx, "            type: number")
				g.P(prfx, "            format: float")
				g.P(prfx, "            example: 1.0")
			case Float64Type:
				g.P(prfx, "            type: number")
				g.P(prfx, "            format: double")
				g.P(prfx, "            example: 1.0")
			case BytesType:
				g.P(prfx, "            type: string")
				g.P(prfx, "            format: byte")
				g.P(prfx, "            example: sample")
			case EnumType:
				g.P(prfx, "            example: 1")
				g.P(prfx, "            type: string")

				values := prm.Field.Enum.Values[0].Desc.Name()
				for i := 1; i < len(prm.Field.Enum.Values); i++ {
					values = values + ", " + prm.Field.Enum.Values[i].Desc.Name()
				}
				g.P(prfx, "            enum: [", values, "]")
			case StringType:
				g.P(prfx, "            type: string")
				g.P(prfx, "            example: sample")
			case BoolType:
				g.P(prfx, "            type: boolean")
				g.P(prfx, "            example: false")
			case TimeType:
				g.P(prfx, "            type: string")
				g.P(prfx, "            format: date-time")
				g.P(prfx, "            example: '2017-07-21T17:32:28Z'")
			}
		}
	}
}

func renderRequestBodyOpenAPI(
	g *protogen.GeneratedFile,
	prms []Parameter,
	prefix string,
) {
	for _, prm := range prms {
		if prm.IsPath {
			continue
		}

		g.P(prefix, "                ", prm.Field.Desc.JSONName(), ":")
		prfx := ""
		if prm.IsList {
			g.P(prefix, "                     type: array")
			g.P(prefix, "                     items:")
			prfx = "  "
		}

		if len(prm.Holding) != 0 {
			g.P(prefix, prfx, "                     schema:")
			g.P(prefix, prfx, "                       type: object")
			g.P(prefix, prfx, "                       properties:")
			renderRequestBodyOpenAPI(g, prm.Holding, prefix+prfx+"  ")
		} else {
			switch prm.Type {
			case Int32Type:
				g.P(prefix, prfx, "                     type: integer")
				g.P(prefix, prfx, "                     format: int32")
				g.P(prefix, prfx, "                     example: 1")
			case UInt32Type:
				g.P(prefix, prfx, "                     type: integer")
				g.P(prefix, prfx, "                     format: int32")
				g.P(prefix, prfx, "                     example: 1")
			case Int64Type:
				g.P(prefix, prfx, "                     type: integer")
				g.P(prefix, prfx, "                     format: int64")
				g.P(prefix, prfx, "                     example: 1")
			case UInt64Type:
				g.P(prefix, prfx, "                     type: integer")
				g.P(prefix, prfx, "                     format: int64")
				g.P(prefix, prfx, "                     example: 1")
			case Float32Type:
				g.P(prefix, prfx, "                     type: number")
				g.P(prefix, prfx, "                     format: float")
				g.P(prefix, prfx, "                     example: 1.0")
			case Float64Type:
				g.P(prefix, prfx, "                     type: number")
				g.P(prefix, prfx, "                     format: double")
				g.P(prefix, prfx, "                     example: 1.0")
			case BytesType:
				g.P(prefix, prfx, "                     type: string")
				g.P(prefix, prfx, "                     format: byte")
				g.P(prefix, prfx, "                     example: sample")
			case EnumType:
				g.P(prefix, prfx, "                     type: integer")
				g.P(prefix, prfx, "                     format: int32")
				g.P(prefix, prfx, "                     example: 1")
				g.P(prefix, prfx, "                     type: string")

				values := prm.Field.Enum.Values[0].Desc.Name()
				for i := 1; i < len(prm.Field.Enum.Values); i++ {
					values = values + ", " + prm.Field.Enum.Values[i].Desc.Name()
				}
				g.P(prefix, prfx, "                     enum: [", values, "]")
			case StringType:
				g.P(prefix, prfx, "                     type: string")
				g.P(prefix, prfx, "                     example: sample")
			case BoolType:
				g.P(prefix, prfx, "                     type: boolean")
				g.P(prefix, prfx, "                     example: false")
			case TimeType:
				g.P(prefix, prfx, "                     type: string")
				g.P(prefix, prfx, "                     format: date-time")
				g.P(prefix, prfx, "                     example: '2017-07-21T17:32:28Z'")
			}
		}
	}
}

func ToPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}

func generateOpenAPIComponentSchema(
	g *protogen.GeneratedFile,
	s map[string]struct{},
	m *protogen.Message,
	keyPrefix string,
	skipUserContext bool,
) error {
	foundMessages := []*protogen.Message{}
	if _, ok := s[keyPrefix+m.GoIdent.GoName]; !ok {
		s[keyPrefix+m.GoIdent.GoName] = struct{}{}
		g.P("    ", keyPrefix+m.GoIdent.GoName, ":")
		g.P("      type: object")
		g.P("      properties:")

		for _, fld := range m.Fields {
			field := fld

			if skipUserContext && field.Desc.JSONName() == "userContext" {
				continue
			}
			g.P("        ", field.Desc.JSONName(), ":")

			prfx := ""
			if field.Desc.IsMap() {
				for _, f := range field.Message.Fields {
					if f.Desc.JSONName() == "value" {
						g.P("          type: object")
						g.P("          additionalProperties:")
						prfx = "  "
						field = f
					}
				}
			}
			if field.Desc.IsList() {
				g.P("          type: array")
				g.P("          items:")
				prfx = "  "
			}

			kind := field.Desc.Kind()
			switch kind {
			case protoreflect.BoolKind:
				g.P(prfx, "          type: boolean")
				g.P(prfx, "          example: false")
			case protoreflect.EnumKind: // TODO
				g.P(prfx, "          type: string")

				values := field.Enum.Values[0].Desc.Name()
				for i := 1; i < len(field.Enum.Values); i++ {
					values = values + ", " + field.Enum.Values[i].Desc.Name()
				}
				g.P(prfx, "          enum: [", values, "]")
			case protoreflect.Int32Kind,
				protoreflect.Sint32Kind,
				protoreflect.Uint32Kind:
				g.P(prfx, "          type: integer")
				g.P(prfx, "          format: int32")
				g.P(prfx, "          example: 1")
			case protoreflect.Int64Kind,
				protoreflect.Sint64Kind,
				protoreflect.Uint64Kind:
				g.P(prfx, "          type: integer")
				g.P(prfx, "          format: int64")
				g.P(prfx, "          example: 1")
			case protoreflect.Sfixed32Kind,
				protoreflect.Fixed32Kind,
				protoreflect.FloatKind:
				g.P(prfx, "          type: number")
				g.P(prfx, "          format: float")
				g.P(prfx, "          example: 1.0")
			case protoreflect.Sfixed64Kind,
				protoreflect.Fixed64Kind,
				protoreflect.DoubleKind:
				g.P(prfx, "          type: number")
				g.P(prfx, "          format: double")
				g.P(prfx, "          example: 1.0")
			case protoreflect.StringKind:
				g.P(prfx, "          type: string")
				g.P(prfx, "          example: sample")
			case protoreflect.BytesKind:
				g.P(prfx, "          type: string")
				g.P(prfx, "          format: byte")
				g.P(prfx, "          example: false")
			case protoreflect.MessageKind:
				if field.Message.Desc.FullName() == "google.protobuf.Timestamp" {
					g.P(prfx, "          type: string")
					g.P(prfx, "          format: date-time")
					g.P(prfx, "          example: '2017-07-21T17:32:28Z'")
				} else if field.Message.Desc.FullName() == "google.protobuf.Struct" {
					g.P(prfx, "          type: object")
				} else {
					foundMessages = append(foundMessages, field.Message)
					g.P(prfx, "          $ref: '#/components/schemas/", keyPrefix+field.Message.GoIdent.GoName, "'")
				}

			case protoreflect.GroupKind: // TODO
			}
		}
	}

	for _, found := range foundMessages {
		generateOpenAPIComponentSchema(g, s, found, keyPrefix, false)
	}
	return nil
}

func generateOpenAPIComponentSchemaFromParameters(
	g *protogen.GeneratedFile,
	key string,
	prms []Parameter,
	skipUserContext bool,
) {
	foundMessages := []Parameter{}
	g.P("    ", key, ":")
	g.P("      type: object")
	g.P("      properties:")
	for idx := range prms {

		if prms[idx].IsPath {
			continue
		}

		field := prms[idx].Field

		if skipUserContext && field.Desc.JSONName() == "userContext" {
			continue
		}

		g.P("        ", field.Desc.JSONName(), ":")

		prfx := ""
		if field.Desc.IsMap() {
			for _, f := range field.Message.Fields {
				if f.Desc.JSONName() == "value" {
					g.P("          type: object")
					g.P("          additionalProperties:")
					prfx = "  "
					field = f
				}
			}
		}
		if field.Desc.IsList() {
			g.P("          type: array")
			g.P("          items:")
			prfx = "  "
		}

		kind := field.Desc.Kind()
		switch kind {
		case protoreflect.BoolKind:
			g.P(prfx, "          type: boolean")
			g.P(prfx, "          example: false")
		case protoreflect.EnumKind: // TODO
			g.P(prfx, "          type: string")

			values := field.Enum.Values[0].Desc.Name()
			for i := 1; i < len(field.Enum.Values); i++ {
				values = values + ", " + field.Enum.Values[i].Desc.Name()
			}
			g.P(prfx, "          enum: [", values, "]")
		case protoreflect.Int32Kind,
			protoreflect.Sint32Kind,
			protoreflect.Uint32Kind:
			g.P(prfx, "          type: integer")
			g.P(prfx, "          format: int32")
			g.P(prfx, "          example: 1")
		case protoreflect.Int64Kind,
			protoreflect.Sint64Kind,
			protoreflect.Uint64Kind:
			g.P(prfx, "          type: integer")
			g.P(prfx, "          format: int64")
			g.P(prfx, "          example: 1")
		case protoreflect.Sfixed32Kind,
			protoreflect.Fixed32Kind,
			protoreflect.FloatKind:
			g.P(prfx, "          type: number")
			g.P(prfx, "          format: float")
			g.P(prfx, "          example: 1.0")
		case protoreflect.Sfixed64Kind,
			protoreflect.Fixed64Kind,
			protoreflect.DoubleKind:
			g.P(prfx, "          type: number")
			g.P(prfx, "          format: double")
			g.P(prfx, "          example: 1.0")
		case protoreflect.StringKind:
			g.P(prfx, "          type: string")
			g.P(prfx, "          example: sample")
		case protoreflect.BytesKind:
			g.P(prfx, "          type: string")
			g.P(prfx, "          format: byte")
			g.P(prfx, "          example: false")
		case protoreflect.MessageKind:
			if field.Message.Desc.FullName() == "google.protobuf.Timestamp" {
				g.P(prfx, "          type: string")
				g.P(prfx, "          format: date-time")
				g.P(prfx, "          example: '2017-07-21T17:32:28Z'")
			} else if field.Message.Desc.FullName() == "google.protobuf.Struct" {
				g.P(prfx, "          type: object")
			} else {
				foundMessages = append(foundMessages, prms[idx])
				g.P(prfx, "          $ref: '#/components/schemas/", key, field.GoName, "'")
			}

		case protoreflect.GroupKind: // TODO
		}
	}

	for _, found := range foundMessages {
		generateOpenAPIComponentSchemaFromParameters(
			g,
			key+found.Field.GoName,
			found.Holding,
			false,
		)
	}
}

// GeneratePermisionMaps generates permission map
func GeneratePermisionMaps(
	srvs []Server,
	g *protogen.GeneratedFile,
) error {
	resources := map[string]interface{}{}

	for _, srv := range srvs {
		cnqs := map[string]interface{}{}
		for _, rpc := range srv.Paths {
			rfs := map[string]interface{}{}
			rfs["Roles"] = rpc.Roles
			rfs["Features"] = rpc.Features
			cnqs[rpc.Method.Input.GoIdent.GoName] = rfs
		}
		resources[srv.Service.GoName] = cnqs
	}

	jsonraw, err := json.Marshal(resources)
	if err != nil {
		panic(err)
	}
	g.P(string(jsonraw))

	return nil
}
