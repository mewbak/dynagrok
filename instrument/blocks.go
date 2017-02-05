package instrument

import (
	"fmt"
	"strings"
	"go/ast"
)

import ()

type block []ast.Stmt

func (blk *block) String() string {
	parts := make([]string, 0, len(*blk))
	for _, item := range *blk {
		parts = append(parts, fmt.Sprintf("%T 0x%x", item, ptr(item)))
	}
	return fmt.Sprintf("[%v]", strings.Join(parts, ", "))
}

func Blocks(blk *[]ast.Stmt, id *int, do func(*[]ast.Stmt, int) error) error {
	var idspot int
	if id == nil {
		id = &idspot
	}
	cId := *id
	(*id)++
	for _, stmt := range *blk {
		v := &blocksVisitor{
			do:do,
			count: id,
		}
		ast.Walk(v, stmt)
		if v.err != nil {
			return v.err
		}
	}
	return do(blk, cId)
}

type blocksVisitor struct {
	err error
	do func(*[]ast.Stmt, int) error
	count *int
}

func (v *blocksVisitor) Visit(n ast.Node) (ast.Visitor) {
	if n == nil || v.err != nil {
		return nil
	}
	var blk *[]ast.Stmt
	switch x := n.(type) {
	case *ast.BlockStmt:
		blk = &x.List
	case *ast.CommClause:
		blk = &x.Body
	case *ast.CaseClause:
		blk = &x.Body
	case *ast.FuncLit:
		return nil
	// prevent putting stmts in blocks that can't recieve them
	case *ast.TypeSwitchStmt:
		for _, stmt := range x.Body.List {
			ast.Walk(v, stmt)
		}
		return nil
	case *ast.SwitchStmt:
		for _, stmt := range x.Body.List {
			ast.Walk(v, stmt)
		}
		return nil
	case *ast.SelectStmt:
		for _, stmt := range x.Body.List {
			ast.Walk(v, stmt)
		}
		return nil
	}
	if blk != nil {
		err := Blocks(blk, v.count, v.do)
		if err != nil {
			v.err = err
		}
		return nil
	}
	return v
}

