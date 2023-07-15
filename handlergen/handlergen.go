// Package main main
package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BetaLixT/golang-tooling/handlergen/handlers"
	"go.uber.org/zap"
)

const (
	GRPC_CONTRACT_DIRECTORY = "./pkg/app/server/contracts"
	HANDLER_DIRECTORY       = "./pkg/app/server/handlers"
)

func main() {
	var pkg *build.Package
	var err error
	lgr, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	dat, err := os.ReadFile("./go.mod")
	if err != nil {
		panic(err)
	}
	mod := string(dat)
	modLines := strings.Split(mod, "\n")
	if len(modLines) == 0 {
		panic("invalid mod file, invalid line count")
	}

	moduleName := strings.TrimSpace(strings.TrimPrefix(modLines[0], "module"))

	pkg, err = build.ImportDir(GRPC_CONTRACT_DIRECTORY, 0)
	if err != nil {
		panic(fmt.Errorf("couldn't find package %v", err))
	}

	fset := token.NewFileSet() // share one fset across the whole package
	var files []string

	for _, file := range pkg.GoFiles {
		if strings.Contains(file, "grpc.pb") {
			files = append(files, file)
		}
	}
	for _, file := range files {
		f, err := parser.ParseFile(fset, filepath.Join(pkg.Dir, file), nil, parser.ParseComments)
		if err != nil {
			continue
		}

		cmap := ast.NewCommentMap(fset, f, f.Comments)

		ispecs := []ServerInterfaceSpec{}
		for _, decl := range f.Decls {
			decl, ok := decl.(*ast.GenDecl)
			if !ok || decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				spec := spec.(*ast.TypeSpec)

				if !IsGrpcServer(spec.Name.Name) {
					continue
				}

				ispecs = append(
					ispecs,
					ServerInterfaceSpec{
						TypeSpec:   spec,
						CommentMap: cmap.Filter(decl),
					},
				)
			}
		}

		hndlrPackage, err := build.ImportDir(HANDLER_DIRECTORY, 0)
		files := map[string]struct{}{}
		if err == nil {
			for _, file := range hndlrPackage.GoFiles {
				files[file] = struct{}{}
			}
		}

		// handlerBuilders := []handlers.HandlerBuilder{}
		for _, ispec := range ispecs {
			bldr := handlers.NewBuilder(
				pkg,
				ispec.TypeSpec,
				moduleName,
				lgr,
			)

			// Find if handler structure already exists
			if _, ok := files[bldr.GetHandlerFileName()]; ok {
				bldr.FileExists(true)
				f, err := parser.ParseFile(
					fset,
					filepath.Join(hndlrPackage.Dir, bldr.GetHandlerFileName()),
					nil,
					parser.ParseComments,
				)
				if err != nil {
					lgr.Warn(
						"failed to parse file",
						zap.String("fileName", bldr.GetHandlerFileName()),
					)
					continue
				}

				for _, decl := range f.Decls {
					decl, ok := decl.(*ast.GenDecl)
					if !ok || decl.Tok != token.TYPE {
						continue
					}
					for _, spec := range decl.Specs {
						spec := spec.(*ast.TypeSpec)

						lgr.Info("found type", zap.String("typeName", spec.Name.Name))
						if bldr.GetHandlerStructName() == spec.Name.Name {
							bldr.RegisterStructure(hndlrPackage, spec)
							break
						}
					}
				}

			} else {
				bldr.FileExists(false)
			}
			bldr.Build()
			// Generate handler structure and functions if nothing exists
			// Create functions that were not implemented if structure exists
		}
	}
}

type ServerInterfaceSpec struct {
	TypeSpec   *ast.TypeSpec
	CommentMap ast.CommentMap
}

func IsGrpcServer(name string) bool {
	if strings.HasPrefix(name, "Unsafe") {
		return false
	}
	if strings.HasPrefix(name, "Unimplemented") {
		return false
	}
	return strings.HasSuffix(name, "Server")
}

func ToPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}

func ToPublicName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}
