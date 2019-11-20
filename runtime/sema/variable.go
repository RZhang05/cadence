package sema

import (
	"github.com/dapperlabs/flow-go/language/runtime/ast"
	"github.com/dapperlabs/flow-go/language/runtime/common"
)

type Variable struct {
	Identifier      string
	DeclarationKind common.DeclarationKind
	// Type is the type of the variable
	Type Type
	// Access is the access modifier
	Access ast.Access
	// IsConstant indicates if the variable is read-only
	IsConstant bool
	// Depth is the depth of scopes in which the variable was declared
	Depth int
	// ArgumentLabels are the argument labels that must be used in an invocation of the variable
	ArgumentLabels []string
	// Pos is the position where the variable was declared
	Pos *ast.Position
}
