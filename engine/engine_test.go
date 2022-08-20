package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	t.Run("opcode", func(t *testing.T) {
		t.Logf("%s", OpCode(0).String())
		t.Logf("%s", OP_EXIT)
		t.Logf("%s", OpCode(31))
		d := []byte{extendedOpcodeMask | 0x01}
		require.Panics(t, func() {
			parseOpcode(d)
		})
		d = []byte{0x01, 0x02}
		require.NotPanics(t, func() {
			parseOpcode(d)
		})
	})
	t.Run("1", func(t *testing.T) {
		e := NewEngine()
		e.Run(nil, nil)
	})
	t.Run("2", func(t *testing.T) {
		e := NewEngine()

		code := OP_EXIT.AsBytes()
		e.Run(code, nil)
		opcode, remaining := parseOpcode(code)
		require.EqualValues(t, OP_EXIT, opcode)
		require.EqualValues(t, 0, len(remaining))
	})
	t.Run("wrong opcode", func(t *testing.T) {
		e := NewEngine()

		opcode := OpCode(150)
		code := opcode.AsBytes()
		require.Panics(t, func() {
			e.Run(code, nil)
		})
	})

}
