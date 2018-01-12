package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	// +=========+
	// |  Flags  |
	// +=========+
	// - Variables
	var instFilePath string
	flag.StringVar(&instFilePath, "instFilePath", "", "Specify path to instrumented Go file")
	var tracerPath string
	flag.StringVar(&tracerPath, "tracerPath", "../traceInst/tracer", "Specifies path to tracer package - relative to code file location")
	var function string
	flag.StringVar(&function, "function", "main", "Specifies name of function to trace")

	// - Program Execution
	var overwrite bool
	flag.BoolVar(&overwrite, "overwrite", false, "Overwrites the edited the code file")

	// - Printables
	var printFuncs bool
	flag.BoolVar(&printFuncs, "printFuncs", false, "Prints list of functions")
	var printImports bool
	flag.BoolVar(&printImports, "printImports", false, "Prints list of imports")
	var printOutputCode bool
	flag.BoolVar(&printOutputCode, "printOutputCode", false, "Prints the generated output code")

	flag.Parse()

	// +--------------+
	// |  Code start  |
	// +--------------+
	// Checks for need to run
	if instFilePath == "" {
		fmt.Println("instFilePath not specified\n")
		flag.Usage()
		return
	}

	if !(printFuncs || printImports || printOutputCode || overwrite) {
		fmt.Println("No arguments specified\n")
		flag.Usage()
		return
	}

	// Parses INST_FILEPATH into ast
	fset := token.NewFileSet()
	astf, err := parser.ParseFile(fset, instFilePath, nil, 0)
	if err != nil {
		panic(err)
	}

	importRedundant := false
	tracerPath = "\"" + tracerPath + "\""
	// fmt.Println("Imports and Functions")
	// fmt.Println("=====================")
	noOfGenDecls := 0
	for i := range astf.Decls {
		switch x := astf.Decls[i].(type) {
		// Imports
		case *ast.GenDecl:
			// Appends "tracer" package to imports
			if printOutputCode || overwrite {
				for _, s := range x.Specs {
					switch s.(type) {
					case *ast.ImportSpec:
						if s.(*ast.ImportSpec).Path.Value == tracerPath {
							importRedundant = true
							break
						}
					}
				}
			}

			// Prints each import name
			if printImports {
				for _, s := range x.Specs {
					if is, ok := s.(*ast.ImportSpec); ok {
						fmt.Println("- Import:", is.Path.Value)
					}
				}
			}
			// fmt.Println("GenDecl:", x)
			noOfGenDecls++
		// Functions
		case *ast.FuncDecl:
			// Prints each function name
			if printFuncs {
				fmt.Println("- Function:", x.Name)
			}
			// Adds tracer.Start() and tracer.Stop() to main function
			if printOutputCode || overwrite {
				if x.Name.Name == function {
					startRedundant := false
					stopRedundant := false
					for _, v := range x.Body.List {
						switch v.(type) {
						case *ast.ExprStmt:
							expr := v.(*ast.ExprStmt)
							switch expr.X.(*ast.CallExpr).Fun.(type) {
							case *ast.SelectorExpr:
								selectorExpr := expr.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr)
								if selectorExpr.X.(*ast.Ident).Name == "tracer" {
									if selectorExpr.Sel.Name == "Start" || selectorExpr.Sel.Name == "Stop" {
										if selectorExpr.Sel.Name == "Start" {
											// fmt.Println("Start redundant")
											startRedundant = true
										}
										if selectorExpr.Sel.Name == "Stop" {
											// fmt.Println("Stop redundant")
											stopRedundant = true
										}
										break
									}
								}
							default:
								fmt.Println(expr.X.(*ast.CallExpr).Fun)
							}
						}
					}

					if !startRedundant {
						x.Body.List = append([]ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("Start")}}}}, x.Body.List...)
					}
					if !stopRedundant {
						x.Body.List = append(x.Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("Stop")}}})
					}
				}
			}
		}
	}

	if !importRedundant {
		newGenDecl := &ast.GenDecl{Specs: []ast.Spec{&ast.ImportSpec{Path: &ast.BasicLit{Value: tracerPath, Kind: token.STRING}}}, Tok: token.IMPORT}
		astf.Decls = append([]ast.Decl{newGenDecl}, astf.Decls...)
	}

	// Formats Go file into byte format
	if !(printOutputCode || overwrite) {
		return
	}

	var buf bytes.Buffer
	err = format.Node(&buf, fset, astf)
	if err != nil {
		panic(err)
	}

	// Prints output code
	if printOutputCode {
		fmt.Println()
		fmt.Println()
		fmt.Println("Output Go File")
		fmt.Println("==============")
		fmt.Println(buf.String())
	}

	// Overwrites instrumented Go file
	if overwrite {
		outfile, err := os.Create(instFilePath)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := outfile.Close(); err != nil {
				panic(err)
			}
		}()
		outfile.Write(buf.Bytes())
	}
}
