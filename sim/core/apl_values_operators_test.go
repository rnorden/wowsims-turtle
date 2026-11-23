package core

import (
	"math"
	"testing"
	"time"

	"github.com/isfir/wowsims-turtle/sim/core/proto"
)

// Non-constant test values that, like most real APL values (e.g. autoSwingTime),
// only implement the getter matching their declared type.
type testAPLDuration struct {
	DefaultAPLValueImpl
	val time.Duration
}

func (v *testAPLDuration) Type() proto.APLValueType                { return proto.APLValueType_ValueTypeDuration }
func (v *testAPLDuration) GetDuration(_ *Simulation) time.Duration { return v.val }
func (v *testAPLDuration) String() string                          { return v.val.String() }

type testAPLInt struct {
	DefaultAPLValueImpl
	val int32
}

func (v *testAPLInt) Type() proto.APLValueType   { return proto.APLValueType_ValueTypeInt }
func (v *testAPLInt) GetInt(_ *Simulation) int32 { return v.val }
func (v *testAPLInt) String() string             { return "testInt" }

type testAPLFloat struct {
	DefaultAPLValueImpl
	val float64
}

func (v *testAPLFloat) Type() proto.APLValueType       { return proto.APLValueType_ValueTypeFloat }
func (v *testAPLFloat) GetFloat(_ *Simulation) float64 { return v.val }
func (v *testAPLFloat) String() string                 { return "testFloat" }

func TestValueMathTyping(t *testing.T) {
	sim := &Simulation{}
	rot := &APLRotation{unit: &Unit{}}
	c := func(s string) APLValue { return rot.newValueConst(&proto.APLValueConst{Val: s}) }
	swing := func() APLValue { return &testAPLDuration{val: 2 * time.Second} }

	const (
		mul = proto.APLValueMath_OpMul
		div = proto.APLValueMath_OpDiv
		add = proto.APLValueMath_OpAdd
		sub = proto.APLValueMath_OpSub
	)
	durT := proto.APLValueType_ValueTypeDuration
	floatT := proto.APLValueType_ValueTypeFloat
	intT := proto.APLValueType_ValueTypeInt

	durCases := []struct {
		name     string
		op       proto.APLValueMath_MathOperator
		lhs, rhs APLValue
		want     time.Duration
	}{
		{"duration * float const", mul, swing(), c("0.1"), 200 * time.Millisecond},
		{"duration * small float const", mul, swing(), c("0.05"), 100 * time.Millisecond},
		{"duration * percent const", mul, swing(), c("25%"), 500 * time.Millisecond},
		{"float const * duration", mul, c("0.1"), swing(), 200 * time.Millisecond},
		{"duration * float value", mul, swing(), &testAPLFloat{val: 0.75}, 1500 * time.Millisecond},
		{"float value * duration", mul, &testAPLFloat{val: 0.75}, swing(), 1500 * time.Millisecond},
		{"duration * int const", mul, swing(), c("3"), 6 * time.Second},
		{"int const * duration", mul, c("3"), swing(), 6 * time.Second},
		{"duration * int value", mul, swing(), &testAPLInt{val: 3}, 6 * time.Second},
		{"int value * duration", mul, &testAPLInt{val: 3}, swing(), 6 * time.Second},
		{"duration const * float const", mul, c("1.5s"), c("0.5"), 750 * time.Millisecond},
		{"duration / float const", div, swing(), c("0.5"), 4 * time.Second},
		{"duration / float const (shrink)", div, swing(), c("4.0"), 500 * time.Millisecond},
		{"duration / int const", div, swing(), c("4"), 500 * time.Millisecond},
		{"duration / int value", div, swing(), &testAPLInt{val: 4}, 500 * time.Millisecond},
		{"duration + duration", add, swing(), c("1.5s"), 3500 * time.Millisecond},
		{"duration - duration", sub, swing(), c("1.5s"), 500 * time.Millisecond},
		{"duration + int const (seconds)", add, swing(), c("1"), 3 * time.Second},
	}
	for _, tc := range durCases {
		v := rot.newValueMathFromValues(tc.op, tc.lhs, tc.rhs)
		if v.Type() != durT {
			t.Errorf("%s: expected duration type, got %s", tc.name, v.Type())
			continue
		}
		if got := v.GetDuration(sim); got != tc.want {
			t.Errorf("%s: expected %s, got %s", tc.name, tc.want, got)
		}
	}

	floatCases := []struct {
		name     string
		op       proto.APLValueMath_MathOperator
		lhs, rhs APLValue
		want     float64
	}{
		{"duration / duration", div, swing(), c("0.5s"), 4},
		{"duration / duration (fraction)", div, c("3s"), c("2s"), 1.5},
		{"number / duration", div, c("1"), swing(), 0.5},
		{"float * float", mul, c("0.5"), c("0.25"), 0.125},
		{"int * float const", mul, c("2"), c("0.25"), 0.5},
		{"float value * int value", mul, &testAPLFloat{val: 1.5}, &testAPLInt{val: 3}, 4.5},
		{"int value / float value", div, &testAPLInt{val: 3}, &testAPLFloat{val: 2}, 1.5},
	}
	for _, tc := range floatCases {
		v := rot.newValueMathFromValues(tc.op, tc.lhs, tc.rhs)
		if v.Type() != floatT {
			t.Errorf("%s: expected float type, got %s", tc.name, v.Type())
			continue
		}
		if got := v.GetFloat(sim); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s: expected %f, got %f", tc.name, tc.want, got)
		}
	}

	intCases := []struct {
		name     string
		op       proto.APLValueMath_MathOperator
		lhs, rhs APLValue
		want     int32
	}{
		{"int * int", mul, c("3"), &testAPLInt{val: 4}, 12},
		{"int / int (integer division)", div, c("7"), c("2"), 3},
		{"int + int", add, c("7"), c("2"), 9},
	}
	for _, tc := range intCases {
		v := rot.newValueMathFromValues(tc.op, tc.lhs, tc.rhs)
		if v.Type() != intT {
			t.Errorf("%s: expected int type, got %s", tc.name, v.Type())
			continue
		}
		if got := v.GetInt(sim); got != tc.want {
			t.Errorf("%s: expected %d, got %d", tc.name, tc.want, got)
		}
	}
}

// A scaled duration must compare correctly against duration and number constants.
func TestValueMathInCompare(t *testing.T) {
	sim := &Simulation{}
	rot := &APLRotation{unit: &Unit{}}
	c := func(s string) APLValue { return rot.newValueConst(&proto.APLValueConst{Val: s}) }

	scaled := rot.newValueMathFromValues(proto.APLValueMath_OpMul, &testAPLDuration{val: 2 * time.Second}, c("0.1"))

	lhs, rhs := rot.coerceToSameType(scaled, c("150ms"))
	cmp := &APLValueCompare{op: proto.APLValueCompare_OpGt, lhs: lhs, rhs: rhs}
	if !cmp.GetBool(sim) {
		t.Errorf("expected 2s * 0.1 > 150ms")
	}

	// Bare numbers compared against durations are interpreted as seconds.
	lhs, rhs = rot.coerceToSameType(scaled, c("0.25"))
	cmp = &APLValueCompare{op: proto.APLValueCompare_OpLt, lhs: lhs, rhs: rhs}
	if !cmp.GetBool(sim) {
		t.Errorf("expected 2s * 0.1 < 0.25")
	}
}
