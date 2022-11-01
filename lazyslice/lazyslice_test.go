package lazyslice

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

const howMany = 250

var data [][]byte

func init() {
	data = make([][]byte, howMany)
	for i := range data {
		data[i] = make([]byte, 2)
		binary.BigEndian.PutUint16(data[i], uint16(i))
	}
}

func TestLazySliceSemantics(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		ls := ArrayFromBytes(nil)
		require.EqualValues(t, 0, len(ls.Bytes()))
		require.Panics(t, func() {
			ls.NumElements()
		})
	})
	t.Run("empty", func(t *testing.T) {
		ls := EmptyArray()
		require.EqualValues(t, []byte{0, 0}, ls.Bytes())

		require.EqualValues(t, 0, ls.NumElements())
	})
	t.Run("serialize all nil", func(t *testing.T) {
		ls := EmptyArray()
		ls.Push(nil)
		ls.Push(nil)
		ls.Push(nil)
		require.EqualValues(t, 3, ls.NumElements())
		lsBin := ls.Bytes()
		require.EqualValues(t, []byte{byte(dataLenBytes0), 3}, lsBin)
		lsBack := ArrayFromBytes(lsBin)
		require.EqualValues(t, 3, ls.NumElements())
		lsBack.ForEach(func(i int, d []byte) bool {
			require.EqualValues(t, 0, len(d))
			return true
		})
	})
	t.Run("serialize some nil", func(t *testing.T) {
		ls := EmptyArray()
		ls.Push(nil)
		ls.Push(nil)
		ls.Push(data[17])
		ls.Push(nil)
		ls.Push([]byte("1234567890"))
		require.EqualValues(t, 5, ls.NumElements())
		lsBin := ls.Bytes()
		lsBack := ArrayFromBytes(lsBin)
		require.EqualValues(t, 5, lsBack.NumElements())
		require.EqualValues(t, 0, len(lsBack.At(0)))
		require.EqualValues(t, 0, len(lsBack.At(1)))
		require.EqualValues(t, data[17], lsBack.At(2))
		require.EqualValues(t, 0, len(lsBack.At(3)))
		require.EqualValues(t, []byte("1234567890"), lsBack.At(4))
	})
	t.Run("deserialize rubbish", func(t *testing.T) {
		ls := EmptyArray()
		ls.Push(data[17])
		lsBin := ls.Bytes()
		lsBack := ArrayFromBytes(lsBin)
		require.NotPanics(t, func() {
			require.EqualValues(t, data[17], ls.At(0))
		})
		lsBinWrong := append(lsBin, 1, 2, 3)
		lsBack = ArrayFromBytes(lsBinWrong)
		require.Panics(t, func() {
			lsBack.At(0)
		})
	})
	t.Run("push+boundaries", func(t *testing.T) {
		ls := ArrayFromBytes(nil)
		require.Panics(t, func() {
			ls.Push(data[17])
		})
		ls.SetEmptyArray()
		require.NotPanics(t, func() {
			ls.Push(data[17])
		})
		require.EqualValues(t, data[17], ls.At(0))
		require.EqualValues(t, 1, ls.NumElements())
		ser := ls.Bytes()
		lsBack := ArrayFromBytes(ser)
		require.EqualValues(t, 1, lsBack.NumElements())
		require.EqualValues(t, ls.At(0), lsBack.At(0))
		require.Panics(t, func() {
			ls.At(1)
		})
		require.Panics(t, func() {
			lsBack.At(100)
		})
	})
	t.Run("too long", func(t *testing.T) {
		require.NotPanics(t, func() {
			ls := EmptyArray()
			ls.Push(bytes.Repeat(data[0], 256))
		})
		require.NotPanics(t, func() {
			ls := EmptyArray()
			ls.Push(bytes.Repeat(data[0], 257))
		})
		require.NotPanics(t, func() {
			ls := EmptyArray()
			for i := 0; i < 255; i++ {
				ls.Push(data[0])
			}
		})
		require.Panics(t, func() {
			ls := EmptyArray(300)
			for i := 0; i < 301; i++ {
				ls.Push(data[0])
			}
		})
		require.Panics(t, func() {
			ls := EmptyArray()
			for i := 0; i < math.MaxUint16+1; i++ {
				ls.Push(data[0])
			}
		})
	})
	t.Run("serialize prefix", func(t *testing.T) {
		da := ArrayFromBytes([]byte{byte(dataLenBytes0), 0})
		bin := da.Bytes()
		daBack := ArrayFromBytes(bin)
		require.EqualValues(t, 0, daBack.NumElements())
		require.EqualValues(t, bin, daBack.Bytes())

		da = ArrayFromBytes(emptyArrayPrefix.Bytes())
		bin = da.Bytes()
		daBack = ArrayFromBytes(bin)
		require.EqualValues(t, 0, daBack.NumElements())
		require.EqualValues(t, bin, daBack.Bytes())

		da = ArrayFromBytes([]byte{byte(dataLenBytes0), 17})
		bin = da.Bytes()
		daBack = ArrayFromBytes(bin)
		require.EqualValues(t, 17, daBack.NumElements())
		for i := 0; i < 17; i++ {
			require.EqualValues(t, 0, len(daBack.At(i)))
		}
		require.Panics(t, func() {
			daBack.At(18)
		})
	})
	t.Run("serialize short", func(t *testing.T) {
		ls := EmptyArray()
		for i := 0; i < 100; i++ {
			ls.Push(bytes.Repeat(data[0], 100))
		}
		lsBack := ArrayFromBytes(ls.Bytes())
		require.EqualValues(t, ls.NumElements(), lsBack.NumElements())
		for i := 0; i < 100; i++ {
			require.EqualValues(t, ls.At(i), lsBack.At(i))
		}
	})
	t.Run("serialization long 1", func(t *testing.T) {
		ls := EmptyArray()
		for i := 0; i < 100; i++ {
			ls.Push(bytes.Repeat(data[0], 2000))
		}
		daBytes := ls.Bytes()
		daBack := ArrayFromBytes(daBytes)
		require.EqualValues(t, ls.NumElements(), daBack.NumElements())
		for i := 0; i < 100; i++ {
			require.EqualValues(t, ls.At(i), daBack.At(i))
		}
	})
	t.Run("serialization long 2", func(t *testing.T) {
		ls1 := EmptyArray()
		for i := 0; i < 100; i++ {
			ls1.Push(bytes.Repeat(data[0], 2000))
		}
		ls2 := EmptyArray()
		for i := 0; i < 100; i++ {
			ls2.Push(bytes.Repeat(data[0], 2000))
		}
		for i := 0; i < 100; i++ {
			require.EqualValues(t, ls1.At(i), ls2.At(i))
		}
		require.EqualValues(t, ls1.NumElements(), ls2.NumElements())
		require.EqualValues(t, ls1.Bytes(), ls2.Bytes())
	})
}

func TestSliceTreeSemantics(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		st := TreeFromBytes(nil)
		b := st.BytesAtPath(nil)
		require.EqualValues(t, 0, len(b))
	})
	t.Run("empty", func(t *testing.T) {
		st := TreeEmpty()
		b := st.BytesAtPath(nil)
		require.EqualValues(t, []byte{0, 0}, b)
	})
	t.Run("nil panic", func(t *testing.T) {
		st := TreeFromBytes(nil)
		require.Panics(t, func() {
			st.BytesAtPath(Path(1))
		})
	})
	t.Run("nonsense panic", func(t *testing.T) {
		st := TreeFromBytes([]byte("0123456789"))
		require.Panics(t, func() {
			st.BytesAtPath(Path(0))
		})
	})
	t.Run("empty panic", func(t *testing.T) {
		st := TreeEmpty()
		require.Panics(t, func() {
			st.BytesAtPath(Path(0))
		})
	})
	t.Run("level 1-1", func(t *testing.T) {
		sa := EmptyArray()
		for i := 0; i < howMany; i++ {
			sa.Push(data[i])
		}
		st := TreeFromBytes(sa.Bytes())
		t.Logf("ser len = %d bytes (%d x uint16)", len(sa.Bytes()), howMany)
		for i := 0; i < howMany; i++ {
			var tmp []byte
			tmp = st.BytesAtPath(Path(byte(i)))
			require.EqualValues(t, uint16(i), binary.BigEndian.Uint16(tmp))
		}
		require.Panics(t, func() {
			st.BytesAtPath(Path(howMany))
		})
	})
}
