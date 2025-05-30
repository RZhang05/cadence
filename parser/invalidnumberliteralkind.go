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

package parser

import (
	"github.com/onflow/cadence/errors"
)

//go:generate stringer -type=InvalidNumberLiteralKind

type InvalidNumberLiteralKind uint

const (
	InvalidNumberLiteralKindUnknown InvalidNumberLiteralKind = iota
	InvalidNumberLiteralKindLeadingUnderscore
	InvalidNumberLiteralKindTrailingUnderscore
	InvalidNumberLiteralKindUnknownPrefix
	InvalidNumberLiteralKindMissingDigits
)

func (k InvalidNumberLiteralKind) Description() string {
	switch k {
	case InvalidNumberLiteralKindLeadingUnderscore:
		return "leading underscore"
	case InvalidNumberLiteralKindTrailingUnderscore:
		return "trailing underscore"
	case InvalidNumberLiteralKindUnknownPrefix:
		return "unknown prefix"
	case InvalidNumberLiteralKindMissingDigits:
		return "missing digits"
	case InvalidNumberLiteralKindUnknown:
		return "unknown"
	}

	panic(errors.NewUnreachableError())
}
