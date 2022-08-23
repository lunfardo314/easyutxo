package easyutxo

import (
	"bytes"
	"errors"
	"io"
	"math"
)

// LazySlice can be interpreted two ways:
// - as byte slice
// - as serialized append-only array of up to 255 byte slices
// Serialization is optimized by analyzing maximum length of the data element
type LazySlice struct {
	bytes  []byte
	parsed [][]byte
}

func LazySliceFromBytes(data []byte) *LazySlice {
	return &LazySlice{
		bytes: data,
	}
}

func LazySliceEmptyArray() *LazySlice {
	return LazySliceFromBytes(make([]byte, 2))
}

func (a *LazySlice) SetData(data []byte) {
	a.bytes = data
	a.parsed = nil
}

func (a *LazySlice) SetEmptyArray() {
	a.SetData([]byte{0, 0})
}

func (a *LazySlice) Push(data []byte) {
	if len(a.parsed) >= 255 {
		panic("LazySlice.PushDataAtPath: can't contain more than 256 values")
	}
	a.ensureParsed()
	a.parsed = append(a.parsed, data)
	a.bytes = nil // invalidate bytes
}

func (a *LazySlice) SetAt(idx byte, data []byte) {
	a.parsed[idx] = data
	a.bytes = nil // invalidate bytes
}

func (a *LazySlice) ForEach(fun func(i byte, data []byte) bool) {
	for i := byte(0); i < a.NumElements(); i++ {
		if !fun(i, a.At(i)) {
			break
		}
	}
}

func (a *LazySlice) validBytes() bool {
	if len(a.bytes) > 0 {
		return true
	}
	return len(a.parsed) == 0
}

func (a *LazySlice) invalidateBytes() {
	a.bytes = nil
}

func (a *LazySlice) ensureParsed() {
	if a.parsed != nil {
		return
	}
	var err error
	a.parsed, err = parseArray(a.bytes)
	if err != nil {
		panic(err)
	}
}

func (a *LazySlice) ensureBytes() {
	if a.bytes != nil {
		return
	}
	if len(a.parsed) == 0 {
		// bytes == nil
		return
	}
	var buf bytes.Buffer
	if err := encodeArray(a.parsed, &buf); err != nil {
		panic(err)
	}
	a.bytes = buf.Bytes()
}

func (a *LazySlice) At(idx byte) []byte {
	a.ensureParsed()
	return a.parsed[idx]
}

func (a *LazySlice) NumElements() byte {
	a.ensureParsed()
	return byte(len(a.parsed))
}

func (a *LazySlice) Bytes() []byte {
	a.ensureBytes()
	return a.bytes
}

type lenBytesType byte

// prefix of the serialized slice array are two bytes
// 0 byte with ArrayMaxData.. code, the number of bits reserved for element data length
// 1 byte is number of elements in the array
const (
	ArrayLenBytes0  = lenBytesType(0)
	ArrayLenBytes8  = lenBytesType(1)
	ArrayLenBytes16 = lenBytesType(2)
	ArrayLenBytes32 = lenBytesType(4)
)

func writeData(data [][]byte, lenBytes lenBytesType, w io.Writer) error {
	if lenBytes == ArrayLenBytes0 {
		return nil // all empty
	}
	for _, d := range data {
		switch lenBytes {
		case ArrayLenBytes8:
			if err := WriteInteger(w, byte(len(d))); err != nil {
				return err
			}
		case ArrayLenBytes16:
			if err := WriteInteger(w, uint16(len(d))); err != nil {
				return err
			}
		case ArrayLenBytes32:
			if err := WriteInteger(w, uint32(len(d))); err != nil {
				return err
			}
		}
		if _, err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

// decodeElement 'reads' element without memory allocation, just cutting a slice
// from the data. Suitable for immutable data
func decodeElement(buf []byte, dl lenBytesType) ([]byte, []byte, error) {
	var sz int
	switch dl {
	case ArrayLenBytes0:
		sz = 0
	case ArrayLenBytes8:
		sz = int(buf[0])
	case ArrayLenBytes16:
		sz = int(DecodeInteger[uint16](buf[:2]))
	case ArrayLenBytes32:
		sz = int(DecodeInteger[uint32](buf[:4]))
	default:
		return nil, nil, errors.New("wrong lenBytesType value")
	}
	return buf[int(dl)+sz:], buf[int(dl) : int(dl)+sz], nil
}

// decodeData decodes by splitting into slices, reusing the same underlying array
func decodeData(data []byte, dl lenBytesType, n byte) ([][]byte, error) {
	ret := make([][]byte, n)
	var err error
	for i := 0; i < int(n); i++ {
		data, ret[i], err = decodeElement(data, dl)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func calcLenPrefix(data [][]byte) ([2]byte, error) {
	if len(data) > 255 {
		return [2]byte{}, errors.New("can't be more than 255 elements")
	}
	if len(data) == 0 {
		return [2]byte{byte(ArrayLenBytes0), 0}, nil
	}
	dl := lenBytesType(0)
	var t lenBytesType
	for _, d := range data {
		t = ArrayLenBytes0
		switch {
		case len(d) > math.MaxUint32:
			return [2]byte{}, errors.New("data can't be longer that MaxInt32")
		case len(d) > math.MaxUint16:
			t = ArrayLenBytes32
		case len(d) > math.MaxUint8:
			t = ArrayLenBytes16
		case len(d) > 0:
			t = ArrayLenBytes8
		}
		if dl < t {
			dl = t
		}
	}
	return [2]byte{byte(dl), byte(len(data))}, nil
}

func encodeArray(data [][]byte, w io.Writer) error {
	prefix, err := calcLenPrefix(data)
	if err != nil {
		return err
	}
	if _, err = w.Write(prefix[:]); err != nil {
		return err
	}
	return writeData(data, lenBytesType(prefix[0]), w)
}

func parseArray(data []byte) ([][]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("unexpected EOF")
	}
	return decodeData(data[2:], lenBytesType(data[0]), data[1])
}

type LazySliceTree struct {
	sa       *LazySlice
	subtrees map[byte]*LazySliceTree
}

func LazySliceTreeFromBytes(data []byte) *LazySliceTree {
	return &LazySliceTree{
		sa:       LazySliceFromBytes(data),
		subtrees: make(map[byte]*LazySliceTree),
	}
}

func LazySliceTreeEmpty() *LazySliceTree {
	return LazySliceTreeFromBytes([]byte{0, 0})
}

// Bytes recursively updates bytes in the tree
func (st *LazySliceTree) Bytes() []byte {
	if st.sa.validBytes() {
		return st.sa.Bytes()
	}
	for i, str := range st.subtrees {
		st.sa.SetAt(i, str.Bytes())
	}
	return st.sa.Bytes()
}

// takes from cache or creates a subtree, if the data is ns nil
func (st *LazySliceTree) getSubtree(idx byte) *LazySliceTree {
	ret, ok := st.subtrees[idx]
	if ok {
		return ret
	}
	return LazySliceTreeFromBytes(st.sa.At(idx))
}

// PushDataAtPath LazySlice at the end of the path must exist and must be LazySlice
func (st *LazySliceTree) PushDataAtPath(data []byte, path ...byte) {
	if len(path) == 0 {
		st.sa.Push(data)
		return
	}
	subtree := st.getSubtree(path[0])
	subtree.PushDataAtPath(data, path[1:]...)
	st.subtrees[path[0]] = subtree
	st.sa.invalidateBytes()
	return
}

// SetDataAtPathAtIdx LazySlice at the end of the path must exist and must be LazySlice
func (st *LazySliceTree) SetDataAtPathAtIdx(idx byte, data []byte, path ...byte) {
	if len(path) == 0 {
		st.sa.SetAt(idx, data)
		delete(st.subtrees, idx)
		return
	}
	subtree := st.getSubtree(path[0])
	subtree.SetDataAtPathAtIdx(idx, data, path[1:]...)
	st.subtrees[path[0]] = subtree
	st.sa.invalidateBytes()
}

// BytesAtPath returns serialized for of the element at path
func (st *LazySliceTree) BytesAtPath(path ...byte) []byte {
	if len(path) == 0 {
		return st.Bytes()
	}
	subtree := st.getSubtree(path[0])
	ret := subtree.BytesAtPath(path[1:]...)
	st.subtrees[path[0]] = subtree
	return ret
}

func (st *LazySliceTree) GetDataAtPathAtIdx(idx byte, path ...byte) []byte {
	if len(path) == 0 {
		return st.sa.At(idx)
	}
	subtree := st.getSubtree(path[0])
	ret := subtree.GetDataAtPathAtIdx(idx, path[1:]...)
	st.subtrees[path[0]] = subtree
	return ret
}

// PushNewSubtreeAtPath pushes creates a new LazySlice at the end of the path, if it exists
func (st *LazySliceTree) PushNewSubtreeAtPath(path ...byte) {
	st.PushDataAtPath(LazySliceFromBytes(make([]byte, 2)).Bytes(), path...)
}

// NumElementsAtPath returns number of elements of the LazySlice at the end of path
func (st *LazySliceTree) NumElementsAtPath(path ...byte) byte {
	if len(path) == 0 {
		return st.sa.NumElements()
	}
	subtree := st.getSubtree(path[0])
	if subtree == nil {
		panic("subtree cannot be nil")
	}
	return subtree.NumElementsAtPath(path[1:]...)
}
