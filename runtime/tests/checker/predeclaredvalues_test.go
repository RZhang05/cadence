/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package checker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckPredeclaredValues(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		valueDeclaration := stdlib.StandardLibraryFunction{
			Name: "foo",
			Type: &sema.FunctionType{
				ReturnTypeAnnotation: &sema.TypeAnnotation{
					Type: sema.VoidType,
				},
			},
		}

		_, err := ParseAndCheckWithOptions(t,
			`
            pub fun test() {
                foo()
            }
        `,
			ParseAndCheckOptions{
				Options: []sema.Option{
					sema.WithPredeclaredValues(
						[]sema.ValueDeclaration{
							valueDeclaration,
						},
					),
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("predicated", func(t *testing.T) {

		// Declare four programs.
		// Program 0x1 imports 0x2, 0x3, and 0x4.
		// All programs attempt to call a function 'foo'.
		// Only predeclare a function 'foo' for 0x2 and 0x4.
		// Both functions have the same name, but different types.

		location1 := ast.AddressLocation{
			Address: common.BytesToAddress([]byte{0x1}),
		}

		location2 := ast.AddressLocation{
			Address: common.BytesToAddress([]byte{0x2}),
		}

		location3 := ast.AddressLocation{
			Address: common.BytesToAddress([]byte{0x3}),
		}

		location4 := ast.AddressLocation{
			Address: common.BytesToAddress([]byte{0x4}),
		}

		valueDeclaration1 := stdlib.StandardLibraryFunction{
			Name: "foo",
			Type: &sema.FunctionType{
				ReturnTypeAnnotation: &sema.TypeAnnotation{
					Type: sema.VoidType,
				},
			},
			Available: func(location ast.Location) bool {
				addressLocation, ok := location.(ast.AddressLocation)
				return ok && addressLocation == location2
			},
		}

		valueDeclaration2 := stdlib.StandardLibraryFunction{
			Name: "foo",
			Type: &sema.FunctionType{
				Parameters: []*sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "n",
						TypeAnnotation: sema.NewTypeAnnotation(&sema.IntType{}),
					},
				},
				ReturnTypeAnnotation: &sema.TypeAnnotation{
					Type: sema.VoidType,
				},
			},
			Available: func(location ast.Location) bool {
				addressLocation, ok := location.(ast.AddressLocation)
				return ok && addressLocation == location4
			},
		}

		program2, err := parser2.ParseProgram(`let x = foo()`)
		require.NoError(t, err)

		program3, err := parser2.ParseProgram(`let y = foo()`)
		require.NoError(t, err)

		program4, err := parser2.ParseProgram(`let z = foo(1)`)
		require.NoError(t, err)

		_, err = ParseAndCheckWithOptions(t,
			`
              import 0x2
              import 0x3
              import 0x4

              fun main() {
                  foo()
              }
            `,
			ParseAndCheckOptions{
				Location: location1,
				Options: []sema.Option{
					sema.WithPredeclaredValues(
						[]sema.ValueDeclaration{
							valueDeclaration1,
							valueDeclaration2,
						},
					),
					sema.WithImportHandler(
						func(checker *sema.Checker, location ast.Location) (sema.Import, *sema.CheckerError) {
							checker, err := checker.EnsureLoaded(location, func() *ast.Program {
								switch location {
								case location2:
									return program2
								case location3:
									return program3
								case location4:
									return program4
								default:
									t.Fatal("invalid location", location)
									return nil
								}
							})
							if err != nil {
								return nil, err
							}
							return sema.CheckerImport{
								Checker: checker,
							}, nil
						},
					),
				},
			},
		)

		errs := ExpectCheckerErrors(t, err, 2)

		// The illegal use of 'foo' in 0x3 should be reported

		var importedProgramError *sema.ImportedProgramError
		utils.RequireErrorAs(t, errs[0], &importedProgramError)
		require.Equal(t, location3, importedProgramError.ImportLocation)
		importedErrs := ExpectCheckerErrors(t, importedProgramError.CheckerError, 1)
		require.IsType(t, &sema.NotDeclaredError{}, importedErrs[0])

		// The illegal use of 'foo' in 0x1 should be reported

		require.IsType(t, &sema.NotDeclaredError{}, errs[1])
	})
}