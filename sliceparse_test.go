package easyutxo

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

var data = []byte("a")

func TestSliceArray(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		da := SliceArrayFromBytes(nil)
		da.Push(data)
		require.EqualValues(t, 1, da.NumElements())
		ser := da.Bytes()
		daBack := SliceArrayFromBytes(ser)
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
		da := SliceArrayFromBytes(nil)
		da.Push(data)
		require.EqualValues(t, 1, da.NumElements())
		serFull := da.Bytes()
		daBack := SliceArrayFromBytes(serFull)
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
		da := SliceArrayFromBytes(nil)
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
			da := SliceArrayFromBytes(nil)
			da.Push(bytes.Repeat(data, 256))
		})
		require.NotPanics(t, func() {
			da := SliceArrayFromBytes(nil)
			da.Push(bytes.Repeat(data, 257))
		})
		require.NotPanics(t, func() {
			da := SliceArrayFromBytes(nil)
			for i := 0; i < 255; i++ {
				da.Push(data)
			}
		})
		require.Panics(t, func() {
			da := SliceArrayFromBytes(nil)
			for i := 0; i < 256; i++ {
				da.Push(data)
			}
		})
		require.Panics(t, func() {
			da := SliceArrayFromBytes(nil)
			for i := 0; i < math.MaxUint16+1; i++ {
				da.Push(data)
			}
		})
	})
	t.Run("serialization short", func(t *testing.T) {
		da := SliceArrayFromBytes(nil)
		for i := 0; i < 100; i++ {
			da.Push(bytes.Repeat(data, 100))
		}
		daBack := SliceArrayFromBytes(da.Bytes())
		require.EqualValues(t, da.NumElements(), daBack.NumElements())
		for i := byte(0); i < 100; i++ {
			require.EqualValues(t, da.At(i), daBack.At(i))
		}
	})
	t.Run("serialization long 1", func(t *testing.T) {
		da := SliceArrayFromBytes(nil)
		for i := 0; i < 100; i++ {
			da.Push(bytes.Repeat(data, 2000))
		}
		daBytes := da.Bytes()
		daBack := SliceArrayFromBytes(daBytes)
		require.EqualValues(t, da.NumElements(), daBack.NumElements())
		for i := byte(0); i < 100; i++ {
			require.EqualValues(t, da.At(i), daBack.At(i))
		}
	})
	t.Run("serialization long 2", func(t *testing.T) {
		da1 := SliceArrayFromBytes(nil)
		for i := 0; i < 100; i++ {
			da1.Push(bytes.Repeat(data, 2000))
		}
		da2 := SliceArrayFromBytes(nil)
		for i := 0; i < 100; i++ {
			da2.Push(bytes.Repeat(data, 2000))
		}
		for i := byte(0); i < 100; i++ {
			require.EqualValues(t, da1.At(i), da2.At(i))
		}
		require.EqualValues(t, da1.NumElements(), da2.NumElements())
		require.EqualValues(t, da1.Bytes(), da2.Bytes())
	})
}

const howMany = 250

var data1 [][]byte

func init() {
	data1 = make([][]byte, howMany)

	for i := range data1 {
		data1[i] = make([]byte, 2)
		binary.LittleEndian.PutUint16(data1[i], uint16(i))
	}
}

func TestSliceTree(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)
		b := st.BytesAtPath()
		t.Logf("retrieve(nil) = %v", b)
	})
	t.Run("empty panic", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)
		require.Panics(t, func() {
			st.BytesAtPath(1)
		})
	})
	t.Run("level 1-1", func(t *testing.T) {
		sa := SliceArrayFromBytes(nil)
		for i := 0; i < howMany; i++ {
			sa.Push(data1[i])
		}
		st := SliceTreeFromBytes(sa.Bytes())
		t.Logf("ser len = %d bytes (%d x uint16)", len(sa.Bytes()), howMany)
		for i := 0; i < howMany; i++ {
			var tmp []byte
			tmp = st.BytesAtPath(byte(i))
			require.EqualValues(t, uint16(i), binary.LittleEndian.Uint16(tmp))
		}
		require.Panics(t, func() {
			st.BytesAtPath(howMany)
		})
	})
	t.Run("level 1-2", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)
		for i := 0; i < howMany; i++ {
			st.PushDataAtPath(data1[i])
		}
		for i := 0; i < howMany; i++ {
			var tmp []byte
			tmp = st.BytesAtPath(byte(i))
			require.EqualValues(t, uint16(i), binary.LittleEndian.Uint16(tmp))
		}
		require.Panics(t, func() {
			st.BytesAtPath(howMany)
		})
	})
	t.Run("level 2 panic", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)
		require.Panics(t, func() {
			st.PushDataAtPath(data1[0], 1)
		})
	})
	t.Run("level 2 panic and not", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)

		st.PushNewArrayAtPath()
		require.NotPanics(t, func() {
			st.PushDataAtPath(data1[0], 0)
		})

		require.Panics(t, func() {
			st.PushDataAtPath(data1[0], 1)
		})

		st.PushNewArrayAtPath()
		require.NotPanics(t, func() {
			st.PushDataAtPath(data1[0], 1)
		})
	})
	t.Run("level 3", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)
		st.PushNewArrayAtPath()
		st.PushNewArrayAtPath()
		st.PushNewArrayAtPath(0)
		st.PushNewArrayAtPath(0)
		st.PushNewArrayAtPath(1)
		st.PushNewArrayAtPath(1)
		st.PushNewArrayAtPath(1)
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

		st.PushDataAtPath(data1[3], 1, 2)
		require.EqualValues(t, 1, st.NumElementsAtPath(1, 2))
		dataBack := st.BytesAtPath(1, 2, 0)
		require.EqualValues(t, data1[3], dataBack)
		require.Panics(t, func() {
			st.BytesAtPath(1, 2, 1)
		})

		st.SetDataAtPathAt(data1[17], 0, 1, 2)
		require.EqualValues(t, 1, st.NumElementsAtPath(1, 2))
		dataBack = st.BytesAtPath(1, 2, 0)
		require.EqualValues(t, data1[17], dataBack)
		require.Panics(t, func() {
			st.BytesAtPath(1, 2, 1)
		})
		require.Panics(t, func() {
			st.BytesAtPath(1, 2, 0, 0)
		})
	})
	t.Run("serialize", func(t *testing.T) {
		st := SliceTreeFromBytes(nil)
		s := st.Bytes()
		st1 := SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len root: %d", len(s))

		st.PushNewArrayAtPath()
		s = st.Bytes()
		st1 = SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 1 node: %d", len(s))

		st.PushNewArrayAtPath()
		s = st.Bytes()
		st1 = SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 2 nodes: %d", len(s))

		st.PushNewArrayAtPath(0)
		s = st.Bytes()
		st1 = SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 3 nodes: %d", len(s))

		st.PushNewArrayAtPath(0)
		s = st.Bytes()
		st1 = SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 4 nodes: %d", len(s))

		st.PushNewArrayAtPath(1)
		s = st.Bytes()
		st1 = SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 5 nodes: %d", len(s))

		st.PushNewArrayAtPath(1)
		s = st.Bytes()
		st1 = SliceArrayFromBytes(s)
		require.EqualValues(t, s, st1.Bytes())
		t.Logf("len 6 nodes: %d", len(s))

		var d [100]byte
		_, _ = rand.Read(d[:])

		st.PushDataAtPath(d[:], 1, 1)
		s = st.Bytes()
		s1 := make([]byte, len(s))
		copy(s1, s)
		st1 = SliceArrayFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100 bytes data: %d", len(s))

		st.PushDataAtPath(d[:], 1, 1)
		s = st.Bytes()
		s1 = make([]byte, len(s))
		copy(s1, s)
		st1 = SliceArrayFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100+100 bytes data: %d", len(s))

		var dd [1000]byte
		_, _ = rand.Read(dd[:])

		st.PushDataAtPath(dd[:], 1, 1)
		s = st.Bytes()
		s1 = make([]byte, len(s))
		copy(s1, s)
		st1 = SliceArrayFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100+100+1000 bytes data: %d", len(s))

		st.SetDataAtPathAt(dd[:500], st.NumElementsAtPath(1, 1)-1, 1, 1)
		s = st.Bytes()
		s1 = make([]byte, len(s))
		copy(s1, s)
		st1 = SliceArrayFromBytes(s1)
		require.EqualValues(t, s1, st1.Bytes())
		t.Logf("len with 100+100+(1000-500) bytes data: %d", len(s))
	})
}
