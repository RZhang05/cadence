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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestOverwriteError_Error(t *testing.T) {

	require.EqualError(t,
		&OverwriteError{
			Address: NewUnmeteredAddressValueFromBytes([]byte{0x1}),
			Path: PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		},
		"failed to save object: path /storage/test in account 0x0000000000000001 already stores an object",
	)
}

func TestErrorOutputIncludesLocationRage(t *testing.T) {
	t.Parallel()
	require.Equal(t,
		Error{
			Location: TestLocation,
			Err: &DereferenceError{
				Cause: "the value being referenced has been destroyed or moved",
				LocationRange: LocationRange{
					Location: TestLocation,
					HasPosition: ast.Range{
						StartPos: ast.Position{Offset: 0, Column: 0, Line: 0},
						EndPos:   ast.Position{Offset: 0, Column: 0, Line: 0},
					},
				},
			},
		}.Error(),
		"Execution failed:\nerror: dereference failed\n --> test:0:0\n",
	)
}
