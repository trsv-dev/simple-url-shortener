package main

import (
	"strings"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/gostaticanalysis/nilerr"
	"github.com/trsv-dev/simple-url-shortener/cmd/shortener/staticlint/noexitmain"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	checks := []*analysis.Analyzer{
		printf.Analyzer,
		structtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		ineffassign.Analyzer,
		nilerr.Analyzer,
		noexitmain.Analyzer,
	}

	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			checks = append(checks, v.Analyzer)
		}
	}

	for _, v := range stylecheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "ST") {
			checks = append(checks, v.Analyzer)
		}
	}

	multichecker.Main(checks...)
}
