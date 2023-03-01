// Code generated by "stringer -type=Opcode"; DO NOT EDIT.

package opcode

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Unknown-0]
	_ = x[Return-1]
	_ = x[ReturnValue-2]
	_ = x[Jump-3]
	_ = x[JumpIfFalse-4]
	_ = x[IntAdd-5]
	_ = x[IntSubtract-6]
	_ = x[IntMultiply-7]
	_ = x[IntDivide-8]
	_ = x[IntMod-9]
	_ = x[IntEqual-10]
	_ = x[IntNotEqual-11]
	_ = x[IntLess-12]
	_ = x[IntGreater-13]
	_ = x[IntLessOrEqual-14]
	_ = x[IntGreaterOrEqual-15]
	_ = x[GetConstant-16]
	_ = x[True-17]
	_ = x[False-18]
	_ = x[GetLocal-19]
	_ = x[SetLocal-20]
	_ = x[GetGlobal-21]
	_ = x[GetField-22]
	_ = x[SetField-23]
	_ = x[Call-24]
	_ = x[New-25]
	_ = x[Pop-26]
	_ = x[CheckType-27]
}

const _Opcode_name = "UnknownReturnReturnValueJumpJumpIfFalseIntAddIntSubtractIntMultiplyIntDivideIntModIntEqualIntNotEqualIntLessIntGreaterIntLessOrEqualIntGreaterOrEqualGetConstantTrueFalseGetLocalSetLocalGetGlobalGetFieldSetFieldCallNewPopCheckType"

var _Opcode_index = [...]uint8{0, 7, 13, 24, 28, 39, 45, 56, 67, 76, 82, 90, 101, 108, 118, 132, 149, 160, 164, 169, 177, 185, 194, 202, 210, 214, 217, 220, 229}

func (i Opcode) String() string {
	if i >= Opcode(len(_Opcode_index)-1) {
		return "Opcode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Opcode_name[_Opcode_index[i]:_Opcode_index[i+1]]
}