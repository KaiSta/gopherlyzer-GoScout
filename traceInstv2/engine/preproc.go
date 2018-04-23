package engine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	raceDetector = false
)

//ASTParser stores somethign
type ASTParser struct {
	FSet         *token.FileSet
	File         *ast.File
	Path         string
	WithCommLink bool
	outPath      string
	nameID       int
	funID        int
}

//NewASTParser creates a new ASTParser
func NewASTParser(p, out string, withCommLink bool) *ASTParser {
	ex := &ASTParser{
		FSet:         token.NewFileSet(),
		File:         nil,
		Path:         p,
		WithCommLink: withCommLink,
		outPath:      out,
		nameID:       0,
		funID:        0,
	}
	astf, err := parser.ParseFile(ex.FSet, ex.Path, nil, 0)
	if err != nil {
		panic(err)
	}
	ex.File = astf
	return ex
}

//Run does something
func (p *ASTParser) Run() {
	for _, d := range p.File.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			p.handleFuncDecl(x)
		case *ast.GenDecl:
			for i, spec := range x.Specs {
				switch y := spec.(type) {
				case *ast.TypeSpec:
					tspec := &ast.TypeSpec{Assign: y.Assign, Comment: y.Comment, Name: y.Name}
					if stru, ok := y.Type.(*ast.StructType); ok {
						nStru := &ast.StructType{Incomplete: stru.Incomplete, Struct: stru.Struct, Fields: &ast.FieldList{Closing: stru.Fields.Closing, Opening: stru.Fields.Opening, List: []*ast.Field{}}}
						for _, f := range stru.Fields.List {
							nf := &ast.Field{Comment: f.Comment, Names: f.Names}
							switch vv := f.Type.(type) {
							case *ast.ChanType:
								nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
								nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
								nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: vv.Value, Names: []*ast.Ident{ast.NewIdent("value")}})
								nf.Type = &ast.ChanType{Value: nStruct}
							default:
								nf.Type = f.Type
							}
							nStru.Fields.List = append(nStru.Fields.List, nf)
						}
						tspec.Type = nStru
						x.Specs[i] = tspec
					}
				}
			}
		default:
			fmt.Println("####", reflect.TypeOf(x))
		}
	}
	var buf bytes.Buffer
	err := format.Node(&buf, p.FSet, p.File)
	if err != nil {
		panic(err)
	}
	fmt.Println("---")
	fmt.Println(buf.String())
	fmt.Println("---")
	file, err := os.Create(p.outPath)
	if err != nil {
		panic(err)
	}
	file.Write(buf.Bytes())
	file.Close()
}

func (p *ASTParser) handleFuncDecl(fun *ast.FuncDecl) {
	if p.WithCommLink {
		p.convertParams(fun)
	}

	if fun.Name.String() == "main" {
		sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RegisterThread")}
		call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"main\""}}}
		fun.Body.List = append([]ast.Stmt{&ast.ExprStmt{X: call}}, fun.Body.List...)
	}
	fun.Body = p.handleBlockStmt(fun.Body)
}

func (p *ASTParser) convertParams(fun *ast.FuncDecl) {
	if fun.Type.Params.List == nil {
		return
	}
	for _, p := range fun.Type.Params.List {
		switch y := p.Type.(type) {
		case *ast.ChanType:
			nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
			nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
			nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: y.Value, Names: []*ast.Ident{ast.NewIdent("value")}})
			y.Value = nStruct
		}
	}
	if fun.Type.Results == nil || fun.Type.Results.List == nil {
		return
	}
	for _, r := range fun.Type.Results.List {
		switch y := r.Type.(type) {
		case *ast.ChanType:
			nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
			nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
			nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: y.Value, Names: []*ast.Ident{ast.NewIdent("value")}})
			y.Value = nStruct
		}
	}
}

func (p *ASTParser) isMapMake(x ast.Expr) *ast.CallExpr {
	if call, ok := x.(*ast.CallExpr); ok {
		if tmp, ok := call.Fun.(*ast.Ident); ok {
			if tmp.Name != "make" {
				return nil
			}
		}
		if _, ok := call.Args[0].(*ast.MapType); ok {
			return call
		}
	}
	return nil
}
func (p *ASTParser) isGenMake(x ast.Expr) *ast.CallExpr {
	if call, ok := x.(*ast.CallExpr); ok {
		if tmp, ok := call.Fun.(*ast.Ident); ok {
			if tmp.Name == "make" {
				return call
			}
		} else {
			//fmt.Println("!!!!!", reflect.TypeOf(call.Fun), p.FSet.Position(x.Pos()).Line)
		}
	}
	return nil
}

func (p *ASTParser) handleAssign(assign *ast.AssignStmt, nBlockStmt *ast.BlockStmt) {
	//if rhs is empty, then there is nothing todo here since there aren't any channels involved
	if len(assign.Rhs) == 0 {
		return
	}
	var isMake bool
	var isGMake bool

	if call := p.isChanMake(assign.Rhs[0]); call != nil {
		if p.WithCommLink {
			var c *ast.ChanType
			var ok bool
			if c, ok = call.Args[0].(*ast.ChanType); !ok {
				nBlockStmt.List = append(nBlockStmt.List, assign)
				return
			}

			nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
			nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
			nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: c.Value, Names: []*ast.Ident{ast.NewIdent("value")}})
			c.Value = nStruct
		}
		isMake = true
	} else if call := p.isGenMake(assign.Rhs[0]); call != nil {
		isGMake = true
	}
	rcv := false
	for _, a := range assign.Rhs {
		//switch y := assign.Rhs[0].(type) {
		switch y := a.(type) {
		case *ast.UnaryExpr:
			if y.Op == token.ARROW {
				if p.WithCommLink {
					p.handleRcvStmt(y, nBlockStmt)
					assign.Rhs[0] = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
					rcv = true
				} else {
					blockstmt := &ast.BlockStmt{List: []ast.Stmt{}}
					p.handleRcvStmt(y, blockstmt)
					nBlockStmt.List = append(nBlockStmt.List, blockstmt.List[0])
					rcv = true
				}
			}
		case *ast.BinaryExpr:
			p.handleBinaryExpr(y, nBlockStmt)
		case *ast.ParenExpr:
			p.handleParenExpr(y, nBlockStmt)
		// if p.WithCommLink {
		// 	if p.handleParenExpr(y, nBlockStmt) {
		// 		assign.Rhs[0] = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		// 	}
		// } else {
		// 	blockstmt := &ast.BlockStmt{List: []ast.Stmt{}}
		// 	if p.handleParenExpr(y, blockstmt) {
		// 		nBlockStmt.List = append(nBlockStmt.List, blockstmt.List[0])
		// 	}
		// }
		default:
			fmt.Println(">>>", reflect.TypeOf(a))
		}
	}
	var tmpList []ast.Stmt
	if raceDetector && !isMake /*&& assign.Tok != token.DEFINE*/ {
		// sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("WriteAcc")}
		// call := &ast.CallExpr{Fun: sel, Args: assign.Lhs}
		add := true
		//isBasic := false
		for _, a := range assign.Rhs {
			addRhs := true
			//	switch zz := assign.Rhs[0].(type) {
			switch zz := a.(type) {
			case *ast.CompositeLit:
				if sel, ok := zz.Type.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Mutex" {
						add = false
					}
				}
				addRhs = false
			case *ast.BasicLit:
				addRhs = false
			case *ast.CallExpr:
				tmp := &ast.BlockStmt{}
				r, _ := p.handleAtomicCall(zz, tmp)
				if r {
					nBlockStmt.List = append(nBlockStmt.List, tmp.List[0])
					nBlockStmt.List = append(nBlockStmt.List, tmp.List[2])
					nBlockStmt.List = append(nBlockStmt.List, tmp.List[3])
				} else {
					p.race_handleCallExpr(zz, nBlockStmt)
				}
				addRhs = false

			case *ast.UnaryExpr:
				if _, ok := zz.X.(*ast.CompositeLit); ok {
					addRhs = false
				} else {
					p.race_handleUnaryExpr(zz, nBlockStmt)
				}
				addRhs = false
			case *ast.Ident:
				if zz.Name == "true" || zz.Name == "false" || zz.Name == "nil" {
					addRhs = false
				}
			case *ast.ParenExpr:
				addRhs = false
			case *ast.BinaryExpr:
				p.race_handleBinaryExpr(zz, nBlockStmt)
				addRhs = false
			case *ast.SliceExpr:
				tmpList = append(tmpList, &ast.ExprStmt{X: p.getReadAcc(zz.X)})
				addRhs = false
			case *ast.TypeAssertExpr:
				if call, ok := zz.X.(*ast.CallExpr); ok {
					p.handleAtomicValue(call, nBlockStmt)
					addRhs = false
				}
			default:
				fmt.Println("handleAssign$$$", reflect.TypeOf(assign.Rhs[0]), p.FSet.Position(zz.Pos()).Line)
			}

			//	if add {
			//	tmp := p.getWriteAcc(assign.Lhs[0])
			//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: tmp})

			if !isGMake && addRhs && !rcv {
				//for _, a := range assign.Rhs {
				tmpList = append(tmpList, &ast.ExprStmt{X: p.getReadAcc(a)})
				//}
			}
			//	}
		}

		if add {
			for _, a := range assign.Lhs {
				if id, ok := a.(*ast.Ident); ok {
					if id.Name == "_" {
						continue
					}
				}
				tmpList = append(tmpList, &ast.ExprStmt{X: p.getWriteAcc(a)})
			}
		}

	}

	// if raceDetector && !isMake {
	// 	// check right hand side for read access
	// 	for _, r := range assign.Rhs {
	// 		switch x := r.(type) {
	// 		case *ast.CallExpr:
	// 			if !isAtomic {
	// 				p.race_handleCallExpr(x, nBlockStmt)
	// 			}
	// 		case *ast.UnaryExpr:
	// 			p.race_handleUnaryExpr(x, nBlockStmt)
	// 		case *ast.BinaryExpr:
	// 			p.race_handleBinaryExpr(x, nBlockStmt)
	// 		}
	// 	}
	// }

	nBlockStmt.List = append(nBlockStmt.List, assign)
	if isMake {
		sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RegisterChan")}
		capCall := &ast.CallExpr{Fun: ast.NewIdent("cap"), Args: []ast.Expr{assign.Lhs[0]}}
		call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{assign.Lhs[0], capCall}}
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: call})
	}
	if len(tmpList) > 0 {
		nBlockStmt.List = append(nBlockStmt.List, tmpList...)
	}
}

func (p *ASTParser) handleParenExpr(x *ast.ParenExpr, nBlockStmt *ast.BlockStmt) bool {
	if p.isRcv(x.X) {
		nBlock := &ast.BlockStmt{}
		p.handleRcvStmt(x.X.(*ast.UnaryExpr), nBlock)
		nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
		if p.WithCommLink {
			x.X = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
		return true
	}
	if v, ok := x.X.(*ast.BinaryExpr); ok {

		p.handleBinaryExpr(v, nBlockStmt)
		x.X = v
		p.race_handleBinaryExpr(v, nBlockStmt)
		return true
	}
	return false
}

func (p *ASTParser) handleBinaryExpr(x *ast.BinaryExpr, nBlockStmt *ast.BlockStmt) {
	switch v := x.X.(type) {
	case *ast.UnaryExpr:
		if v.Op == token.ARROW {
			b := &ast.BlockStmt{}
			p.handleRcvStmt(v, b)

			if p.WithCommLink {
				nBlockStmt.List = append(nBlockStmt.List, b.List...)
				x.X = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
			} else {
				nBlockStmt.List = append(nBlockStmt.List, b.List[0])
			}
		}
	case *ast.BinaryExpr:
		p.handleBinaryExpr(v, nBlockStmt)
	case *ast.ParenExpr:
		nBlock := &ast.BlockStmt{}
		if p.handleParenExpr(v, nBlock) {
			if p.WithCommLink {
				nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
			} else {
				nBlockStmt.List = append(nBlockStmt.List, nBlock.List[0])
			}
			//x.X = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	case *ast.Ident:
		//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
	default:
		fmt.Println("####", reflect.TypeOf(v), v)
	}

	switch v := x.Y.(type) {
	case *ast.UnaryExpr:
		if v.Op == token.ARROW {
			b := &ast.BlockStmt{}
			p.handleRcvStmt(v, b)
			if p.WithCommLink {
				nBlockStmt.List = append(nBlockStmt.List, b.List...)
				x.Y = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
			} else {
				nBlockStmt.List = append(nBlockStmt.List, b.List[0])
			}
		}
	case *ast.BinaryExpr:
		p.handleBinaryExpr(v, nBlockStmt)
	case *ast.ParenExpr:
		nBlock := &ast.BlockStmt{}
		if p.handleParenExpr(v, nBlock) {
			if p.WithCommLink {
				nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
			} else {
				nBlockStmt.List = append(nBlockStmt.List, nBlock.List[0])
			}
			//	x.Y = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	case *ast.Ident:
		//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
	default:
		fmt.Println("####", reflect.TypeOf(v), v)
	}
	//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: x})
}

func (p *ASTParser) race_handleBinaryExpr(x *ast.BinaryExpr, nBlockStmt *ast.BlockStmt) {
	if !raceDetector {
		return
	}
	switch v := x.X.(type) {
	case *ast.CallExpr:
		p.race_handleCallExpr(v, nBlockStmt)
	case *ast.BinaryExpr:
		p.race_handleBinaryExpr(v, nBlockStmt)
	case *ast.UnaryExpr:
		p.race_handleUnaryExpr(v, nBlockStmt)
	case *ast.Ident:
		/*	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("ReadAcc")}
			call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{v}}
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: call}) */
		if v.Name != "nil" && v.Name != "false" && v.Name != "true" {
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		}
	case *ast.SelectorExpr:
		ignore := false
		if id, ok := v.X.(*ast.Ident); ok {
			if id.Name == "io" {
				ignore = true
			}
		}
		if !ignore {
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(x.X)})
		}
	}
	switch v := x.Y.(type) {
	case *ast.CallExpr:
		p.race_handleCallExpr(v, nBlockStmt)
	case *ast.BinaryExpr:
		p.race_handleBinaryExpr(v, nBlockStmt)
	case *ast.UnaryExpr:
		p.race_handleUnaryExpr(v, nBlockStmt)
	case *ast.Ident:
		/*sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("ReadAcc")}
		call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{v}}
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: call})*/
		if v.Name != "nil" && v.Name != "false" && v.Name != "true" {
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		}
	case *ast.SelectorExpr:
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(x.Y)})
	}
}
func (p *ASTParser) race_handleUnaryExpr(x *ast.UnaryExpr, nBlockStmt *ast.BlockStmt) {
	if !raceDetector {
		return
	}
	switch v := x.X.(type) {
	case *ast.CallExpr:
		p.race_handleCallExpr(v, nBlockStmt)
	case *ast.BinaryExpr:
		p.race_handleBinaryExpr(v, nBlockStmt)
	case *ast.UnaryExpr:
		p.race_handleUnaryExpr(v, nBlockStmt)
	case *ast.Ident:
		/*sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("WriteAcc")}
		call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{v}}
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: call})*/
		if x.Op == token.AND || x.Op == token.NOT || x.Op == token.XOR {
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		} else if x.Op == token.ARROW {
		} else {
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getWriteAcc(v)})
		}
	case *ast.SelectorExpr:
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
	}
}

func (p *ASTParser) race_handleSendStmt(x *ast.SendStmt, nBlockStmt *ast.BlockStmt) {
	if !raceDetector {
		return
	}
	if _, ok := x.Value.(*ast.BasicLit); ok {
		return
	} else if id, ok := x.Value.(*ast.Ident); ok && id.Obj == nil {
		if id.Name == "true" || id.Name == "false" {
			return
		}
	} else if _, ok := x.Value.(*ast.CallExpr); ok {
		return
	}
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(x.Value)})
}

func (p *ASTParser) is_MapMake(e *ast.Ident) bool {
	if e.Obj == nil {
		return false
	}
	switch v := e.Obj.Decl.(type) {
	case *ast.AssignStmt:
		fmt.Println(v)
	}
	return false
}

func (p *ASTParser) race_handleCallExpr(x *ast.CallExpr, nBlockStmt *ast.BlockStmt) int {
	if !raceDetector {
		return 0
	}
	for _, a := range x.Args {
		switch v := a.(type) {
		case *ast.CallExpr:
			p.race_handleCallExpr(v, nBlockStmt)
		case *ast.BinaryExpr:
			p.race_handleBinaryExpr(v, nBlockStmt)
		case *ast.UnaryExpr:
			p.race_handleUnaryExpr(v, nBlockStmt)
		case *ast.Ident:
			/*sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("ReadAcc")}
			call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{v}}
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: call})*/
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		case *ast.SelectorExpr:
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		case *ast.IndexExpr:
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		case *ast.SliceExpr:
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v.X)})
		case *ast.StarExpr:
			nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
		default:
			fmt.Println("racecall>>>", reflect.TypeOf(a), p.FSet.Position(x.Pos()).Line)
		}
	}
	return 0
}

func (p *ASTParser) race_handleIncDec(x ast.Expr, nBlockStmt *ast.BlockStmt) {
	if !raceDetector {
		return
	}
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getWriteAcc(x)})
}

func (p *ASTParser) getReadAcc(e ast.Expr) *ast.CallExpr {
	tmp := p.FSet.Position(e.Pos())
	q, r := filepath.Abs(tmp.Filename)
	q = tmp.Filename
	if r != nil {
		//	panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

	un := &ast.UnaryExpr{Op: token.AND, X: e}
	switch v := e.(type) {
	case *ast.IndexExpr:
		if id, ok := v.X.(*ast.Ident); ok {
			if assign, ok := id.Obj.Decl.(*ast.AssignStmt); ok {
				if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
					if _, ok := call.Args[0].(*ast.MapType); ok {
						un.X = v.X
					}
				}
			}
		}
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			un.X = v.X
		}
	case *ast.StarExpr:
		un.X = &ast.ParenExpr{X: v}
	}

	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("ReadAcc")}
	//call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{un, b, p.getGID()}}
	call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{un, b, p.getTIDFromCache()}}
	return call
}
func (p *ASTParser) getWriteAcc(e ast.Expr) *ast.CallExpr {
	tmp := p.FSet.Position(e.Pos())
	q, r := filepath.Abs(tmp.Filename)
	q = tmp.Filename
	if r != nil {
		panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

	un := &ast.UnaryExpr{Op: token.AND, X: e}
	switch v := e.(type) {
	case *ast.IndexExpr:
		if id, ok := v.X.(*ast.Ident); ok {
			if assign, ok := id.Obj.Decl.(*ast.AssignStmt); ok {
				if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
					if len(call.Args) > 0 {
						if _, ok := call.Args[0].(*ast.MapType); ok {
							un.X = v.X
						}
					}
				}
			}
		}
	case *ast.StarExpr:
		un.X = &ast.ParenExpr{X: v}
	}

	//un := &ast.UnaryExpr{Op: token.AND, X: e}
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("WriteAcc")}
	//call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{un, b, p.getGID()}}
	call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{un, b, p.getTIDFromCache()}}
	return call
}

func (p *ASTParser) handleSendStmt(x *ast.SendStmt, nBlockStmt *ast.BlockStmt) {
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreSend")}
	sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostSend")}
	//sel3 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetGID")}
	gpid := p.getTIDFromCache() // &ast.CallExpr{Fun: sel3}

	tmp := p.FSet.Position(x.Pos())
	q, r := filepath.Abs(tmp.Filename)
	q = tmp.Filename
	if r != nil {
		panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
	prep := &ast.CallExpr{Fun: sel, Args: []ast.Expr{x.Chan, b, p.getTIDFromCache()}}
	commit := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{x.Chan, b, p.getTIDFromCache()}}

	var t ast.Expr
	if chanT, ok := p.getType(x.Chan).(*ast.ChanType); ok {
		if s, ok := chanT.Value.(*ast.StructType); ok {
			if len(s.Fields.List) == 0 { //empty struct
				t = chanT.Value
			} else {
				t = s.Fields.List[1].Type
			}

		} else {
			t = chanT.Value
		}
	} else if id, ok := p.getType(x.Chan).(*ast.Ident); ok {
		vs := id.Obj.Decl.(*ast.ValueSpec)
		if call, ok := vs.Values[0].(*ast.CallExpr); ok {
			if call.Fun.(*ast.Ident).String() == "make" {
				if chanT, ok := call.Args[0].(*ast.ChanType); ok {
					if s, ok := chanT.Value.(*ast.StructType); ok {
						t = s.Fields.List[1].Type
					} else {
						t = chanT.Value
					}
				}
			}
		}
	} else if sel, ok := p.getType(x.Chan).(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok {
			if ass, ok := id.Obj.Decl.(*ast.AssignStmt); ok {
				if comp, ok := ass.Rhs[0].(*ast.CompositeLit); ok {
					if id, ok := comp.Type.(*ast.Ident); ok {
						if spec, ok := id.Obj.Decl.(*ast.TypeSpec); ok {
							if stru, ok := spec.Type.(*ast.StructType); ok {
								for _, f := range stru.Fields.List {
									switch v := f.Type.(type) {
									case *ast.ChanType:
										if sel.Sel.Name == f.Names[0].Name {
											t = v.Value
										}
									}
								}
							}
						}
					}
				}
			}
		}
		// if sel.Sel.Obj == nil {
		// 	return
		// }
		// vs := sel.Sel.Obj.Decl.(*ast.ValueSpec)
		// if call, ok := vs.Values[0].(*ast.CallExpr); ok {
		// 	if call.Fun.(*ast.Ident).String() == "make" {
		// 		if chanT, ok := call.Args[0].(*ast.ChanType); ok {
		// 			if s, ok := chanT.Value.(*ast.StructType); ok {
		// 				t = s.Fields.List[1].Type
		// 			} else {
		// 				t = chanT.Value
		// 			}
		// 		}
		// 	}
		// }
	} else {
		fmt.Println("Foo", p.FSet.Position(x.Pos()).Line, "||", reflect.TypeOf(p.getType(x.Chan)), x.Chan)
		return
	}
	if t == nil {
		return //handle it somewhere else because we couldnt find the type
	}

	nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
	nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
	nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: t, Names: []*ast.Ident{ast.NewIdent("value")}})

	ex := x.Value
	if rcv, ok := x.Value.(*ast.UnaryExpr); ok {
		if rcv.Op == token.ARROW { //rcv
			b2 := &ast.BlockStmt{}
			p.handleRcvStmt(rcv, b2)
			nBlockStmt.List = append(nBlockStmt.List, b2.List...)
			ex = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	}

	clit := &ast.CompositeLit{Type: nStruct, Elts: []ast.Expr{gpid, ex}}
	nChan := &ast.SendStmt{Chan: x.Chan, Arrow: x.Arrow, Value: clit}

	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})

	if p.WithCommLink {
		nBlockStmt.List = append(nBlockStmt.List, nChan)
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: commit})
	} else {
		nBlockStmt.List = append(nBlockStmt.List, x)
	}
}

func (p *ASTParser) handleRcvStmt(x *ast.UnaryExpr, nBlockStmt *ast.BlockStmt) {
	p.nameID++
	//	chanT := p.getType(x.X).(*ast.ChanType)

	var t ast.Expr
	if chanT, ok := p.getType(x.X).(*ast.ChanType); ok {
		if s, ok := chanT.Value.(*ast.StructType); ok {
			if len(s.Fields.List) == 0 {
				t = chanT.Value
			} else {
				t = s.Fields.List[1].Type
			}

		} else {
			t = chanT.Value
		}
	} else if id, ok := p.getType(x.X).(*ast.Ident); ok {
		if vs, ok := id.Obj.Decl.(*ast.ValueSpec); ok {
			if call, ok := vs.Values[0].(*ast.CallExpr); ok {
				if call.Fun.(*ast.Ident).String() == "make" {
					if chanT, ok := call.Args[0].(*ast.ChanType); ok {
						if s, ok := chanT.Value.(*ast.StructType); ok {
							t = s.Fields.List[1].Type
						} else {
							t = chanT.Value
						}
					}
				}
			}
		}
	} else if sel, ok := p.getType(x.X).(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok {
			if ass, ok := id.Obj.Decl.(*ast.AssignStmt); ok {
				if comp, ok := ass.Rhs[0].(*ast.CompositeLit); ok {
					if id, ok := comp.Type.(*ast.Ident); ok {
						if spec, ok := id.Obj.Decl.(*ast.TypeSpec); ok {
							if stru, ok := spec.Type.(*ast.StructType); ok {
								for _, f := range stru.Fields.List {
									switch v := f.Type.(type) {
									case *ast.ChanType:
										if sel.Sel.Name == f.Names[0].Name {
											t = v.Value
										}
									}
								}
							}
						}
					}
				}
			}
		}
	} else {
		fmt.Println("Fooo22,", p.FSet.Position(x.Pos()).Line)
		return
	}

	if t == nil {
		return //handle it somewhere else because we couldnt find the type
	}

	nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
	nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
	nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: p.getType(t), Names: []*ast.Ident{ast.NewIdent("value")}})

	assign := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID))}, Tok: token.DEFINE, Rhs: []ast.Expr{x}}

	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreRcv")}
	sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostRcv")}
	sel3 := &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("threadId")}

	tmp := p.FSet.Position(x.Pos())
	q, r := filepath.Abs(tmp.Filename)
	q = tmp.Filename
	if r != nil {
		panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

	prep := &ast.CallExpr{Fun: sel, Args: []ast.Expr{x.X, b, p.getTIDFromCache()}}
	commit := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{x.X, b, sel3, p.getTIDFromCache()}}

	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})

	if p.WithCommLink {
		nBlockStmt.List = append(nBlockStmt.List, assign)
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: commit})
	} else {
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: x})
	}
}

func (p *ASTParser) handleIfStmt(ifs *ast.IfStmt) *ast.IfStmt {
	ifs.Body = p.handleBlockStmt(ifs.Body)

	if ifs.Else == nil {
		return ifs
	}

	if elseIf, ok := ifs.Else.(*ast.IfStmt); ok {
		ifs.Else = p.handleIfStmt(elseIf)
	} else if bkStmt, ok := ifs.Else.(*ast.BlockStmt); ok {
		ifs.Else = p.handleBlockStmt(bkStmt)
	}
	return ifs
}

func (p *ASTParser) handleForStmt(f *ast.ForStmt) *ast.ForStmt {
	f.Body = p.handleBlockStmt(f.Body)
	return f
}
func (p *ASTParser) handleRangeStmt(f *ast.RangeStmt) ast.Stmt {
	f.Body = p.handleBlockStmt(f.Body)

	t := p.getType(f.X)
	if _, ok := t.(*ast.ChanType); !ok {
		return f
	}

	rcv := &ast.UnaryExpr{Op: token.ARROW, X: f.X, OpPos: f.Pos()}
	tmp := &ast.BlockStmt{}
	p.handleRcvStmt(rcv, tmp)
	if ass, ok := tmp.List[1].(*ast.AssignStmt); ok {
		ass.Lhs = append(ass.Lhs, ast.NewIdent("err"))
		tmp.List[1] = ass
	}
	equals := &ast.BinaryExpr{X: ast.NewIdent("err"), Y: ast.NewIdent("nil"), Op: token.NEQ}
	ifstmt := &ast.IfStmt{Cond: equals, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: ast.NewIdent("break")}}}}
	assign := &ast.AssignStmt{Tok: token.DEFINE, Lhs: []ast.Expr{f.Key}}
	assign.Rhs = append(assign.Rhs, &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")})

	var block *ast.BlockStmt
	if len(tmp.List) == 3 {
		block = &ast.BlockStmt{List: append([]ast.Stmt{tmp.List[0], tmp.List[1], tmp.List[2], ifstmt, assign}, f.Body.List...)}
	} else if len(tmp.List) == 2 {
		block = &ast.BlockStmt{List: append([]ast.Stmt{tmp.List[0], tmp.List[1], ifstmt, assign}, f.Body.List...)}

	}
	forstmt := &ast.ForStmt{Body: block}
	return forstmt
}

func (p *ASTParser) isRWMutex(can ast.Expr) bool {
	var xx *ast.Ident
	if tmp, ok := can.(*ast.Ident); ok {
		xx = tmp
	} else if tmp, ok := can.(*ast.SelectorExpr); ok {
		xx = tmp.Sel
	}
	if xx.Obj == nil || xx.Obj.Decl == nil {
		return false
	}

	var pkg string
	var field string
	if tmp1, ok := xx.Obj.Decl.(*ast.AssignStmt); ok {
		if tmp2, ok := tmp1.Rhs[0].(*ast.CompositeLit); ok {
			if tmp3, ok := tmp2.Type.(*ast.SelectorExpr); ok {
				pkg = tmp3.X.(*ast.Ident).Name
				field = tmp3.Sel.Name
			}
		}
	} else if tmp1, ok := xx.Obj.Decl.(*ast.Field); ok {
		if tmp2, ok := tmp1.Type.(*ast.StarExpr); ok {
			if tmp3, ok := tmp2.X.(*ast.SelectorExpr); ok {
				pkg = tmp3.X.(*ast.Ident).Name
				field = tmp3.Sel.Name
			}
		}
	}

	if pkg == "sync" && field == "RWMutex" {
		return true
	}

	return false
}

func (p *ASTParser) handleMutexOps(call *ast.CallExpr, out *ast.BlockStmt) int {
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		tmp := p.FSet.Position(call.Pos())
		q, r := filepath.Abs(tmp.Filename)
		q = tmp.Filename
		if r != nil {
			//panic(r)
			//don't panic here. r is nil if the external function is somehow unknown. adding tracer.RegisterThread in the main
			//function triggered this panic here because its a selectorExpr but the function inside is unknown to the parser
			return 0 // return 0 because it's no mutex so we handled nothing
		}
		b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

		if fun.Sel.Name == "Lock" {
			preLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer")}
			if p.isRWMutex(fun.X) {
				preLockSel.Sel = ast.NewIdent("RPreLock")
			} else {
				preLockSel.Sel = ast.NewIdent("PreLock")
			}
			//	preLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreLock")}
			un := &ast.UnaryExpr{Op: token.AND, X: fun.X}
			preLockCall := &ast.CallExpr{Fun: preLockSel, Args: []ast.Expr{un, b, p.getTIDFromCache()}}
			out.List = append(out.List, &ast.ExprStmt{X: preLockCall})
			out.List = append(out.List, &ast.ExprStmt{X: call})

			return 1
		} else if fun.Sel.Name == "Unlock" {
			// postLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostUnlock")}
			// postLockCall := &ast.CallExpr{Fun: postLockSel, Args: []ast.Expr{fun.X, b}}
			// out.List = append(out.List, &ast.ExprStmt{X: postLockCall})
			postLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer")}
			if p.isRWMutex(fun.X) {
				postLockSel.Sel = ast.NewIdent("RPostLock")
			} else {
				postLockSel.Sel = ast.NewIdent("PostLock")
			}

			//postLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostLock")}
			un := &ast.UnaryExpr{Op: token.AND, X: fun.X}
			postLockCall := &ast.CallExpr{Fun: postLockSel, Args: []ast.Expr{un, b, p.getTIDFromCache()}}
			out.List = append(out.List, &ast.ExprStmt{X: postLockCall})
			out.List = append(out.List, &ast.ExprStmt{X: call})

			return 1
		} else if fun.Sel.Name == "RLock" {
			preLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RRPreLock")}
			preLockCall := &ast.CallExpr{Fun: preLockSel, Args: []ast.Expr{fun.X, b, p.getTIDFromCache()}}
			out.List = append(out.List, &ast.ExprStmt{X: preLockCall})
			out.List = append(out.List, &ast.ExprStmt{X: call})
			return 1
		} else if fun.Sel.Name == "RUnlock" {
			postLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RRPostLock")}
			postLockCall := &ast.CallExpr{Fun: postLockSel, Args: []ast.Expr{fun.X, b, p.getTIDFromCache()}}
			out.List = append(out.List, &ast.ExprStmt{X: postLockCall})
			out.List = append(out.List, &ast.ExprStmt{X: call})
			return 1
		}
	}
	return 0
}

func (p *ASTParser) handleAtomicCall(call *ast.CallExpr, out *ast.BlockStmt) (bool, int) {
	if !raceDetector {
		return false, 0 //quickfix since we don't handle atomics atm
	}
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		tmp := p.FSet.Position(call.Pos())
		q, r := filepath.Abs(tmp.Filename)
		if r != nil {
			//same problem as for mutex
			//	panic(r)
			return false, 0
		}
		q = tmp.Filename
		b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

		if id, ok := fun.X.(*ast.Ident); ok {
			if id.Name == "atomic" {

				tmp := &ast.BlockStmt{List: make([]ast.Stmt, 0)}
				p.race_handleCallExpr(call, tmp)

				out.List = append(out.List, tmp.List[1:]...)

				preLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreLock")}
				var exp ast.Expr
				if un, ok := call.Args[0].(*ast.UnaryExpr); ok {
					exp = un.X
				}
				preLockCall := &ast.CallExpr{Fun: preLockSel, Args: []ast.Expr{call.Args[0], b, p.getTIDFromCache()}}
				out.List = append(out.List, &ast.ExprStmt{X: preLockCall})

				out.List = append(out.List, &ast.ExprStmt{X: call})

				if strings.HasPrefix(fun.Sel.Name, "Load") {
					out.List = append(out.List, &ast.ExprStmt{X: p.getReadAcc(exp)})
				} else {
					out.List = append(out.List, &ast.ExprStmt{X: p.getWriteAcc(exp)})
				}

				postLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostLock")}
				postLockCall := &ast.CallExpr{Fun: postLockSel, Args: []ast.Expr{call.Args[0], b, p.getTIDFromCache()}}
				out.List = append(out.List, &ast.ExprStmt{X: postLockCall})

				//	postULockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostUnlock")}
				//	postULockCall := &ast.CallExpr{Fun: postULockSel, Args: []ast.Expr{exp, b}}
				//	out.List = append(out.List, &ast.ExprStmt{X: postULockCall})
				return true, 1
			}

			//return p.handleAtomicValue(call, out), 0
		}
	}
	if p.handleAtomicValue(call, out) {
		return true, 0
	}
	return false, 0
}
func (p *ASTParser) handleAtomicValue(call *ast.CallExpr, out *ast.BlockStmt) bool {
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		if id, ok := fun.X.(*ast.Ident); ok {
			if id.Obj == nil {
				return false
			}
			if valspec, ok := id.Obj.Decl.(*ast.ValueSpec); ok {
				if sel, ok := valspec.Type.(*ast.SelectorExpr); ok {
					if id2, ok := sel.X.(*ast.Ident); ok {
						if id2.Name == "atomic" {
							tmp := p.FSet.Position(call.Pos())
							b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(tmp.Filename), tmp.Line)}
							preLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreLock")}
							preLockCall := &ast.CallExpr{Fun: preLockSel, Args: []ast.Expr{&ast.UnaryExpr{X: id, Op: token.AND}, b, p.getTIDFromCache()}}
							out.List = append(out.List, &ast.ExprStmt{preLockCall})

							if strings.HasPrefix(fun.Sel.Name, "Load") {
								out.List = append(out.List, &ast.ExprStmt{X: p.getReadAcc(id)})
							} else {
								out.List = append(out.List, &ast.ExprStmt{X: p.getWriteAcc(id)})
							}
							postLockSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostLock")}
							postLockCall := &ast.CallExpr{Fun: postLockSel, Args: []ast.Expr{&ast.UnaryExpr{X: id, Op: token.AND}, b, p.getTIDFromCache()}}
							out.List = append(out.List, &ast.ExprStmt{postLockCall})

							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (p *ASTParser) race_handleIfStmt(x *ast.IfStmt, out *ast.BlockStmt) {
	if !raceDetector {
		return
	}
	switch y := x.Cond.(type) {
	case *ast.BinaryExpr:
		p.race_handleBinaryExpr(y, out)
	case *ast.UnaryExpr:
		p.race_handleUnaryExpr(y, out)
	}

	if els, ok := x.Else.(*ast.IfStmt); ok {
		p.race_handleIfStmt(els, out)
	}
}

func (p *ASTParser) getTIDFromCache() *ast.Ident {
	AddCache = true
	return ast.NewIdent("myTIDCache")
}

var AddCache = false

func (p *ASTParser) handleBlockStmt(in *ast.BlockStmt) *ast.BlockStmt {
	threadIDCache := &ast.AssignStmt{
		Lhs:    []ast.Expr{ast.NewIdent("myTIDCache")},
		TokPos: 0,
		Tok:    token.DEFINE,
		Rhs:    []ast.Expr{p.getGID()},
	}
	//out := &ast.BlockStmt{Lbrace: in.Lbrace, Rbrace: in.Rbrace, List: make([]ast.Stmt, 0)}
	out := &ast.BlockStmt{in.Lbrace, []ast.Stmt{}, in.Rbrace}
	for _, b := range in.List {
		switch x := b.(type) {
		case *ast.ExprStmt: //check for unary expr which could contain a receive without assignment
			switch y := x.X.(type) {
			case *ast.UnaryExpr:
				if raceDetector {
					p.race_handleUnaryExpr(y, out)
				}
				if y.Op == token.ARROW { //rcv
					p.handleRcvStmt(y, out)
				}
			case *ast.CallExpr:

				handled := 0
				if p.handleClose(y, out) > 0 {
					continue
				}
				handled += p.handleMutexOps(y, out)

				if isAtomic, h := p.handleAtomicCall(y, out); isAtomic {
					handled += h
				} else {
					handled += p.race_handleCallExpr(y, out)
				}

				handled += p.handleRegCalls(y, out)

				if handled == 0 {
					out.List = append(out.List, x)
				}

			default:
				fmt.Println(">>>", reflect.TypeOf(x.X), p.FSet.Position(x.X.Pos()).Line)
				out.List = append(out.List, x)
			}
		case *ast.AssignStmt: //receive with assign, or channel creation
			//	out.List = append(out.List, p.handleAssign(x, out))
			p.handleAssign(x, out)
		case *ast.SendStmt:
			if raceDetector {
				p.race_handleSendStmt(x, out)
			}
			p.handleSendStmt(x, out)
		case *ast.IfStmt:
			p.race_handleIfStmt(x, out)
			out.List = append(out.List, p.handleIfStmt(x))
		case *ast.ForStmt:
			out.List = append(out.List, p.handleForStmt(x))
		case *ast.RangeStmt:
			out.List = append(out.List, p.handleRangeStmt(x))
		case *ast.SelectStmt:
			p.handleSelect(x, out)
		case *ast.GoStmt:
			fmt.Println(p.FSet.Position(x.Pos()).Line)
			//get new sigWaitID
			p.nameID++
			varName := fmt.Sprintf("tmp%v", p.nameID)
			getSWIDSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetWaitSigID")}
			callgetSWID := &ast.CallExpr{Fun: getSWIDSel, Args: make([]ast.Expr, 0)}
			ass := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(varName)}, Rhs: []ast.Expr{callgetSWID}, Tok: token.DEFINE}
			out.List = append(out.List, ass)

			sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("Signal")}
			call2 := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{ast.NewIdent(varName), p.getTIDFromCache()}}

			out.List = append(out.List, &ast.ExprStmt{X: call2})

			var funName string
			anno := false
			selF := false
			var list []*ast.Field
			if id, ok := x.Call.Fun.(*ast.Ident); ok {
				funName = fmt.Sprintf("\"%v%v\"", id.Name, p.funID)
				p.funID++
				if id.Obj != nil {
					if funDec, ok := id.Obj.Decl.(*ast.FuncDecl); ok {
						if funDec.Type.Params != nil {
							//list = funDec.Type.Params.List
							for i, f := range funDec.Type.Params.List {
								var nf *ast.Field
								if _, ok := x.Call.Args[i].(*ast.Ident); ok {
									//nf = &ast.Field{Type: f.Type, Names: []*ast.Ident{n}}
									nf = &ast.Field{Type: f.Type, Names: f.Names}
								} else if _, ok := x.Call.Args[i].(*ast.BasicLit); ok {
									if len(f.Names) > 0 {
										nf = &ast.Field{Type: f.Type, Names: f.Names}
									} else {
										nf = &ast.Field{Type: f.Type}
									}
								} else {
									nf = &ast.Field{Type: f.Type, Names: f.Names}
								}
								list = append(list, nf)
							}
						}
					}
				}
			} else if sel, ok := x.Call.Fun.(*ast.SelectorExpr); ok {
				var id string
				switch tmp := sel.X.(type) {
				case *ast.Ident:
					id = tmp.Name
				case *ast.SelectorExpr:
					id = fmt.Sprintf("%v.%v", tmp.X, tmp.Sel)
				}
				//	id := sel.X.(*ast.Ident)
				funName = fmt.Sprintf("\"%v.%v%v\"", id, sel.Sel.Name, p.funID)
				p.funID++
				selF = true
			} else {
				//anonymous function
				funName = fmt.Sprintf("\"fun%v\"", p.funID)
				p.funID++
				anno = true
			}

			b := &ast.BasicLit{Kind: token.STRING, Value: funName}
			selRegister := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RegisterThread")}
			callRegister := &ast.CallExpr{Fun: selRegister, Args: []ast.Expr{b}}

			sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("Wait")}
			call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{ast.NewIdent(varName), p.getGID()}}
			ft := &ast.FuncType{Params: &ast.FieldList{List: list}}
			var body *ast.BlockStmt
			if selF {
				body = &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callRegister}, &ast.ExprStmt{X: call}, &ast.ExprStmt{X: x.Call}}}
			} else if !anno {
				args := make([]ast.Expr, 0)
				for _, a := range x.Call.Args {
					switch val := a.(type) {
					case *ast.UnaryExpr:
						if val.Op == token.AND {
							args = append(args, val.X)
						} else {
							args = append(args, a)
						}
					default:
						args = append(args, a)
					}
				}
				//	ncall := &ast.CallExpr{Fun: x.Call.Fun, Args: args}
				ncall := &ast.CallExpr{Fun: x.Call.Fun, Args: make([]ast.Expr, 0)}
				for _, field := range list {
					for _, id := range field.Names {
						ncall.Args = append(ncall.Args, id)
					}
				}

				body = &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callRegister}, &ast.ExprStmt{X: call}, &ast.ExprStmt{X: ncall}}}
				//body = &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: call}, &ast.ExprStmt{X: x.Call}}}
			} else {

				fl := x.Call.Fun.(*ast.FuncLit)
				list := p.handleBlockStmt(fl.Body).List
				body = &ast.BlockStmt{List: append([]ast.Stmt{&ast.ExprStmt{X: callRegister}, &ast.ExprStmt{X: call}}, list...)}
			}
			fl := &ast.FuncLit{Body: body, Type: ft}

			if !selF {
				x.Call = &ast.CallExpr{Fun: fl, Args: x.Call.Args}
			} else {
				x.Call = &ast.CallExpr{Fun: fl}
			}

			out.List = append(out.List, x)
		case *ast.DeclStmt:
			switch y := x.Decl.(type) {
			case *ast.GenDecl:
				if z, ok := y.Specs[0].(*ast.ValueSpec); ok {
					//tmp := &ast.BlockStmt{}
					if len(z.Values) > 0 {
						p.handleAssign(&ast.AssignStmt{Lhs: []ast.Expr{z.Names[0]}, Rhs: []ast.Expr{z.Values[0]}, Tok: token.DEFINE}, out)
					} else {
						out.List = append(out.List, x)
						p.handleAssign(&ast.AssignStmt{Lhs: []ast.Expr{z.Names[0]}, Rhs: []ast.Expr{}, Tok: token.DEFINE}, out)
					}

					// for _, e := range tmp.List {
					// 	fmt.Println(e, reflect.TypeOf(e))
					// }
				}
			}
		case *ast.IncDecStmt:
			if raceDetector {
				p.race_handleIncDec(x.X, out)
			}
			out.List = append(out.List, x)
		case *ast.ReturnStmt:
			out.List = append(out.List, x)
		default:
			fmt.Println("handleBlock>>>", reflect.TypeOf(x), p.FSet.Position(x.Pos()).Line)
			out.List = append(out.List, x)
		}
	}

	//if AddCache {
	out.List = append([]ast.Stmt{threadIDCache}, out.List...)
	//}
	AddCache = false
	return out
}

func (p *ASTParser) handleRegCalls(c *ast.CallExpr, nBlockStmt *ast.BlockStmt) int {
	for i, arg := range c.Args {
		if p.isRcv(arg) {
			nBlock := &ast.BlockStmt{}
			p.handleRcvStmt(arg.(*ast.UnaryExpr), nBlock)

			nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
			sel := &ast.SelectorExpr{X: nBlock.List[1].(*ast.AssignStmt).Lhs[0], Sel: ast.NewIdent("value")}
			c.Args[i] = sel
		}
	}
	return 0
	//	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: c})  UNSURE!!
	//fmt.Println(c.Fun, reflect.TypeOf(c.Fun))
}

func (p *ASTParser) isSend(n ast.Node) bool {
	_, ok := n.(*ast.SendStmt)
	return ok
}
func (p *ASTParser) isRcv(n ast.Node) bool {

	if un, ok := n.(*ast.UnaryExpr); ok {
		return un.Op == token.ARROW
	}
	return false
}

func (p *ASTParser) handleClose(c *ast.CallExpr, nBlockStmt *ast.BlockStmt) int {
	if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "close" {
		sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreClose")}
		sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostClose")}

		tmp := p.FSet.Position(c.Pos())
		q, r := filepath.Abs(tmp.Filename)
		q = tmp.Filename
		if r != nil {
			panic(r)
		}
		b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

		prep := &ast.CallExpr{Fun: sel, Args: append(c.Args, b, p.getGID())}
		com := &ast.CallExpr{Fun: sel2, Args: append(c.Args, b, p.getGID())}

		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: c})
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: com})
		return 1
	}
	return 0
	//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: c})
}

func (p *ASTParser) handleSelect(s *ast.SelectStmt, nBlockStmt *ast.BlockStmt) {
	fmt.Println(">>>!!!>>>", p.FSet.Position(s.Pos()))
	var args []ast.Expr
	for _, cc := range s.Body.List {
		var op string
		var e ast.Expr
		var pos *ast.BasicLit
		comm := cc.(*ast.CommClause)
		if comm.Comm == nil {
			//default case
			op = "?"
			e = nil
			tmp := p.FSet.Position(s.Pos())
			q, r := filepath.Abs(tmp.Filename)
			q = tmp.Filename
			if r != nil {
				panic(r)
			}
			pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

			sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PostRcv")}
			//sel3 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetGID")}
			tmp2 := p.FSet.Position(comm.Pos())
			pa, r := filepath.Abs(tmp.Filename)
			pa = tmp.Filename
			if r != nil {
				panic(r)
			}
			b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(pa), tmp2.Line)}
			dummy := &ast.BasicLit{Kind: token.INT, Value: "0"}
			//commit := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{ast.NewIdent("nil"), b, dummy, &ast.CallExpr{Fun: sel3}}}
			commit := &ast.CallExpr{
				Fun:      sel2,
				Lparen:   0,
				Args:     []ast.Expr{ast.NewIdent("nil"), b, dummy, p.getTIDFromCache()},
				Ellipsis: 0,
				Rparen:   0,
			}
			comm.Body = append([]ast.Stmt{&ast.ExprStmt{X: commit}}, comm.Body...)
		} else {
			switch y := comm.Comm.(type) {
			case *ast.SendStmt:
				op = "!"
				e = y.Chan
				tmp := p.FSet.Position(s.Pos())
				q, r := filepath.Abs(tmp.Filename)
				q = tmp.Filename
				if r != nil {
					panic(r)
				}
				pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
				b := &ast.BlockStmt{}
				p.handleSendStmt(y, b)
				// b.list is empty it means that the select case uses a external channel for which no type informations are available.
				// if len(b.List) == 0 {
				// 	continue
				// }

				body := p.handleBlockStmt(&ast.BlockStmt{List: comm.Body})
				tidCache := body.List[0]
				body.List = body.List[1:]
				bTmp := &ast.BlockStmt{List: []ast.Stmt{}}
				p.race_handleSendStmt(y, bTmp)
				body.List = append(bTmp.List, body.List...)

				// b.list is empty means that the select case uses a external channel for which no type informations are available.
				if len(b.List) > 0 {
					comm.Comm = b.List[1]
					if p.WithCommLink {
						comm.Body = append([]ast.Stmt{b.List[2]}, body.List...) //comm.Body...)
					}
				} else {
					comm.Body = body.List
				}
				comm.Body = append([]ast.Stmt{tidCache}, comm.Body...)
				//nBlockStmt.List = append(nBlockStmt.List, b.List[0])
			case *ast.ExprStmt:
				switch z := y.X.(type) {
				case *ast.UnaryExpr:
					op = "?"
					e = z.X
					tmp := p.FSet.Position(s.Pos())
					q, r := filepath.Abs(tmp.Filename)
					q = tmp.Filename
					if r != nil {
						panic(r)
					}
					pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
					b := &ast.BlockStmt{}
					p.handleRcvStmt(z, b)
					// if len(b.List) == 0 {
					// 	continue
					// }

					body := p.handleBlockStmt(&ast.BlockStmt{List: comm.Body})
					tidCache := body.List[0]
					body.List = body.List[1:]
					if len(b.List) > 0 {
						comm.Comm = b.List[1]
						if p.WithCommLink {
							comm.Body = append([]ast.Stmt{b.List[2]}, body.List...) //comm.Body...)
						}
					} else {
						comm.Body = body.List
					}
					comm.Body = append([]ast.Stmt{tidCache}, comm.Body...)
					//	nBlockStmt.List = append(nBlockStmt.List, b.List[0])
				}
			case *ast.AssignStmt:
				op = "?"
				tmp := p.FSet.Position(s.Pos())
				q, r := filepath.Abs(tmp.Filename)
				q = tmp.Filename
				if r != nil {
					panic(r)
				}
				pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
				b := &ast.BlockStmt{}
				p.handleRcvStmt(y.Rhs[0].(*ast.UnaryExpr), b)
				// if len(b.List) == 0 {
				// 	// foobar situation ignore this case and handle manually later
				// 	continue
				// }
				if len(b.List) > 0 {
					e = (b.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)).Args[0]
					comm.Comm = b.List[1]
				}

				body := p.handleBlockStmt(&ast.BlockStmt{List: comm.Body})
				tidCache := body.List[0]
				body.List = body.List[1:]
				bTmp := &ast.BlockStmt{List: []ast.Stmt{}}
				p.handleAssign(y, bTmp)

				if len(bTmp.List) > 1 {
					bTmp.List = bTmp.List[2:]
					y.Rhs[0] = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
					body.List = append(bTmp.List, body.List...)
				}

				if len(b.List) > 0 {
					if p.WithCommLink {
						comm.Body = append([]ast.Stmt{b.List[2], y}, body.List...) //comm.Body...)
					}
				} else {
					comm.Body = body.List
				}
				comm.Body = append([]ast.Stmt{tidCache}, comm.Body...)
				//nBlockStmt.List = append(nBlockStmt.List, b.List[0])
			}
		}
		sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("SelectEv")}
		b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v\"", op)}

		if e != nil {
			args = append(args, &ast.CompositeLit{Type: sel, Elts: []ast.Expr{e, b, pos}})
		} else {
			args = append(args, &ast.CompositeLit{Type: sel, Elts: []ast.Expr{ast.NewIdent("nil"), b, pos}})
		}
	}
	args = append([]ast.Expr{p.getGID()}, args...)
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("PreSelect")}
	prep := &ast.CallExpr{Fun: sel, Args: args}
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})
	nBlockStmt.List = append(nBlockStmt.List, s)
}

func (p *ASTParser) isChanMake(x ast.Expr) *ast.CallExpr {
	if fun, ok := x.(*ast.CallExpr); ok {
		if n, ok := fun.Fun.(*ast.Ident); ok && n.Name == "make" {
			switch fun.Args[0].(type) {
			case *ast.ChanType:
				return fun
			default:
				fmt.Println("inSwichIsChamMake", reflect.TypeOf(fun.Args[0]), p.FSet.Position(x.Pos()).Line)
			}
		} else {
			if ok {
				pos := p.FSet.Position(x.Pos())
				fmt.Println("check isChanMake:", reflect.TypeOf(fun.Fun), n.Name, pos.Line)
			}
		}
	}
	return nil
}

func (p *ASTParser) getType(x ast.Expr) ast.Expr {
	switch y := x.(type) {
	case *ast.BasicLit:
		switch y.Kind {
		case token.INT:
			return ast.NewIdent("int")
		case token.FLOAT:
			return ast.NewIdent("float64")
		case token.CHAR:
			return ast.NewIdent("byte")
		case token.STRING:
			return ast.NewIdent("string")
		}
	case *ast.Ident:
		if y.Obj != nil {
			switch z := y.Obj.Decl.(type) {
			case *ast.Field:
				return z.Type
			case *ast.AssignStmt:
				if call, ok := z.Rhs[0].(*ast.CallExpr); ok {
					if len(call.Args) > 0 {
						if c, ok := call.Args[0].(*ast.ChanType); ok {
							return c
						}
					}
				}
			}
		}
	default:
		fmt.Println("getType??", reflect.TypeOf(x))
	}

	return x
}

func (p *ASTParser) escapeSlashes(s string) string {
	return strings.Replace(s, "\\", "\\\\", -1)
}

func (p *ASTParser) getGID() *ast.CallExpr {
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetGID")}
	call := &ast.CallExpr{Fun: sel}
	return call
}
