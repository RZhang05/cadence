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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckNilCoalescingNilIntToOptional(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let one = 1
      let none: Int? = nil
      let x: Int? = none ?? one
    `)

	require.NoError(t, err)
}

func TestCheckNilCoalescingNilIntToOptionals(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let one = 1
      let none: Int?? = nil
      let x: Int? = none ?? one
    `)

	require.NoError(t, err)
}

func TestCheckNilCoalescingNilIntToOptionalNilLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let one = 1
      let x: Int? = nil ?? one
    `)

	require.NoError(t, err)
}

func TestCheckInvalidNilCoalescingMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: Int? = nil ?? false
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckNilCoalescingRightSubtype(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: Int? = nil ?? nil
    `)

	require.NoError(t, err)
}

func TestCheckNilCoalescingNilInt(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let one = 1
      let none: Int? = nil
      let x: Int = none ?? one
    `)

	require.NoError(t, err)
}

func TestCheckInvalidNilCoalescingOptionalsInt(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let one = 1
      let none: Int?? = nil
      let x: Int = none ?? one
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckNilCoalescingNilLiteralInt(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     let one = 1
     let x: Int = nil ?? one
   `)

	require.NoError(t, err)
}

func TestCheckInvalidNilCoalescingMismatchNonOptional(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     let x: Int = nil ?? false
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidNilCoalescingRightSubtype(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     let x: Int = nil ?? nil
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidNilCoalescingNonMatchingTypes(t *testing.T) {

	t.Parallel()

	t.Run("with super type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x: Int? = 1
          let y = x ?? false
       `)

		require.NoError(t, err)
	})

	t.Run("no super type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x: Int? = 1
          let y: Int = x ?? false
       `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckNilCoalescingAny(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     let x: AnyStruct? = 1
     let y = x ?? false
  `)

	require.NoError(t, err)
}

func TestCheckNilCoalescingOptionalRightHandSide(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
     let x: Int? = 1
     let y: Int? = 2
     let z = x ?? y
  `)

	require.NoError(t, err)

	assert.IsType(t, &sema.OptionalType{Type: sema.IntType}, RequireGlobalValue(t, checker.Elaboration, "z"))
}

func TestCheckNilCoalescingBothOptional(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
     let x: Int?? = 1
     let y: Int? = 2
     let z = x ?? y
  `)

	require.NoError(t, err)

	assert.IsType(t, &sema.OptionalType{Type: sema.IntType}, RequireGlobalValue(t, checker.Elaboration, "z"))
}

func TestCheckNilCoalescingWithNever(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithPanic(t,
		`
          access(all) let x: Int? = nil
          access(all) let y = x ?? panic("nope")
        `,
	)

	require.NoError(t, err)
}
