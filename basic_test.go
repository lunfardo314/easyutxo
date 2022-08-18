package easyutxo

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

var data = []byte("a")

func TestDataArray(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		da := NewDataArray()
		da.Push(data)
		require.EqualValues(t, 1, da.Len())
		ser := MustBytes(da)
		daBack, err := ParseDataArray(ser)
		require.NoError(t, err)
		require.EqualValues(t, 1, daBack.Len())
		require.EqualValues(t, da.At(0), daBack.At(0))
		require.Panics(t, func() {
			da.At(1)
		})
		require.Panics(t, func() {
			daBack.At(100)
		})
	})
	t.Run("not empty", func(t *testing.T) {
		da := NewDataArray()
		require.Panics(t, func() {
			da.Push(nil)
		})
		require.Panics(t, func() {
			da.At(0)
		})
		err := da.Write(io.Discard)
		require.Error(t, err)
	})
	t.Run("too long", func(t *testing.T) {
		require.NotPanics(t, func() {
			da := NewDataArray()
			da.Push(bytes.Repeat(data, 256))
		})
		require.Panics(t, func() {
			da := NewDataArray()
			da.Push(bytes.Repeat(data, 257))
		})
		require.NotPanics(t, func() {
			da := NewDataArray()
			for i := 0; i < 256; i++ {
				da.Push(data)
			}
		})
		require.Panics(t, func() {
			da := NewDataArray()
			for i := 0; i < 257; i++ {
				da.Push(data)
			}
		})
	})
}
