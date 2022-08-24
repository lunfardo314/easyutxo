package lazyslice

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"math"
	"testing"

	"github.com/lunfardo314/easyutxo"
	"github.com/stretchr/testify/require"
)

const howMany = 250

var data [][]byte

func init() {
	data = make([][]byte, howMany)
	for i := range data {
		data[i] = easyutxo.EncodeInteger(uint16(i))
	}
}

func TestLazySliceSemantics(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		ls := LazySliceFromBytes(nil)
		require.EqualValues(t, 0, len(ls.Bytes()))
		require.Panics(t, func() {
			ls.NumElements()
		})
	})
	t.Run("empty", func(t *testing.T) {
		ls := LazySliceEmptyArray()
		require.EqualValues(t, []byte{0, 0}, ls.Bytes())

		require.EqualValues(t, 0, ls.NumElements())
	})
	t.Run("serialize all nil", func(t *testing.T) {
		ls := LazySliceEmptyArray()
		ls.Push(nil)
		ls.Push(nil)
		ls.Push(nil)
		require.EqualValues(t, 3, ls.NumElements())
		lsBin := ls.Bytes()
		require.EqualValues(t, []byte{byte(DataLenBytes0), 3}, lsBin)
		lsBack := LazySliceFromBytes(lsBin)
		require.EqualValues(t, 3, ls.NumElements())
		lsBack.ForEach(func(i int, d []byte) bool {
			require.EqualValues(t, 0, len(d))
			return true
		})
	})
	t.Run("serialize some nil", func(t *testing.T) {
		ls := LazySliceEmptyArray()
		ls.Push(nil)
		ls.Push(nil)
		ls.Push(data[17])
		ls.Push(nil)
		ls.Push([]byte("1234567890"))
		require.EqualValues(t, 5, ls.NumElements())
		lsBin := ls.Bytes()
		lsBack := LazySliceFromBytes(lsBin)
		require.EqualValues(t, 5, lsBack.NumElements())
		require.EqualValues(t, 0, len(lsBack.At(0)))
		require.EqualValues(t, 0, len(lsBack.At(1)))
		require.EqualValues(t, data[17], lsBack.At(2))
		require.EqualValues(t, 0, len(lsBack.At(3)))
		require.EqualValues(t, []byte("1234567890"), lsBack.At(4))
	})
	t.Run("push+boundaries", func(t *testing.T) {
		ls := LazySliceFromBytes(nil)
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
		lsBack := LazySliceFromBytes(ser)
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
			ls := LazySliceEmptyArray()
			ls.Push(bytes.Repeat(data[0], 256))
		})
		require.NotPanics(t, func() {
			ls := LazySliceEmptyArray()
			ls.Push(bytes.Repeat(data[0], 257))
		})
		require.NotPanics(t, func() {
			ls := LazySliceEmptyArray()
			for i := 0; i < 255; i++ {
				ls.Push(data[0])
			}
		})
		require.Panics(t, func() {
			ls := LazySliceEmptyArray().WithMaxNumberOfElements(300)
			for i := 0; i < 301; i++ {
				ls.Push(data[0])
			}
		})
		require.Panics(t, func() {
			ls := LazySliceEmptyArray()
			for i := 0; i < math.MaxUint16+1; i++ {
				ls.Push(data[0])
			}
		})
	})
	t.Run("serialize prefix", func(t *testing.T) {
		da := LazySliceFromBytes([]byte{byte(DataLenBytes0), 0})
		bin := da.Bytes()
		daBack := LazySliceFromBytes(bin)
		require.EqualValues(t, 0, daBack.NumElements())
		require.EqualValues(t, bin, daBack.Bytes())

		da = LazySliceFromBytes(emptyArrayPrefix.Bytes())
		bin = da.Bytes()
		daBack = LazySliceFromBytes(bin)
		require.EqualValues(t, 0, daBack.NumElements())
		require.EqualValues(t, bin, daBack.Bytes())

		da = LazySliceFromBytes([]byte{byte(DataLenBytes0), 17})
		bin = da.Bytes()
		daBack = LazySliceFromBytes(bin)
		require.EqualValues(t, 17, daBack.NumElements())
		for i := 0; i < 17; i++ {
			require.EqualValues(t, 0, len(daBack.At(i)))
		}
		require.Panics(t, func() {
			daBack.At(18)
		})
	})
	t.Run("serialize short", func(t *testing.T) {
		ls := LazySliceEmptyArray()
		for i := 0; i < 100; i++ {
			ls.Push(bytes.Repeat(data[0], 100))
		}
		lsBack := LazySliceFromBytes(ls.Bytes())
		require.EqualValues(t, ls.NumElements(), lsBack.NumElements())
		for i := 0; i < 100; i++ {
			require.EqualValues(t, ls.At(i), lsBack.At(i))
		}
	})
	t.Run("serialization long 1", func(t *testing.T) {
		ls := LazySliceEmptyArray()
		for i := 0; i < 100; i++ {
			ls.Push(bytes.Repeat(data[0], 2000))
		}
		daBytes := ls.Bytes()
		daBack := LazySliceFromBytes(daBytes)
		require.EqualValues(t, ls.NumElements(), daBack.NumElements())
		for i := 0; i < 100; i++ {
			require.EqualValues(t, ls.At(i), daBack.At(i))
		}
	})
	t.Run("serialization long 2", func(t *testing.T) {
		ls1 := LazySliceEmptyArray()
		for i := 0; i < 100; i++ {
			ls1.Push(bytes.Repeat(data[0], 2000))
		}
		ls2 := LazySliceEmptyArray()
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
		st := LazySliceTreeFromBytes(nil)
		b := st.BytesAtPath()
		require.EqualValues(t, 0, len(b))
	})
	t.Run("empty", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		b := st.BytesAtPath()
		require.EqualValues(t, []byte{0, 0}, b)
	})
	t.Run("nil panic", func(t *testing.T) {
		st := LazySliceTreeFromBytes(nil)
		require.Panics(t, func() {
			st.BytesAtPath(1)
		})
	})
	t.Run("nonsense panic", func(t *testing.T) {
		st := LazySliceTreeFromBytes([]byte("0123456789"))
		require.Panics(t, func() {
			st.BytesAtPath(0)
		})
	})
	t.Run("empty panic", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		require.Panics(t, func() {
			st.BytesAtPath(0)
		})
	})
	t.Run("level 1-1", func(t *testing.T) {
		sa := LazySliceEmptyArray()
		for i := 0; i < howMany; i++ {
			sa.Push(data[i])
		}
		st := LazySliceTreeFromBytes(sa.Bytes())
		t.Logf("ser len = %d bytes (%d x uint16)", len(sa.Bytes()), howMany)
		for i := 0; i < howMany; i++ {
			var tmp []byte
			tmp = st.BytesAtPath(byte(i))
			require.EqualValues(t, uint16(i), easyutxo.DecodeInteger[uint16](tmp))
		}
		require.Panics(t, func() {
			st.BytesAtPath(howMany)
		})
	})
	t.Run("level 1-2", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		for i := 0; i < howMany; i++ {
			st.PushDataAtPath(data[i])
		}
		for i := 0; i < howMany; i++ {
			var tmp []byte
			tmp = st.BytesAtPath(byte(i))
			require.EqualValues(t, uint16(i), binary.BigEndian.Uint16(tmp))
		}
		require.Panics(t, func() {
			st.BytesAtPath(howMany)
		})
	})
	t.Run("level 2 panic", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		require.Panics(t, func() {
			st.PushDataAtPath(data[0], 1)
		})
	})
	t.Run("level 2 panic and not", func(t *testing.T) {
		st := LazySliceTreeEmpty()

		st.PushNewSubtreeAtPath()
		require.NotPanics(t, func() {
			st.PushDataAtPath(data[0], 0)
		})

		require.Panics(t, func() {
			st.PushDataAtPath(data[0], 1)
		})

		st.PushNewSubtreeAtPath()
		require.NotPanics(t, func() {
			st.PushDataAtPath(data[0], 1)
		})
	})
	t.Run("level 3", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		st.PushNewSubtreeAtPath()
		st.PushNewSubtreeAtPath()
		st.PushNewSubtreeAtPath(0)
		st.PushNewSubtreeAtPath(0)
		st.PushNewSubtreeAtPath(1)
		st.PushNewSubtreeAtPath(1)
		st.PushNewSubtreeAtPath(1)
		require.EqualValues(t, 2, st.NumElementsAtPath())
		require.EqualValues(t, 2, st.NumElementsAtPath(0))
		require.EqualValues(t, 3, st.NumElementsAtPath(1))
		require.EqualValues(t, 0, st.NumElementsAtPath(0, 0))
		require.EqualValues(t, 0, st.NumElementsAtPath(0, 1))
		require.EqualValues(t, 0, st.NumElementsAtPath(1, 0))
		require.EqualValues(t, 0, st.NumElementsAtPath(1, 1))
		require.EqualValues(t, 0, st.NumElementsAtPath(1, 2))
		require.Panics(t, func() {
			st.NumElementsAtPath(0, 2)
		})
		require.Panics(t, func() {
			st.NumElementsAtPath(1, 3)
		})

		st.PushDataAtPath(data[3], 1, 2)
		require.EqualValues(t, 1, st.NumElementsAtPath(1, 2))
		dataBack := st.BytesAtPath(1, 2, 0)
		require.EqualValues(t, data[3], dataBack)
		require.Panics(t, func() {
			st.BytesAtPath(1, 2, 1)
		})

		bs := []byte("1234567890")
		st.SetDataAtPathAtIdx(0, bs, 1, 2)
		require.EqualValues(t, 1, st.NumElementsAtPath(1, 2))
		dataBack = st.BytesAtPath(1, 2, 0)
		require.EqualValues(t, bs, dataBack)
		require.Panics(t, func() {
			st.BytesAtPath(1, 2, 1)
		})
		require.Panics(t, func() {
			tmp := st.BytesAtPath(1, 2, 0, 0)
			require.EqualValues(t, 0, len(tmp))
		})
		require.Panics(t, func() {
			st.BytesAtPath(1, 2, 0, 18)
		})
	})
	t.Run("serialize", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		s := st.Bytes()
		st1 := LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len root: %d", len(s))

		st.PushNewSubtreeAtPath()
		s = st.Bytes()
		st1 = LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 1 node: %d", len(s))

		st.PushNewSubtreeAtPath()
		s = st.Bytes()
		st1 = LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 2 nodes: %d", len(s))

		st.PushNewSubtreeAtPath(0)
		s = st.Bytes()
		st1 = LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 3 nodes: %d", len(s))

		st.PushNewSubtreeAtPath(0)
		s = st.Bytes()
		st1 = LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 4 nodes: %d", len(s))

		st.PushNewSubtreeAtPath(1)
		s = st.Bytes()
		st1 = LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 5 nodes: %d", len(s))

		st.PushNewSubtreeAtPath(1)
		s = st.Bytes()
		st1 = LazySliceFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 6 nodes: %d", len(s))

		var d [100]byte
		_, _ = rand.Read(d[:])

		st.PushDataAtPath(d[:], 1, 1)
		s = st.Bytes()
		s1 := make([]byte, len(s))
		copy(s1, s)
		st1 = LazySliceFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100 bytes data: %d", len(s))

		st.PushDataAtPath(d[:], 1, 1)
		s = st.Bytes()
		s1 = make([]byte, len(s))
		copy(s1, s)
		st1 = LazySliceFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100+100 bytes data: %d", len(s))

		var dd [1000]byte
		_, _ = rand.Read(dd[:])

		st.PushDataAtPath(dd[:], 1, 1)
		s = st.Bytes()
		s1 = make([]byte, len(s))
		copy(s1, s)
		st1 = LazySliceFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100+100+1000 bytes data: %d", len(s))

		st.SetDataAtPathAtIdx(st.NumElementsAtPath(1, 1)-1, dd[:500], 1, 1)
		s = st.Bytes()
		s1 = make([]byte, len(s))
		copy(s1, s)
		st1 = LazySliceFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100+100+(1000-500) bytes data: %d", len(s))
	})
}

func TestTwoLayer(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		st := LazySliceTreeEmpty()
		for _, d := range data {
			st.PushLayerTwo(d)
		}
		for _, d := range data {
			st.PushLayerTwo(d)
		}
		idx := uint16(0)
		for _, d := range data {
			require.EqualValues(t, d, st.AtIdxLayerTwo(idx))
			idx++
		}
		for _, d := range data {
			require.EqualValues(t, d, st.AtIdxLayerTwo(idx))
			idx++
		}
	})
}
