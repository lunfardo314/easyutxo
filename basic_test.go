package easyutxo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var data = []byte("a")

func TestParams(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		da := NewParams(nil)
		da.Push(data)
		require.EqualValues(t, 1, da.NumElements())
		ser := da.Bytes()
		daBack := NewParams(ser)
		require.EqualValues(t, 1, daBack.NumElements())
		require.EqualValues(t, da.At(0), daBack.At(0))
		require.Panics(t, func() {
			da.At(1)
		})
		require.Panics(t, func() {
			daBack.At(100)
		})
	})
	t.Run("2", func(t *testing.T) {
		da := NewParams(nil)
		da.Push(data)
		require.EqualValues(t, 1, da.NumElements())
		serFull := da.Bytes()
		daBack := NewParams(serFull)
		require.EqualValues(t, 1, daBack.NumElements())
		require.EqualValues(t, da.At(0), daBack.At(0))
		require.Panics(t, func() {
			da.At(1)
		})
		require.Panics(t, func() {
			daBack.At(100)
		})
	})
	t.Run("not empty", func(t *testing.T) {
		da := NewParams(nil)
		daBytes := da.Bytes()
		require.EqualValues(t, []byte{0, 0}, daBytes)
		require.NotPanics(t, func() {
			da.Push(nil)
		})
		require.Panics(t, func() {
			da.At(1)
		})
	})
	t.Run("too long", func(t *testing.T) {
		require.NotPanics(t, func() {
			da := NewParams(nil)
			da.Push(bytes.Repeat(data, 256))
		})
		require.NotPanics(t, func() {
			da := NewParams(nil)
			da.Push(bytes.Repeat(data, 257))
		})
		require.NotPanics(t, func() {
			da := NewParams(nil)
			for i := 0; i < 256; i++ {
				da.Push(data)
			}
		})
		require.Panics(t, func() {
			da := NewParams(nil)
			for i := 0; i < 257; i++ {
				da.Push(data)
			}
		})
	})
	t.Run("serialization short", func(t *testing.T) {
		da := NewParams(nil)
		for i := 0; i < 100; i++ {
			da.Push(bytes.Repeat(data, 100))
		}
		daBack := NewParams(da.Bytes())
		require.EqualValues(t, da.NumElements(), daBack.NumElements())
		for i := byte(0); i < 100; i++ {
			require.EqualValues(t, da.At(i), daBack.At(i))
		}
	})
	t.Run("serialization long", func(t *testing.T) {
		da := NewParams(nil)
		for i := 0; i < 100; i++ {
			da.Push(bytes.Repeat(data, 2000))
		}
		daBytes := da.Bytes()
		daBack := NewParams(daBytes)
		require.EqualValues(t, da.NumElements(), daBack.NumElements())
		for i := byte(0); i < 100; i++ {
			require.EqualValues(t, da.At(i), daBack.At(i))
		}
	})
}
