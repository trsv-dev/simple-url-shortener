// Package noexitmain содержит анализатор,
// запрещающий прямой вызов os.Exit внутри функции main пакета main.
package noexitmain

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer Экспортируемая переменная для multichecker.
var Analyzer = &analysis.Analyzer{
	Name: "OsExitAnalyzer",
	Doc:  "check for direct call os.Exit in main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// пропускаем файлы не из пакета main
		if pass.Pkg.Name() != "main" {
			continue
		}

		// инспектируем только тело функции
		for _, decl := range file.Decls {
			filePath := pass.Fset.Position(file.Package).Filename
			if strings.Contains(filePath, "go-build") {
				continue
			}

			fd, ok := decl.(*ast.FuncDecl)
			// ищем только функцию main без receiver (не метод!)
			if !ok || fd.Recv != nil || fd.Name.Name != "main" {
				continue
			}

			ast.Inspect(fd.Body, func(node ast.Node) bool {
				// это вызов функции? (например, os.Exit(1))
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				// это вызов через точку? (package.Func, а не просто Func)
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// можем безопасно проверять: это os.Exit?
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "avoid using os.Exit directly in main function")
				}

				return true
			})
		}
	}

	return nil, nil
}
