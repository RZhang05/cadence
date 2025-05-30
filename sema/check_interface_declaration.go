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

package sema

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
)

// VisitInterfaceDeclaration checks the given interface declaration.
//
// NOTE: This function assumes that the interface type was previously declared using
// `declareInterfaceType` and exists in `checker.Elaboration.InterfaceDeclarationTypes`,
// and that the members and nested declarations for the interface type were declared
// through `declareInterfaceMembers`.
func (checker *Checker) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) (_ struct{}) {

	wasInInterface := checker.inInterface
	checker.inInterface = true
	defer func() {
		checker.inInterface = wasInInterface
	}()

	const kind = ContainerKindInterface
	interfaceType := checker.Elaboration.InterfaceDeclarationType(declaration)
	if interfaceType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[interfaceType] = true
	defer func() {
		checker.containerTypes[interfaceType] = false
	}()

	checker.checkDeclarationAccessModifier(
		checker.accessFromAstAccess(declaration.Access),
		declaration.DeclarationKind(),
		interfaceType,
		nil,
		declaration.StartPos,
		true,
	)

	inheritedMembers := map[string][]*Member{}
	inheritedTypes := map[string]Type{}

	for _, conformance := range interfaceType.EffectiveInterfaceConformances() {
		// If the currently checking type is also in its own conformance list,
		// then this is a direct/indirect cyclic conformance.
		if conformance.InterfaceType == interfaceType {
			checker.report(CyclicConformanceError{
				InterfaceType: interfaceType,
				Range:         ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Identifier),
			})
		}

		checker.checkInterfaceConformance(
			declaration,
			interfaceType,
			conformance.InterfaceType,
			inheritedMembers,
			inheritedTypes,
		)
	}

	// NOTE: functions are checked separately
	checker.checkFieldsAccessModifier(declaration.Members.Fields(), interfaceType.Members, &declaration.CompositeKind)

	checker.checkNestedIdentifiers(declaration.Members)

	// Activate new scope for nested types and values

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	// Declare nested types

	checker.declareInterfaceNestedTypes(declaration)

	for _, nestedCompositeDeclaration := range declaration.Members.Composites() {
		if nestedCompositeDeclaration.Kind() == common.CompositeKindEvent {
			checker.declareCompositeLikeMembersAndValue(nestedCompositeDeclaration)
		}
	}

	checker.checkInitializers(
		declaration.Members.Initializers(),
		declaration.Members.Fields(),
		interfaceType,
		declaration.DeclarationDocString(),
		interfaceType.InitializerPurity,
		interfaceType.InitializerParameters,
		kind,
		nil,
	)

	checker.checkUnknownSpecialFunctions(declaration.Members.SpecialFunctions())
	checker.checkSpecialFunctionDefaultImplementation(
		declaration,
		declaration.DeclarationKind().Name(),
	)

	checker.checkInterfaceFunctions(
		declaration.Members.Functions(),
		interfaceType,
		declaration.DeclarationKind(),
		&declaration.CompositeKind,
		declaration.DeclarationDocString(),
	)

	fieldPositionGetter := func(name string) ast.Position {
		return interfaceType.FieldPosition(name, declaration)
	}

	checker.checkResourceFieldNesting(
		interfaceType.Members,
		interfaceType.CompositeKind,
		nil,
		fieldPositionGetter,
	)

	if !interfaceType.IsResourceType() && interfaceType.DefaultDestroyEvent != nil {
		checker.report(&DefaultDestroyEventInNonResourceError{
			Kind:  declaration.DeclarationKind().Name(),
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, declaration),
		})
	}

	// NOTE: visit entitlements, then interfaces, then composites
	// DON'T use `nestedDeclarations`, because of non-deterministic order

	for _, nestedEntitlement := range declaration.Members.Entitlements() {
		ast.AcceptDeclaration[struct{}](nestedEntitlement, checker)
	}

	for _, nestedInterface := range declaration.Members.Interfaces() {
		ast.AcceptDeclaration[struct{}](nestedInterface, checker)
	}

	for _, nestedComposite := range declaration.Members.Composites() {
		// only event types may be declared in interfaces.
		// However, the error will be reported later in `declareNestedDeclarations``
		if nestedComposite.Kind() == common.CompositeKindEvent {
			checker.visitCompositeLikeDeclaration(nestedComposite)
		}
		if interfaceType.DefaultDestroyEvent != nil {
			checker.checkDefaultDestroyEvent(interfaceType.DefaultDestroyEvent, nestedComposite, interfaceType, declaration)
		}
	}

	return
}

// declareInterfaceNestedTypes declares the types nested in an interface.
// It is used when declaring the interface's members (`declareInterfaceMembers`)
// and checking the interface declaration (`VisitInterfaceDeclaration`).
//
// It assumes the types were previously added to the elaboration in `InterfaceNestedDeclarations`,
// and the type for the declaration was added to the elaboration in `InterfaceDeclarationTypes`.
func (checker *Checker) declareInterfaceNestedTypes(
	declaration *ast.InterfaceDeclaration,
) {

	interfaceType := checker.Elaboration.InterfaceDeclarationType(declaration)
	nestedDeclarations := checker.Elaboration.InterfaceNestedDeclarations(declaration)

	interfaceType.NestedTypes.Foreach(func(name string, nestedType Type) {
		nestedDeclaration := nestedDeclarations[name]

		identifier := nestedDeclaration.DeclarationIdentifier()
		if identifier == nil {
			// It should be impossible to have a nested declaration
			// that does not have an identifier

			panic(errors.NewUnreachableError())
		}

		_, err := checker.typeActivations.declareType(typeDeclaration{
			identifier:               *identifier,
			ty:                       nestedType,
			declarationKind:          nestedDeclaration.DeclarationKind(),
			access:                   checker.accessFromAstAccess(nestedDeclaration.DeclarationAccess()),
			docString:                nestedDeclaration.DeclarationDocString(),
			allowOuterScopeShadowing: false,
		})
		checker.report(err)
	})
}

func (checker *Checker) checkInterfaceFunctions(
	functions []*ast.FunctionDeclaration,
	selfType NominalType,
	declarationKind common.DeclarationKind,
	compositeKind *common.CompositeKind,
	selfDocString string,
) {
	for _, function := range functions {
		// NOTE: new activation, as function declarations
		// shouldn't be visible in other function declarations,
		// and `self` is only visible inside function

		func() {
			checker.enterValueScope()
			defer checker.leaveValueScope(function.EndPosition, false)

			fnAccess := checker.effectiveMemberAccess(checker.accessFromAstAccess(function.Access), ContainerKindInterface)

			checker.declareSelfValue(fnAccess, selfType, selfDocString)

			mustExit := false
			checkResourceLoss := false

			if function.FunctionBlock != nil {
				if function.FunctionBlock.HasStatements() {
					mustExit = true
					checkResourceLoss = true
				} else if function.FunctionBlock.PreConditions.IsEmpty() &&
					function.FunctionBlock.PostConditions.IsEmpty() {

					checker.report(
						&InvalidImplementationError{
							Pos:             function.FunctionBlock.StartPosition(),
							ContainerKind:   declarationKind,
							ImplementedKind: common.DeclarationKindFunction,
						},
					)
				}
			}

			checker.visitFunctionDeclaration(
				function,
				functionDeclarationOptions{
					mustExit:          mustExit,
					declareFunction:   false,
					checkResourceLoss: checkResourceLoss,
				},
				compositeKind,
			)
		}()
	}
}

// declareInterfaceType declares the type for the given interface declaration
// and records it in the elaboration. It also recursively declares all types
// for all nested declarations.
//
// NOTE: The function does *not* declare any members
//
// See `declareInterfaceMembers` for the declaration of the interface type members.
// See `VisitInterfaceDeclaration` for the checking of the interface declaration.
func (checker *Checker) declareInterfaceType(declaration *ast.InterfaceDeclaration) *InterfaceType {

	identifier := declaration.Identifier

	interfaceType := &InterfaceType{
		Location:      checker.Location,
		Identifier:    identifier.Identifier,
		CompositeKind: declaration.CompositeKind,
		NestedTypes:   &StringTypeOrderedMap{},
		Members:       &StringMemberOrderedMap{},
	}

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       interfaceType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   checker.accessFromAstAccess(declaration.Access),
		docString:                declaration.DocString,
		allowOuterScopeShadowing: false,
	})
	checker.report(err)
	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(
			identifier.Identifier,
			variable,
		)
	}

	checker.Elaboration.SetInterfaceDeclarationWithType(declaration, interfaceType)

	if !declaration.CompositeKind.SupportsInterfaces() {
		checker.report(
			&InvalidInterfaceDeclarationError{
				CompositeKind: declaration.CompositeKind,
				Range:         ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Identifier),
			},
		)
	}

	// Activate new scope for nested declarations

	checker.typeActivations.Enter()
	defer checker.typeActivations.Leave(declaration.EndPosition)

	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, false)

	// Check and declare nested types

	nestedDeclarations, nestedInterfaceTypes, nestedCompositeTypes, nestedEntitlementTypes, nestedEntitlementMapTypes :=
		checker.declareNestedDeclarations(
			declaration.CompositeKind,
			declaration.DeclarationKind(),
			declaration.Members.Composites(),
			declaration.Members.Attachments(),
			declaration.Members.Interfaces(),
			declaration.Members.Entitlements(),
			declaration.Members.EntitlementMaps(),
		)

	checker.Elaboration.SetInterfaceNestedDeclarations(declaration, nestedDeclarations)

	for _, nestedEntitlementType := range nestedEntitlementTypes {
		interfaceType.NestedTypes.Set(nestedEntitlementType.Identifier, nestedEntitlementType)
		nestedEntitlementType.SetContainerType(interfaceType)
	}

	for _, nestedEntitlementMapType := range nestedEntitlementMapTypes {
		interfaceType.NestedTypes.Set(nestedEntitlementMapType.Identifier, nestedEntitlementMapType)
		nestedEntitlementMapType.SetContainerType(interfaceType)
	}

	for _, nestedInterfaceType := range nestedInterfaceTypes {
		interfaceType.NestedTypes.Set(nestedInterfaceType.Identifier, nestedInterfaceType)
		nestedInterfaceType.SetContainerType(interfaceType)
	}

	for _, nestedCompositeType := range nestedCompositeTypes {
		interfaceType.NestedTypes.Set(nestedCompositeType.Identifier, nestedCompositeType)
		nestedCompositeType.SetContainerType(interfaceType)
	}

	return interfaceType
}

func (checker *Checker) declareNestedEvent(
	nestedCompositeDeclaration ast.CompositeLikeDeclaration,
	eventMembers *orderedmap.OrderedMap[string, *Member],
	interfaceType Type,
) {
	checker.declareCompositeLikeMembersAndValue(nestedCompositeDeclaration)

	// Declare nested composites' values (constructor/instance) as members of the containing composite
	identifier := *nestedCompositeDeclaration.DeclarationIdentifier()

	// Find the value declaration
	nestedCompositeDeclarationVariable :=
		checker.valueActivations.Find(identifier.Identifier)

	eventMembers.Set(
		nestedCompositeDeclarationVariable.Identifier,
		&Member{
			Identifier:            identifier,
			Access:                checker.accessFromAstAccess(nestedCompositeDeclaration.DeclarationAccess()),
			ContainerType:         interfaceType,
			TypeAnnotation:        NewTypeAnnotation(nestedCompositeDeclarationVariable.Type),
			DeclarationKind:       nestedCompositeDeclarationVariable.DeclarationKind,
			VariableKind:          ast.VariableKindConstant,
			ArgumentLabels:        nestedCompositeDeclarationVariable.ArgumentLabels,
			IgnoreInSerialization: true,
			DocString:             nestedCompositeDeclaration.DeclarationDocString(),
		})
}

// declareInterfaceMembersAndValue declares the members for the given interface declaration,
// and recursively for all nested declarations.
//
// NOTE: This function assumes that the interface type and the nested declarations' types
// were previously declared using `declareInterfaceType` and exists
// in the elaboration's `InterfaceDeclarationTypes` and `InterfaceNestedDeclarations` fields.
func (checker *Checker) declareInterfaceMembersAndValue(declaration *ast.InterfaceDeclaration) {

	interfaceType := checker.Elaboration.InterfaceDeclarationType(declaration)
	if interfaceType == nil {
		panic(errors.NewUnreachableError())
	}

	compositeKind := declaration.Kind()

	declarationMembers := declaration.Members

	eventMembers := orderedmap.New[StringMemberOrderedMap](len(declarationMembers.Composites()))

	(func() { // Activate new scope for nested declarations

		checker.typeActivations.Enter()
		defer checker.typeActivations.Leave(declaration.EndPosition)

		checker.enterValueScope()
		defer checker.leaveValueScope(declaration.EndPosition, false)

		// Declare nested types

		checker.declareInterfaceNestedTypes(declaration)

		// Declare nested types' explicit conformances
		for _, nestedInterfaceDeclaration := range declarationMembers.Interfaces() {
			// resolve conformances
			nestedInterfaceType := checker.Elaboration.InterfaceDeclarationType(nestedInterfaceDeclaration)
			nestedInterfaceType.ExplicitInterfaceConformances =
				checker.explicitInterfaceConformances(nestedInterfaceDeclaration, nestedInterfaceType)
		}

		// Declare members

		members, fields, origins := checker.defaultMembersAndOrigins(
			declarationMembers,
			interfaceType,
			ContainerKindInterface,
			declaration.DeclarationKind(),
		)

		if interfaceType.CompositeKind == common.CompositeKindContract {
			checker.checkMemberStorability(members)
		}

		interfaceType.Members = members
		interfaceType.Fields = fields
		if checker.PositionInfo != nil {
			checker.PositionInfo.recordMemberOrigins(interfaceType, origins)
		}

		// NOTE: determine initializer parameter types while nested types are in scope,
		// and after declaring nested types as the initializer may use nested type in parameters

		initializers := declarationMembers.Initializers()
		interfaceType.InitializerParameters = checker.initializerParameters(initializers)
		interfaceType.InitializerPurity = checker.initializerPurity(compositeKind, initializers)

		// Declare nested declarations' members
		for _, nestedInterfaceDeclaration := range declarationMembers.Interfaces() {
			checker.declareInterfaceMembersAndValue(nestedInterfaceDeclaration)
		}

		for _, nestedCompositeDeclaration := range declarationMembers.Composites() {
			if nestedCompositeDeclaration.Kind() == common.CompositeKindEvent {
				if nestedCompositeDeclaration.IsResourceDestructionDefaultEvent() {

					checker.Elaboration.SetDefaultDestroyDeclaration(declaration, nestedCompositeDeclaration)

					// Find the value declaration
					nestedEvent :=
						checker.typeActivations.Find(nestedCompositeDeclaration.Identifier.Identifier)
					defaultEventComposite, ok := nestedEvent.Type.(*CompositeType)
					if !ok {
						panic(errors.NewUnreachableError())
					}
					interfaceType.DefaultDestroyEvent = defaultEventComposite
				} else {
					checker.declareNestedEvent(nestedCompositeDeclaration, eventMembers, interfaceType)
				}
			}
		}
	})()
}

func (checker *Checker) declareEntitlementType(declaration *ast.EntitlementDeclaration) *EntitlementType {
	identifier := declaration.Identifier

	entitlementType := NewEntitlementType(checker.memoryGauge, checker.Location, identifier.Identifier)

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       entitlementType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   checker.accessFromAstAccess(declaration.Access),
		docString:                declaration.DocString,
		allowOuterScopeShadowing: false,
	})

	checker.report(err)
	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(
			identifier.Identifier,
			variable,
		)
	}

	checker.Elaboration.SetEntitlementDeclarationWithType(declaration, entitlementType)

	return entitlementType
}

func (checker *Checker) VisitEntitlementDeclaration(declaration *ast.EntitlementDeclaration) (_ struct{}) {

	entitlementType := checker.Elaboration.EntitlementDeclarationType(declaration)
	// all entitlement declarations were previously declared in `declareEntitlementType`
	if entitlementType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.checkDeclarationAccessModifier(
		checker.accessFromAstAccess(declaration.Access),
		declaration.DeclarationKind(),
		entitlementType,
		nil,
		declaration.StartPos,
		true,
	)

	return
}

func (checker *Checker) declareEntitlementMappingType(declaration *ast.EntitlementMappingDeclaration) *EntitlementMapType {
	identifier := declaration.Identifier

	entitlementMapType := NewEntitlementMapType(checker.memoryGauge, checker.Location, identifier.Identifier)

	variable, err := checker.typeActivations.declareType(typeDeclaration{
		identifier:               identifier,
		ty:                       entitlementMapType,
		declarationKind:          declaration.DeclarationKind(),
		access:                   checker.accessFromAstAccess(declaration.Access),
		docString:                declaration.DocString,
		allowOuterScopeShadowing: false,
	})

	checker.report(err)
	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(
			identifier.Identifier,
			variable,
		)
	}

	relations := declaration.Relations()

	entitlementRelations := make([]EntitlementRelation, 0, len(relations))

	for _, association := range relations {
		input := checker.convertNominalType(association.Input)
		inputEntitlement, isEntitlement := input.(*EntitlementType)

		if !isEntitlement {
			checker.report(&InvalidNonEntitlementTypeInMapError{
				Pos: association.Input.Identifier.Pos,
			})
			continue
		}

		output := checker.convertNominalType(association.Output)
		outputEntitlement, isEntitlement := output.(*EntitlementType)

		if !isEntitlement {
			checker.report(&InvalidNonEntitlementTypeInMapError{
				Pos: association.Output.Identifier.Pos,
			})
			continue
		}

		entitlementRelations = append(
			entitlementRelations,
			NewEntitlementRelation(checker.memoryGauge, inputEntitlement, outputEntitlement),
		)
	}

	entitlementMapType.Relations = entitlementRelations

	checker.Elaboration.SetEntitlementMapDeclarationWithType(declaration, entitlementMapType)

	return entitlementMapType
}

func (checker *Checker) VisitEntitlementMappingDeclaration(declaration *ast.EntitlementMappingDeclaration) (_ struct{}) {

	entitlementMapType := checker.Elaboration.EntitlementMapDeclarationType(declaration)
	if entitlementMapType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.checkDeclarationAccessModifier(
		checker.accessFromAstAccess(declaration.Access),
		declaration.DeclarationKind(),
		entitlementMapType,
		nil,
		declaration.StartPos,
		true,
	)

	entitlementMapType.resolveEntitlementMappingInclusions(checker, declaration, map[*EntitlementMapType]struct{}{})

	return
}

// checkInterfaceConformance checks the validity of an interface-conformance of an interface declaration.
// It checks for:
//   - Duplicate conformances
//   - Conflicting members (functions, fields, and type definitions)
func (checker *Checker) checkInterfaceConformance(
	interfaceDeclaration *ast.InterfaceDeclaration,
	interfaceType *InterfaceType,
	conformance *InterfaceType,
	inheritedMembersByName map[string][]*Member,
	inheritedNestedTypes map[string]Type,
) {

	// Ensure the composite kinds match, e.g. a structure shouldn't be able
	// to conform to a resource interface
	checker.checkConformanceKindMatch(interfaceDeclaration, interfaceType, conformance)

	// Check for member (functions and fields) conflicts

	conformance.Members.Foreach(func(name string, conformanceMember *Member) {

		var isDuplicate bool

		// Check if the members coming from other conformances (siblings) have conflicts.
		inheritedMembers, ok := inheritedMembersByName[name]
		if ok {
			for _, conflictingMember := range inheritedMembers {
				conflictingInterface := conflictingMember.ContainerType.(*InterfaceType)
				isDuplicate = checker.checkDuplicateInterfaceMember(
					conformance,
					conformanceMember,
					conflictingInterface,
					conflictingMember,
					interfaceDeclaration.Identifier,
					false, // conflicting member is a sibling
				)
			}
		}

		// Check if the members coming from the current declaration have conflicts.
		declarationMember, ok := interfaceType.Members.Get(name)
		if ok {
			isDuplicate = isDuplicate || checker.checkDuplicateInterfaceMember(
				interfaceType,
				declarationMember,
				conformance,
				conformanceMember,
				declarationMember.Identifier,
				true, // conflicting member is an inherited member
			)
		}

		// Add to the inherited members list, only if it's not a duplicated, to avoid redundant errors.
		if !isDuplicate {
			inheritedMembers = append(inheritedMembers, conformanceMember)
			inheritedMembersByName[name] = inheritedMembers
		}
	})
}

func (checker *Checker) checkDuplicateInterfaceMember(
	interfaceType *InterfaceType,
	interfaceMember *Member,
	conflictingInterfaceType *InterfaceType,
	conflictingMember *Member,
	hasPosition ast.HasPosition,
	isConflictingMemberInherited bool,
) (isDuplicate bool) {

	reportMemberConflictError := func() {
		checker.report(&InterfaceMemberConflictError{
			InterfaceType:            interfaceType,
			ConflictingInterfaceType: conflictingInterfaceType,
			MemberName:               interfaceMember.Identifier.Identifier,
			MemberKind:               interfaceMember.DeclarationKind,
			ConflictingMemberKind:    conflictingMember.DeclarationKind,
			Range:                    ast.NewRangeFromPositioned(checker.memoryGauge, hasPosition),
		})

		isDuplicate = true
	}

	// Check if the two members have identical signatures.
	// If not, report an error.
	if !checker.memberSatisfied(interfaceType, interfaceMember, conflictingMember) {
		reportMemberConflictError()
		return
	}

	// If yes, they are allowed, but subject to the conditions below.
	// - Can have at-most one default implementation
	// - A default implementation can only co-exist with a function with conditions
	// i.e. Considering three possibilities for the conflicting functions:
	//   (1) Declaration only: `fun foo()`
	//   (2) Conditions only:  `fun foo() { pre{} }`
	//   (3) Default funcs:    `fun foo() { ... }`
	//
	// Having conflicting identical functions with:
	//  - (1) and (1) - OK
	//  - (1) and (2) - OK
	//  - (1) and (3) - OK
	//  - (2) and (1) - OK
	//  - (2) and (2) - OK
	//  - (2) and (3) - OK
	//  - (3) and (1) - Not OK (order matters)
	//  - (3) and (2) - OK
	//  - (3) and (3) - Not OK

	if interfaceMember.HasImplementation && conflictingMember.HasImplementation {
		reportMemberConflictError()
		return
	}

	// If the conflicting member is an inherited member, it is OK to override
	// the inherited declaration, by a default implementation.
	// However, a default implementation cannot be overridden by an empty declaration.

	if isConflictingMemberInherited &&
		conflictingMember.HasImplementation && !interfaceMember.HasConditions {
		reportMemberConflictError()
		return
	}

	// If the conflicting member is not an inherited one, (i.e.a member from a sibling conformance)
	// then default implementation takes the precedence.

	return
}
