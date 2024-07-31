/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ast

import (
	"encoding/json"
	"strings"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type FunctionPurity int

const (
	FunctionPurityUnspecified FunctionPurity = iota
	FunctionPurityView
)

func (p FunctionPurity) Keyword() string {
	switch p {
	case FunctionPurityUnspecified:
		return ""
	case FunctionPurityView:
		return "view"
	default:
		panic(errors.NewUnreachableError())
	}
}

func (p FunctionPurity) MarshalJSON() ([]byte, error) {
	switch p {
	case FunctionPurityUnspecified:
		return json.Marshal("Unspecified")
	case FunctionPurityView:
		return json.Marshal("View")
	default:
		panic(errors.NewUnreachableError())
	}
}

type FunctionDeclarationFlags uint8

const (
	FunctionDeclarationFlagsIsStatic FunctionDeclarationFlags = 1 << iota
	FunctionDeclarationFlagsIsNative
)

type FunctionDeclaration struct {
	Purity               FunctionPurity
	TypeParameterList    *TypeParameterList
	ParameterList        *ParameterList
	ReturnTypeAnnotation *TypeAnnotation
	FunctionBlock        *FunctionBlock
	// TODO(preserve-comments): Replace with DeclarationDocString method, since doc string is computed from Comments struct
	DocString  string
	Identifier Identifier
	StartPos   Position `json:"-"`
	Access     Access
	Flags      FunctionDeclarationFlags
	Comments
}

var _ Element = &FunctionDeclaration{}
var _ Declaration = &FunctionDeclaration{}
var _ Statement = &FunctionDeclaration{}

// TODO(preserve-comments): Temporary, add `comments` param to NewFunctionDeclaration in the future
func NewFunctionDeclarationWithComments(
	gauge common.MemoryGauge,
	access Access,
	purity FunctionPurity,
	isStatic bool,
	isNative bool,
	identifier Identifier,
	typeParameterList *TypeParameterList,
	parameterList *ParameterList,
	returnTypeAnnotation *TypeAnnotation,
	functionBlock *FunctionBlock,
	startPos Position,
	docString string,
	comments Comments,
) *FunctionDeclaration {
	decl := NewFunctionDeclaration(
		gauge,
		access,
		purity,
		isStatic,
		isNative,
		identifier,
		typeParameterList,
		parameterList,
		returnTypeAnnotation,
		functionBlock,
		startPos,
		docString,
	)
	decl.Comments = comments
	return decl
}

func NewFunctionDeclaration(
	gauge common.MemoryGauge,
	access Access,
	purity FunctionPurity,
	isStatic bool,
	isNative bool,
	identifier Identifier,
	typeParameterList *TypeParameterList,
	parameterList *ParameterList,
	returnTypeAnnotation *TypeAnnotation,
	functionBlock *FunctionBlock,
	startPos Position,
	docString string,
) *FunctionDeclaration {
	common.UseMemory(gauge, common.FunctionDeclarationMemoryUsage)

	var flags FunctionDeclarationFlags
	if isStatic {
		flags |= FunctionDeclarationFlagsIsStatic
	}
	if isNative {
		flags |= FunctionDeclarationFlagsIsNative
	}

	return &FunctionDeclaration{
		Access:               access,
		Purity:               purity,
		Flags:                flags,
		Identifier:           identifier,
		TypeParameterList:    typeParameterList,
		ParameterList:        parameterList,
		ReturnTypeAnnotation: returnTypeAnnotation,
		FunctionBlock:        functionBlock,
		StartPos:             startPos,
		DocString:            docString,
	}
}

func (*FunctionDeclaration) isDeclaration() {}

func (*FunctionDeclaration) isStatement() {}

func (*FunctionDeclaration) ElementType() ElementType {
	return ElementTypeFunctionDeclaration
}

func (d *FunctionDeclaration) StartPosition() Position {
	return d.StartPos
}

func (d *FunctionDeclaration) EndPosition(memoryGauge common.MemoryGauge) Position {
	if d.FunctionBlock != nil {
		return d.FunctionBlock.EndPosition(memoryGauge)
	}
	if d.ReturnTypeAnnotation != nil {
		return d.ReturnTypeAnnotation.EndPosition(memoryGauge)
	}
	return d.ParameterList.EndPosition(memoryGauge)
}

func (d *FunctionDeclaration) Walk(walkChild func(Element)) {
	// TODO: walk parameters
	// TODO: walk return type
	if d.FunctionBlock != nil {
		walkChild(d.FunctionBlock)
	}
}

func (d *FunctionDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *FunctionDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindFunction
}

func (d *FunctionDeclaration) DeclarationAccess() Access {
	return d.Access
}

func (d *FunctionDeclaration) ToExpression(memoryGauge common.MemoryGauge) *FunctionExpression {
	return NewFunctionExpression(
		memoryGauge,
		d.Purity,
		d.ParameterList,
		d.ReturnTypeAnnotation,
		d.FunctionBlock,
		d.StartPos,
	)
}

func (d *FunctionDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *FunctionDeclaration) DeclarationDocString() string {
	var s strings.Builder
	for _, comment := range d.Comments.Leading {
		if comment.Doc() {
			if s.Len() > 0 {
				s.WriteRune('\n')
			}
			s.Write(comment.Text())
		}
	}
	return s.String()
}

func (d *FunctionDeclaration) Doc() prettier.Doc {
	return FunctionDocument(
		d.Access,
		d.Purity,
		d.IsStatic(),
		d.IsNative(),
		true,
		d.Identifier.Identifier,
		d.TypeParameterList,
		d.ParameterList,
		d.ReturnTypeAnnotation,
		d.FunctionBlock,
	)
}

func (d *FunctionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias FunctionDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
		Range
		IsStatic bool
		IsNative bool
		Flags    FunctionDeclarationFlags `json:",omitempty"`
	}{
		Type:     "FunctionDeclaration",
		Range:    NewUnmeteredRangeFromPositioned(d),
		IsStatic: d.IsStatic(),
		IsNative: d.IsNative(),
		Alias:    (*Alias)(d),
		Flags:    0,
	})
}

func (d *FunctionDeclaration) String() string {
	return Prettier(d)
}

func (d *FunctionDeclaration) IsStatic() bool {
	return d.Flags&FunctionDeclarationFlagsIsStatic != 0
}

func (d *FunctionDeclaration) IsNative() bool {
	return d.Flags&FunctionDeclarationFlagsIsNative != 0
}

// SpecialFunctionDeclaration

type SpecialFunctionDeclaration struct {
	FunctionDeclaration *FunctionDeclaration
	Kind                common.DeclarationKind
}

var _ Element = &SpecialFunctionDeclaration{}
var _ Declaration = &SpecialFunctionDeclaration{}
var _ Statement = &SpecialFunctionDeclaration{}

func NewSpecialFunctionDeclaration(
	gauge common.MemoryGauge,
	kind common.DeclarationKind,
	funcDecl *FunctionDeclaration,
) *SpecialFunctionDeclaration {
	common.UseMemory(gauge, common.SpecialFunctionDeclarationMemoryUsage)

	return &SpecialFunctionDeclaration{
		Kind:                kind,
		FunctionDeclaration: funcDecl,
	}

}
func (*SpecialFunctionDeclaration) isDeclaration() {}

func (*SpecialFunctionDeclaration) isStatement() {}

func (*SpecialFunctionDeclaration) ElementType() ElementType {
	return ElementTypeSpecialFunctionDeclaration
}

func (d *SpecialFunctionDeclaration) StartPosition() Position {
	return d.FunctionDeclaration.StartPosition()
}

func (d *SpecialFunctionDeclaration) EndPosition(memoryGauge common.MemoryGauge) Position {
	return d.FunctionDeclaration.EndPosition(memoryGauge)
}

func (d *SpecialFunctionDeclaration) Walk(walkChild func(Element)) {
	d.FunctionDeclaration.Walk(walkChild)
}

func (d *SpecialFunctionDeclaration) DeclarationIdentifier() *Identifier {
	return d.FunctionDeclaration.DeclarationIdentifier()
}

func (d *SpecialFunctionDeclaration) DeclarationKind() common.DeclarationKind {
	return d.Kind
}

func (d *SpecialFunctionDeclaration) DeclarationAccess() Access {
	return d.FunctionDeclaration.DeclarationAccess()
}

func (d *SpecialFunctionDeclaration) DeclarationMembers() *Members {
	return d.FunctionDeclaration.DeclarationMembers()
}

func (d *SpecialFunctionDeclaration) DeclarationDocString() string {
	return d.FunctionDeclaration.DeclarationDocString()
}

func (d *SpecialFunctionDeclaration) Doc() prettier.Doc {
	return FunctionDocument(
		d.FunctionDeclaration.Access,
		d.FunctionDeclaration.Purity,
		d.FunctionDeclaration.IsStatic(),
		d.FunctionDeclaration.IsNative(),
		false,
		d.Kind.Keywords(),
		d.FunctionDeclaration.TypeParameterList,
		d.FunctionDeclaration.ParameterList,
		d.FunctionDeclaration.ReturnTypeAnnotation,
		d.FunctionDeclaration.FunctionBlock,
	)
}

func (d *SpecialFunctionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias SpecialFunctionDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
		Range
	}{
		Type:  "SpecialFunctionDeclaration",
		Range: NewUnmeteredRangeFromPositioned(d),
		Alias: (*Alias)(d),
	})
}

func (d *SpecialFunctionDeclaration) String() string {
	return Prettier(d)
}
