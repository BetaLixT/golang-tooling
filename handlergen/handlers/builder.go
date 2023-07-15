// Package handlers generates handlers
package handlers

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

// NewBuilder builds, I hate revive....
func NewBuilder(
	interfacePackage *build.Package,
	interfaceType *ast.TypeSpec,
	module string,
	lgr *zap.Logger,
) *handlerBuilder {
	intr := interfaceType.Type.(*ast.InterfaceType)

	bn := toPrivateName(strings.ReplaceAll(
		interfaceType.Name.Name,
		"Server",
		"",
	))

	return &handlerBuilder{
		baseName:         bn,
		interfaceName:    interfaceType.Name.Name,
		interfacePackage: interfacePackage,
		interfaceType:    intr,
		isFileExists:     false,
		structureType:    nil,
		lgr:              lgr,
		module:           module,
	}
}

type handlerBuilder struct {
	baseName         string
	isFileExists     bool
	interfaceName    string
	interfacePackage *build.Package
	interfaceType    *ast.InterfaceType
	structurePackage *build.Package
	structureType    *ast.StructType
	lgr              *zap.Logger
	module           string
}

func (app *handlerBuilder) FileExists(val bool) {
	app.isFileExists = val
}

func (app *handlerBuilder) GetModuleName() string {
	return app.module
}

func (app *handlerBuilder) RegisterStructure(
	p *build.Package,
	t *ast.TypeSpec,
) {
	app.structureType = t.Type.(*ast.StructType)
	app.structurePackage = p
}

func (app *handlerBuilder) GetHandlerFileName() string {
	return app.baseName + ".go"
}

func (app *handlerBuilder) GetHandlerStructName() string {
	return toPublicName(app.baseName) + "Handler"
}

func (app *handlerBuilder) Build() {
	fullFilePath := "pkg/app/server/handlers/" + app.GetHandlerFileName()
	useCasePackagePath := "pkg/domain/usecases/"
	// - parsing interface for functions
	interfaceFunctions := app.parseInterfaceFunctions()

	structFunctions := app.parseStructFunctions()

	src := source{}
	useCaseSrcs := map[string]source{}
	if !app.isFileExists {
		app.lgr.Info(
			"creating file",
			zap.String("fileName", app.GetHandlerFileName()),
		)
		if _, err := os.Stat("pkg/app/server/handlers"); os.IsNotExist(err) {
			os.MkdirAll("pkg/app/server/handlers", 0770)
		} else if err != nil {
			panic(err)
		}

		// file, err := os.Create(fullFilePath)
		// if err != nil {
		// 	panic(err)
		// }
		// file.Close()
		src.Println("package handlers")
		src.Println("import (")
		src.Println("\"context\"")
		src.Println("\"fmt\"")
		src.Println("\"time\"")
		src.Println("\"github.com/betalixt/gorr\"")
		src.Println("\"go.uber.org/zap\"")
		src.Println("\"" + app.GetModuleName() + "/pkg/app/server/common\"")
		src.Println("srvcontracts \"" + app.GetModuleName() + "/pkg/app/server/contracts\"")
		src.Println("\"" + app.GetModuleName() + "/pkg/domain/base/cntxt\"")
		src.Println("\"" + app.GetModuleName() + "/pkg/domain/base/logger\"")
		src.Println("\"" + app.GetModuleName() + "/pkg/domain/contracts\"")
		src.Println("\"" + app.GetModuleName() + "/pkg/domain/usecases\"")
		src.Println(")")
		// create file
	}
	if app.structureType == nil {
		// create structure
		src.Println("type ", app.GetHandlerStructName(), " struct {")
		src.Println("srvcontracts.Unimplemented", app.interfaceName)
		src.Println("lgrf logger.IFactory")
		src.Println("uscs *usecases.UseCases")
		src.Println("}")

		src.Println(
			"var _ srvcontracts.",
			app.interfaceName,
			" = (*",
			app.GetHandlerStructName(),
			")(nil)",
		)

		src.Println("func New", app.GetHandlerStructName(), "(")
		src.Println("lgrf logger.IFactory,")
		src.Println("uscs *usecases.UseCases,")
		src.Println(") *", app.GetHandlerStructName(), "{")
		src.Println("return &", app.GetHandlerStructName(), "{")
		src.Println("lgrf: lgrf,")
		src.Println("uscs: uscs,")
		src.Println("}}")
	}

	for _, fn := range interfaceFunctions {
		// TODO: it's a spike to get it working, optimizations probs possible
		impl := false
		for _, sfn := range structFunctions {
			if sfn.Equals(&fn) {
				impl = true
				break
			}
		}
		if impl {
			continue
		}

		inprm := "cmd"
		if strings.Contains(fn.GetParameters()[1], "Query") {
			inprm = "qry"
		}
		src.Println("func (h *", app.GetHandlerStructName(), ") ", fn.GetFunctionName(), "(")
		src.Println("c ", fn.GetParameters()[0], ",")
		src.Println(inprm, " ", fn.GetParameters()[1], ",")
		src.Println(") (res ", fn.GetReturns()[0], ", err ", fn.GetReturns()[1], ") {")

		src.Println("ctx, ok := c.(cntxt.IContext)")
		src.Println("if !ok {")
		src.Println("	return nil, common.NewInvalidContextProvidedToHandlerError()")
		src.Println("}")

		src.Println("ctx.SetTimeout(2 * time.Minute)")
		src.Println("lgr := h.lgrf.Create(ctx)")
		src.Println("lgr.Info(")
		src.Println("	\"handling\",")
		src.Println("	zap.Any(\"", inprm, "\", ", inprm, "),")
		src.Println(")")
		src.Println("defer func() {")
		src.Println("	if r := recover(); r != nil {")
		src.Println("		var ok bool")
		src.Println("		err, ok = r.(error)")
		src.Println("		if !ok {")
		src.Println("			err = gorr.NewUnexpectedError(fmt.Errorf(\"%v\", r))")
		src.Println("			lgr.Error(")
		src.Println("				\"root panic recovered handling request\",")
		src.Println("				zap.Any(\"panic\", r),")
		src.Println("				zap.Stack(\"stack\"),")
		src.Println("			)")
		src.Println("		} else {")
		src.Println("			lgr.Error(")
		src.Println("				\"root panic recovered handling request\",")
		src.Println("				zap.Error(err),")
		src.Println("				zap.Stack(\"stack\"),")
		src.Println("			)")
		src.Println("		}")
		src.Println("ctx.Cancel()")
		src.Println("	}")
		src.Println("	if err != nil {")
		src.Println("	if _, ok := err.(*gorr.Error); !ok {")
		src.Println("		err = gorr.NewUnexpectedError(err)")
		src.Println("	}}")
		src.Println("}()")
		src.Println("res, err = h.uscs." + toPublicName(app.baseName) + fn.GetFunctionName() + "(")
		src.Println("	ctx,")
		src.Println("	", inprm, ",")
		src.Println(")")
		src.Println("if err != nil {")
		src.Println("	lgr.Error(")
		src.Println("		\"command handling failed\",")
		src.Println("		zap.Error(err),")
		src.Println("	)")
		src.Println("}")
		src.Println("ctx.Cancel()")
		src.Println("return")
		src.Println("}")
		src.Println(" ")

		ucSrc := source{}
		ucSrc.Println("package usecases")
		ucSrc.Println("import (")
		ucSrc.Println("\"github.com/betalixt/gorr\"")
		ucSrc.Println("\"go.uber.org/zap\"")
		// ucSrc.Println("\"" + app.GetModuleName() + "/pkg/app/server/common\"")
		ucSrc.Println("\"" + app.GetModuleName() + "/pkg/domain/contracts\"")
		ucSrc.Println("\"" + app.GetModuleName() + "/pkg/domain/base/cntxt\"")
		ucSrc.Println(")")
		ucSrc.Println("// ", toPublicName(app.baseName)+fn.GetFunctionName(), " usecase for")
		ucSrc.Println("func (u *UseCases) ", toPublicName(app.baseName)+fn.GetFunctionName(), "(")
		ucSrc.Println("ctx cntxt.IUseCaseContext,")
		ucSrc.Println(inprm, " ", fn.GetParameters()[1], ",")
		ucSrc.Println(") (res ", fn.GetReturns()[0], ", err ", fn.GetReturns()[1], ") {")

		ucSrc.Println("lgr := u.lgrf.Create(ctx)")
		ucSrc.Println("lgr.Info(")
		ucSrc.Println("	\"running usecase\",")
		ucSrc.Println("	zap.String(\"resource\", ", "\""+app.GetHandlerStructName()+"\"),")
		ucSrc.Println("	zap.String(\"usecase\", ", "\""+fn.GetFunctionName()+"\"),")
		ucSrc.Println(")")
		ucSrc.Println("")
		ucSrc.Println("err = gorr.NewNotImplemented()")
		ucSrc.Println("if err != nil {")
		ucSrc.Println("lgr.Error(")
		ucSrc.Println("\"error while running usecase\",")
		ucSrc.Println("zap.Error(err),")
		ucSrc.Println(")")
		ucSrc.Println("}")

		ucSrc.Println("return")
		ucSrc.Println("}")

		useCaseFileName := app.baseName + fn.GetFunctionName()
		useCaseSrcs[useCaseFileName] = ucSrc
	}

	// functionsToImplement := []Func{}

	// fmt.Print(src.sourceCode)

	f, err := os.OpenFile(fullFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	if _, err = f.Write(src.Source()); err != nil {
		f.Close()
		panic(err)
	}
	f.Close()

	for key := range useCaseSrcs {
		f, err := os.OpenFile(useCasePackagePath+key+".go", os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}

		ucsrc := useCaseSrcs[key]
		if _, err = f.Write(ucsrc.Source()); err != nil {
			f.Close()
			panic(err)
		}
		f.Close()
	}
	// fmt.Printf("\n%v\n", interfaceFunctions)
	// fmt.Printf("\n%v\n", structFunctions)
}

func (app *handlerBuilder) parseInterfaceFunctions() (interfaceFunctions []Func) {
	interfaceFunctions = []Func{}
	for _, meth := range app.interfaceType.Methods.List {
		// checking if embed, there shouldn't be any for Server interfaces

		if len(meth.Names) != 0 {

			// app.lgr.Info(meth.Names[0].Name)
			if strings.HasPrefix(meth.Names[0].Name, "mustEmbedUnimplemented") {
				continue
			}

			fn := NewFunc()
			fn.SetName(meth.Names[0].Name)
			methCast := meth.Type.(*ast.FuncType)

			// handling parameters
			if methCast.Params != nil {
				for _, p := range methCast.Params.List {
					// app.lgr.Info("parameter", zap.Any("param", p))
					fullType := fullType(app.interfacePackage, p.Type)
					// handles single type multiple parameters
					if len(p.Names) == 0 {
						fn.AddParameter(fullType)
					} else {
						for range p.Names {
							fn.AddParameter(fullType)
						}
					}
				}
			}

			// handling returns
			if methCast.Results != nil {
				for _, p := range methCast.Results.List {
					// app.lgr.Info("res", zap.Any("res", p))
					fullType := fullType(app.interfacePackage, p.Type)
					// handles single type multiple returns
					if len(p.Names) == 0 {
						fn.AddReturn(fullType)
					} else {
						for range p.Names {
							fn.AddReturn(fullType)
						}
					}

				}
			}

			if len(fn.GetParameters()) != 2 {
				continue
			}
			if len(fn.GetReturns()) != 2 {
				continue
			}
			interfaceFunctions = append(interfaceFunctions, *fn)
		}
	}
	return
}

func (app *handlerBuilder) parseStructFunctions() (structFunctions []Func) {
	expectedReceiverStruct := "*handlers." + app.GetHandlerStructName()
	if app.structureType != nil {

		fset := token.NewFileSet()
		f, err := parser.ParseFile(
			fset,
			filepath.Join(app.structurePackage.Dir, app.GetHandlerFileName()),
			nil,
			parser.ParseComments,
		)
		if err != nil {
			panic(err)
		}

		for _, decl := range f.Decls {
			if decl == nil {
				continue
			}
			meth, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if meth.Recv == nil || len(meth.Recv.List) == 0 {
				continue
			}

			receiverStruct := fullType(app.structurePackage, meth.Recv.List[0].Type)
			if expectedReceiverStruct != receiverStruct {
				continue
			}

			fn := NewFunc()
			fn.SetName(meth.Name.Name)

			methCast := meth.Type

			// handling parameters
			if methCast.Params != nil {
				for _, p := range methCast.Params.List {
					// app.lgr.Info("parameter", zap.Any("param", p))
					fullType := fullType(app.structurePackage, p.Type)
					// handles single type multiple parameters
					if len(p.Names) == 0 {
						fn.AddParameter(fullType)
					} else {
						for range p.Names {
							fn.AddParameter(fullType)
						}
					}
				}
			}

			// handling returns
			if methCast.Results != nil {
				for _, p := range methCast.Results.List {
					// app.lgr.Info("res", zap.Any("res", p))
					fullType := fullType(app.structurePackage, p.Type)
					// handles single type multiple returns
					if len(p.Names) == 0 {
						fn.AddReturn(fullType)
					} else {
						for range p.Names {
							fn.AddReturn(fullType)
						}
					}

				}
			}
			structFunctions = append(structFunctions, *fn)
		}
	}
	return
}

func toPrivateName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToLower(inr[0])
	out = string(inr)
	return
}

func toPublicName(in string) (out string) {
	inr := []rune(in)
	inr[0] = unicode.ToUpper(inr[0])
	out = string(inr)
	return
}

func fullType(pkg *build.Package, e ast.Expr) string {
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			// Using typeSpec instead of IsExported here would be
			// more accurate, but it'd be crazy expensive, and if
			// the type isn't exported, there's no point trying
			// to implement it anyway.
			if n.IsExported() {
				n.Name = pkg.Name + "." + n.Name
			}
		case *ast.SelectorExpr:
			return false
		}
		return true
	})
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}
