// Package pkg package
package pkg

import (
	"encoding/json"
	"strings"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

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
	for _, svc := range srvs {
		for _, api := range svc.Paths {
			g.P("  ", api.Path, ":")
			g.P("    ", strings.ToLower(api.HTTPMethod), ":")
			if len(api.Tags) != 0 {
				g.P("      tags:")
				for _, tag := range api.Tags {
					g.P("        - ", tag)
				}
			}
			g.P("      summary: ", api.Summary)         // TODO: escaping
			g.P("      description: ", api.Description) // TODO: escaping
			g.P("      requestBody:")
			g.P("        description: ", api.Method.Input.GoIdent.GoName)
			g.P("        content:")
			g.P("          application/json:")
			g.P("            schema:")
			g.P("              $ref: '#/components/schemas/", api.Method.Input.GoIdent.GoName, "'")
			g.P("        required: true")
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
			); err != nil {
				return err
			}

			if err := generateOpenAPIComponentSchema(
				g,
				schemas,
				api.Method.Input,
			); err != nil {
				return err
			}

		}
	}

	bytes, err := g.Content()
	if err != nil {
		panic(err)
	}
	op := map[string]interface{}{}
	yaml.Unmarshal(bytes, &op)
	jsonraw, err := json.Marshal(op)
	if err != nil {
		panic(err)
	}
	gjson.P(string(jsonraw))
	return nil
}

func ToPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}

func ToPublicName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToUpper(inr[0])
	out = string(inr)
	return
}

func generateOpenAPIComponentSchema(
	g *protogen.GeneratedFile,
	s map[string]struct{},
	m *protogen.Message,
) error {
	foundMessages := []*protogen.Message{}
	if _, ok := s[m.GoIdent.GoName]; !ok {
		s[m.GoIdent.GoName] = struct{}{}
		g.P("    ", m.GoIdent.GoName, ":")
		g.P("      type: object")
		g.P("      properties:")
		for _, fld := range m.Fields {
			field := fld
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
					g.P(prfx, "          $ref: '#/components/schemas/", field.Message.GoIdent.GoName, "'")
				}

			case protoreflect.GroupKind: // TODO
			}
		}
	}

	for _, found := range foundMessages {
		generateOpenAPIComponentSchema(g, s, found)
	}
	return nil
}
