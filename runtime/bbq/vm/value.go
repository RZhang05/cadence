/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
)

type Value interface {
	isValue()
	StaticType(common.MemoryGauge) StaticType
	Transfer(
		config *Config,
		address atree.Address,
		remove bool,
		storable atree.Storable,
	) Value
	String() string
}

type MemberAccessibleValue interface {
	GetMember(config *Config, name string) Value
	SetMember(conf *Config, name string, value Value)
}
