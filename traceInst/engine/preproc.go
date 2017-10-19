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
	FSet    *token.FileSet
	File    *ast.File
	Path    string
	outPath string
	nameID  int
	funID   int
}

//NewASTParser creates a new ASTParser
func NewASTParser(p, out string) *ASTParser {
	ex := &ASTParser{}
	ex.FSet = token.NewFileSet()
	ex.Path = p
	astf, err := parser.ParseFile(ex.FSet, ex.Path, nil, 0)
	if err != nil {
		panic(err)
	}
	ex.File = astf
	ex.outPath = out
	return ex
}

//Run does something
func (p *ASTParser) Run() {
	for _, d := range p.File.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			p.handleFuncDecl(x)
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
	p.convertParams(fun)
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

func (p *ASTParser) handleAssign(assign *ast.AssignStmt, nBlockStmt *ast.BlockStmt) {
	//if rhs is empty, then there is nothing todo here since there aren't any channels involved
	if len(assign.Rhs) == 0 {
		return
	}
	var isMake bool
	if call := p.isChanMake(assign.Rhs[0]); call != nil {
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
		isMake = true
	}

	switch y := assign.Rhs[0].(type) {
	case *ast.UnaryExpr:
		if y.Op == token.ARROW {
			p.handleRcvStmt(y, nBlockStmt)
			assign.Rhs[0] = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	case *ast.BinaryExpr:
		p.handleBinaryExpr(y, nBlockStmt)
	case *ast.ParenExpr:
		if p.handleParenExpr(y, nBlockStmt) {
			assign.Rhs[0] = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	}

	if raceDetector && !isMake /*&& assign.Tok != token.DEFINE*/ {
		// sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("WriteAcc")}
		// call := &ast.CallExpr{Fun: sel, Args: assign.Lhs}
		tmp := p.getWriteAcc(assign.Lhs[0])
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: tmp})
	}

	if raceDetector && !isMake {
		// check right hand side for read access
		for _, r := range assign.Rhs {
			switch x := r.(type) {
			case *ast.CallExpr:
				p.race_handleCallExpr(x, nBlockStmt)
			case *ast.UnaryExpr:
				p.race_handleUnaryExpr(x, nBlockStmt)
			case *ast.BinaryExpr:
				p.race_handleBinaryExpr(x, nBlockStmt)
			}
		}
	}

	nBlockStmt.List = append(nBlockStmt.List, assign)
	if isMake {
		sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RegisterChan")}
		capCall := &ast.CallExpr{Fun: ast.NewIdent("cap"), Args: []ast.Expr{assign.Lhs[0]}}
		call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{assign.Lhs[0], capCall}}
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: call})
	}
}

func (p *ASTParser) handleParenExpr(x *ast.ParenExpr, nBlockStmt *ast.BlockStmt) bool {
	if p.isRcv(x.X) {
		nBlock := &ast.BlockStmt{}
		p.handleRcvStmt(x.X.(*ast.UnaryExpr), nBlock)
		nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
		x.X = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		return true
	}
	if v, ok := x.X.(*ast.BinaryExpr); ok {
		p.handleBinaryExpr(v, nBlockStmt)
		x.X = v
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
			nBlockStmt.List = append(nBlockStmt.List, b.List...)
			x.X = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	case *ast.BinaryExpr:
		p.handleBinaryExpr(v, nBlockStmt)
	case *ast.ParenExpr:
		nBlock := &ast.BlockStmt{}
		if p.handleParenExpr(v, nBlock) {
			nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
			//x.X = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	}

	switch v := x.Y.(type) {
	case *ast.UnaryExpr:
		if v.Op == token.ARROW {
			b := &ast.BlockStmt{}
			p.handleRcvStmt(v, b)
			nBlockStmt.List = append(nBlockStmt.List, b.List...)
			x.Y = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	case *ast.BinaryExpr:
		p.handleBinaryExpr(v, nBlockStmt)
	case *ast.ParenExpr:
		nBlock := &ast.BlockStmt{}
		if p.handleParenExpr(v, nBlock) {
			nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
			//	x.Y = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
		}
	}
	//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: x})
}

func (p *ASTParser) race_handleBinaryExpr(x *ast.BinaryExpr, nBlockStmt *ast.BlockStmt) {
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
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
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
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(v)})
	}
}
func (p *ASTParser) race_handleUnaryExpr(x *ast.UnaryExpr, nBlockStmt *ast.BlockStmt) {
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
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getWriteAcc(v)})
	}
}

func (p *ASTParser) race_handleSendStmt(x *ast.SendStmt, nBlockStmt *ast.BlockStmt) {
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getReadAcc(x.Value)})
}

func (p *ASTParser) race_handleCallExpr(x *ast.CallExpr, nBlockStmt *ast.BlockStmt) {
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
		}
	}
}

func (p *ASTParser) race_handleIncDec(x ast.Expr, nBlockStmt *ast.BlockStmt) {
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: p.getWriteAcc(x)})
}

func (p *ASTParser) getReadAcc(e ast.Expr) *ast.CallExpr {
	tmp := p.FSet.Position(e.Pos())
	q, r := filepath.Abs(tmp.Filename)
	if r != nil {
		panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("ReadAcc")}
	call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{e, b}}
	return call
}
func (p *ASTParser) getWriteAcc(e ast.Expr) *ast.CallExpr {
	fmt.Println("!!!!")
	tmp := p.FSet.Position(e.Pos())
	q, r := filepath.Abs(tmp.Filename)
	if r != nil {
		panic(r)
	}
	fmt.Println("..", tmp)
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("WriteAcc")}
	call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{e, b}}
	return call
}

func (p *ASTParser) handleSendStmt(x *ast.SendStmt, nBlockStmt *ast.BlockStmt) {
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("SendPrep")}
	sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("SendCommit")}
	sel3 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetGID")}
	gpid := &ast.CallExpr{Fun: sel3}

	tmp := p.FSet.Position(x.Pos())
	q, r := filepath.Abs(tmp.Filename)
	if r != nil {
		panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
	prep := &ast.CallExpr{Fun: sel, Args: []ast.Expr{x.Chan, b}}
	commit := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{x.Chan, b}}

	var t ast.Expr
	if chanT, ok := p.getType(x.Chan).(*ast.ChanType); ok {
		if s, ok := chanT.Value.(*ast.StructType); ok {
			t = s.Fields.List[1].Type
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
	nBlockStmt.List = append(nBlockStmt.List, nChan)
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: commit})
}

func (p *ASTParser) handleRcvStmt(x *ast.UnaryExpr, nBlockStmt *ast.BlockStmt) {
	p.nameID++
	//	chanT := p.getType(x.X).(*ast.ChanType)

	var t ast.Expr
	if chanT, ok := p.getType(x.X).(*ast.ChanType); ok {
		if s, ok := chanT.Value.(*ast.StructType); ok {
			t = s.Fields.List[1].Type
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
	}

	nStruct := &ast.StructType{Fields: &ast.FieldList{List: make([]*ast.Field, 0)}}
	nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: ast.NewIdent("uint64"), Names: []*ast.Ident{ast.NewIdent("threadId")}})
	nStruct.Fields.List = append(nStruct.Fields.List, &ast.Field{Type: p.getType(t), Names: []*ast.Ident{ast.NewIdent("value")}})

	assign := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID))}, Tok: token.DEFINE, Rhs: []ast.Expr{x}}

	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RcvPrep")}
	sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RcvCommit")}
	sel3 := &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("threadId")}

	tmp := p.FSet.Position(x.Pos())
	q, r := filepath.Abs(tmp.Filename)
	if r != nil {
		panic(r)
	}
	b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

	prep := &ast.CallExpr{Fun: sel, Args: []ast.Expr{x.X, b}}
	commit := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{x.X, b, sel3}}

	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})
	nBlockStmt.List = append(nBlockStmt.List, assign)
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: commit})
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
	block := &ast.BlockStmt{List: append([]ast.Stmt{tmp.List[0], tmp.List[1], tmp.List[2], ifstmt, assign}, f.Body.List...)}
	forstmt := &ast.ForStmt{Body: block}
	return forstmt
}

func (p *ASTParser) handleBlockStmt(in *ast.BlockStmt) *ast.BlockStmt {
	out := &ast.BlockStmt{Lbrace: in.Lbrace, Rbrace: in.Rbrace, List: make([]ast.Stmt, 0)}
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
				if raceDetector {
					p.race_handleCallExpr(y, out)
				}
				p.handleClose(y, out)
				p.handleRegCalls(y, out)
			//	out.List = append(out.List, x)
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
			out.List = append(out.List, p.handleIfStmt(x))
		case *ast.ForStmt:
			out.List = append(out.List, p.handleForStmt(x))
		case *ast.RangeStmt:
			out.List = append(out.List, p.handleRangeStmt(x))
		case *ast.SelectStmt:
			p.handleSelect(x, out)
		case *ast.GoStmt:
			//get new sigWaitID
			p.nameID++
			varName := fmt.Sprintf("tmp%v", p.nameID)
			getSWIDSel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetWaitSigID")}
			callgetSWID := &ast.CallExpr{Fun: getSWIDSel, Args: make([]ast.Expr, 0)}
			ass := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(varName)}, Rhs: []ast.Expr{callgetSWID}, Tok: token.DEFINE}
			out.List = append(out.List, ass)

			sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("AddSignal")}
			call2 := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{ast.NewIdent(varName)}}
			out.List = append(out.List, &ast.ExprStmt{X: call2})

			var funName string
			anno := false
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
								if n, ok := x.Call.Args[i].(*ast.Ident); ok {
									nf = &ast.Field{Type: f.Type, Names: []*ast.Ident{n}}
								} else if _, ok := x.Call.Args[i].(*ast.BasicLit); ok {
									//fmt.Println("tada!!", f.Type, f.Names)
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
			} else {
				//anonymous function
				funName = fmt.Sprintf("\"fun%v\"", p.funID)
				p.funID++
				anno = true
			}

			b := &ast.BasicLit{Kind: token.STRING, Value: funName}
			sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RegisterThread")}
			call := &ast.CallExpr{Fun: sel, Args: []ast.Expr{b, ast.NewIdent(varName)}}
			ft := &ast.FuncType{Params: &ast.FieldList{List: list}}
			var body *ast.BlockStmt
			if !anno {
				body = &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: call}, &ast.ExprStmt{X: x.Call}}}
			} else {
				fl := x.Call.Fun.(*ast.FuncLit)
				list := p.handleBlockStmt(fl.Body).List
				body = &ast.BlockStmt{List: append([]ast.Stmt{&ast.ExprStmt{X: call}}, list...)}
			}
			fl := &ast.FuncLit{Body: body, Type: ft}
			x.Call = &ast.CallExpr{Fun: fl, Args: x.Call.Args}

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
		default:
			fmt.Println(">>>", reflect.TypeOf(x), p.FSet.Position(x.Pos()).Line)
			out.List = append(out.List, x)
		}
	}
	return out
}

func (p *ASTParser) handleRegCalls(c *ast.CallExpr, nBlockStmt *ast.BlockStmt) {
	for i, arg := range c.Args {
		if p.isRcv(arg) {
			nBlock := &ast.BlockStmt{}
			p.handleRcvStmt(arg.(*ast.UnaryExpr), nBlock)

			nBlockStmt.List = append(nBlockStmt.List, nBlock.List...)
			sel := &ast.SelectorExpr{X: nBlock.List[1].(*ast.AssignStmt).Lhs[0], Sel: ast.NewIdent("value")}
			c.Args[i] = sel
		}
	}
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: c})
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

func (p *ASTParser) handleClose(c *ast.CallExpr, nBlockStmt *ast.BlockStmt) {
	if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "close" {
		sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("ClosePrep")}
		sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("CloseCommit")}

		tmp := p.FSet.Position(c.Pos())
		q, r := filepath.Abs(tmp.Filename)
		if r != nil {
			panic(r)
		}
		b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

		prep := &ast.CallExpr{Fun: sel, Args: append(c.Args, b)}
		com := &ast.CallExpr{Fun: sel2, Args: append(c.Args, b)}

		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: c})
		nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: com})
		return
	}
	//nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: c})
}

func (p *ASTParser) handleSelect(s *ast.SelectStmt, nBlockStmt *ast.BlockStmt) {
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
			if r != nil {
				panic(r)
			}
			pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}

			sel2 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("RcvCommit")}
			sel3 := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("GetGID")}
			tmp2 := p.FSet.Position(comm.Pos())
			pa, r := filepath.Abs(tmp.Filename)
			if r != nil {
				panic(r)
			}
			b := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(pa), tmp2.Line)}
			commit := &ast.CallExpr{Fun: sel2, Args: []ast.Expr{ast.NewIdent("nil"), b, &ast.CallExpr{Fun: sel3}}}
			comm.Body = append([]ast.Stmt{&ast.ExprStmt{X: commit}}, comm.Body...)
		} else {
			switch y := comm.Comm.(type) {
			case *ast.SendStmt:
				op = "!"
				e = y.Chan
				tmp := p.FSet.Position(s.Pos())
				q, r := filepath.Abs(tmp.Filename)
				if r != nil {
					panic(r)
				}
				pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
				b := &ast.BlockStmt{}
				p.handleSendStmt(y, b)
				comm.Comm = b.List[1]
				body := p.handleBlockStmt(&ast.BlockStmt{List: comm.Body})
				comm.Body = append([]ast.Stmt{b.List[2]}, body.List...) //comm.Body...)
				//nBlockStmt.List = append(nBlockStmt.List, b.List[0])
			case *ast.ExprStmt:
				switch z := y.X.(type) {
				case *ast.UnaryExpr:
					op = "?"
					e = z.X
					tmp := p.FSet.Position(s.Pos())
					q, r := filepath.Abs(tmp.Filename)
					if r != nil {
						panic(r)
					}
					pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
					b := &ast.BlockStmt{}
					p.handleRcvStmt(z, b)
					comm.Comm = b.List[1]
					body := p.handleBlockStmt(&ast.BlockStmt{List: comm.Body})
					comm.Body = append([]ast.Stmt{b.List[2]}, body.List...) //comm.Body...)
					//	nBlockStmt.List = append(nBlockStmt.List, b.List[0])
				}
			case *ast.AssignStmt:
				op = "?"
				tmp := p.FSet.Position(s.Pos())
				q, r := filepath.Abs(tmp.Filename)
				if r != nil {
					panic(r)
				}
				pos = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%v:%v\"", p.escapeSlashes(q), tmp.Line)}
				b := &ast.BlockStmt{}
				p.handleRcvStmt(y.Rhs[0].(*ast.UnaryExpr), b)
				e = (b.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)).Args[0]
				comm.Comm = b.List[1]

				y.Rhs[0] = &ast.SelectorExpr{X: ast.NewIdent(fmt.Sprintf("tmp%v", p.nameID)), Sel: ast.NewIdent("value")}
				body := p.handleBlockStmt(&ast.BlockStmt{List: comm.Body})
				comm.Body = append([]ast.Stmt{b.List[2], y}, body.List...) //comm.Body...)
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
	sel := &ast.SelectorExpr{X: ast.NewIdent("tracer"), Sel: ast.NewIdent("SelectPrep")}
	prep := &ast.CallExpr{Fun: sel, Args: args}
	nBlockStmt.List = append(nBlockStmt.List, &ast.ExprStmt{X: prep})
	nBlockStmt.List = append(nBlockStmt.List, s)
}

func (p *ASTParser) isChanMake(x ast.Expr) *ast.CallExpr {
	if fun, ok := x.(*ast.CallExpr); ok {
		if n, ok := fun.Fun.(*ast.Ident); ok {
			if n.Name == "make" {
				return fun
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
					if c, ok := call.Args[0].(*ast.ChanType); ok {
						return c
					}
				}
			}
		}
	default:
		fmt.Println("??", reflect.TypeOf(x))
	}

	return x
}

func (p *ASTParser) escapeSlashes(s string) string {
	return strings.Replace(s, "\\", "\\\\", -1)
}
