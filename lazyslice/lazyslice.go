package lazyslice

import (
	"bytes"
	"errors"
	"io"
	"math"

	"github.com/lunfardo314/easyutxo"
)

// LazySlice can be interpreted two ways:
// - as byte slice
// - as serialized append-only array of up to 255 byte slices
// Serialization is optimized by analyzing maximum length of the data element
type LazySlice struct {
	bytes          []byte
	parsed         [][]byte
	maxNumElements int
}

type lenPrefixType uint16

// prefix of the serialized slice array are two bytes interpreted as uint16
// The highest 2 bits (mask 0xC0) are interpreted as 4 possible DataLenBytes (0, 1, 2 and 4 bytes)
// The rest is interpreted as uint16 of the number of elements in the array. Max 2^14-1 =
// 0 byte with ArrayMaxData code, the number of bits reserved for element data length
// 1 byte is number of elements in the array
const (
	DataLenBytes0  = uint16(0x00) << 14
	DataLenBytes8  = uint16(0x01) << 14
	DataLenBytes16 = uint16(0x02) << 14
	DataLenBytes32 = uint16(0x03) << 14

	DataLenMask  = uint16(0x03) << 14
	ArrayLenMask = ^DataLenMask
	MaxArrayLen  = int(ArrayLenMask) // 16383

	emptyArrayPrefix = lenPrefixType(0)
)

func (dl lenPrefixType) DataLenBytes() int {
	m := uint16(dl) & DataLenMask
	switch m {
	case DataLenBytes0:
		return 0
	case DataLenBytes8:
		return 1
	case DataLenBytes16:
		return 2
	case DataLenBytes32:
		return 4
	}
	panic("very bad")
}

func (dl lenPrefixType) NumElements() int {
	s := uint16(dl) & ArrayLenMask
	return int(s)
}

func (dl lenPrefixType) Bytes() []byte {
	return easyutxo.EncodeInteger(uint16(dl))
}

func LazySliceFromBytes(data []byte) *LazySlice {
	return &LazySlice{
		bytes:          data,
		maxNumElements: MaxArrayLen,
	}
}

func LazySliceEmptyArray() *LazySlice {
	return LazySliceFromBytes(emptyArrayPrefix.Bytes())
}

func (a *LazySlice) WithMaxNumberOfElements(m int) *LazySlice {
	a.maxNumElements = m
	if a.maxNumElements > MaxArrayLen {
		a.maxNumElements = MaxArrayLen
	}
	return a
}

func (a *LazySlice) SetData(data []byte) {
	a.bytes = data
	a.parsed = nil
}

func (a *LazySlice) SetEmptyArray() {
	a.SetData([]byte{0, 0})
}

func (a *LazySlice) Push(data []byte) {
	if len(a.parsed) >= a.maxNumElements {
		panic("LazySlice.Push: too many elements")
	}
	a.ensureParsed()
	a.parsed = append(a.parsed, data)
	a.bytes = nil // invalidate bytes
}

func (a *LazySlice) SetAtIdx(idx byte, data []byte) {
	a.parsed[idx] = data
	a.bytes = nil // invalidate bytes
}

func (a *LazySlice) ForEach(fun func(i int, data []byte) bool) {
	for i := 0; i < a.NumElements(); i++ {
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

func (a *LazySlice) At(idx int) []byte {
	a.ensureParsed()
	return a.parsed[idx]
}

func (a *LazySlice) NumElements() int {
	a.ensureParsed()
	return len(a.parsed)
}

func (a *LazySlice) Bytes() []byte {
	a.ensureBytes()
	return a.bytes
}

func calcLenPrefix(data [][]byte) (lenPrefixType, error) {
	if len(data) > MaxArrayLen {
		return 0, errors.New("too long data")
	}
	if len(data) == 0 {
		return 0, nil
	}
	var dl uint16
	var t uint16
	for _, d := range data {
		t = DataLenBytes0
		switch {
		case len(d) > math.MaxUint32:
			return 0, errors.New("data can't be longer that MaxInt32")
		case len(d) > math.MaxUint16:
			t = DataLenBytes32
		case len(d) > math.MaxUint8:
			t = DataLenBytes16
		case len(d) > 0:
			t = DataLenBytes8
		}
		if dl < t {
			dl = t
		}
	}
	return lenPrefixType(dl | uint16(len(data))), nil
}

func writeData(data [][]byte, numDataLenBytes int, w io.Writer) error {
	if numDataLenBytes == 0 {
		return nil // all empty
	}
	for _, d := range data {
		switch numDataLenBytes {
		case 1:
			if err := easyutxo.WriteInteger(w, byte(len(d))); err != nil {
				return err
			}
		case 2:
			if err := easyutxo.WriteInteger(w, uint16(len(d))); err != nil {
				return err
			}
		case 4:
			if err := easyutxo.WriteInteger(w, uint32(len(d))); err != nil {
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
func decodeElement(buf []byte, numDataLenBytes int) ([]byte, []byte, error) {
	var sz int
	switch numDataLenBytes {
	case 0:
		sz = 0
	case 1:
		sz = int(buf[0])
	case 2:
		sz = int(easyutxo.DecodeInteger[uint16](buf[:2]))
	case 4:
		sz = int(easyutxo.DecodeInteger[uint32](buf[:4]))
	default:
		return nil, nil, errors.New("wrong lenPrefixType value")
	}
	return buf[numDataLenBytes+sz:], buf[numDataLenBytes : numDataLenBytes+sz], nil
}

// decodeData decodes by splitting into slices, reusing the same underlying array
func decodeData(data []byte, numDataLenBytes int, n int) ([][]byte, error) {
	ret := make([][]byte, n)
	var err error
	for i := 0; i < n; i++ {
		data, ret[i], err = decodeElement(data, numDataLenBytes)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func encodeArray(data [][]byte, w io.Writer) error {
	prefix, err := calcLenPrefix(data)
	if err != nil {
		return err
	}
	if _, err = w.Write(prefix.Bytes()); err != nil {
		return err
	}
	return writeData(data, prefix.DataLenBytes(), w)
}

func parseArray(data []byte) ([][]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("unexpected EOF")
	}
	prefix := lenPrefixType(easyutxo.DecodeInteger[uint16](data[:2]))
	return decodeData(data[2:], prefix.DataLenBytes(), prefix.NumElements())
}

type LazySliceTree struct {
	sa       *LazySlice
	subtrees map[byte]*LazySliceTree
}

func LazySliceTreeFromBytes(data []byte) *LazySliceTree {
	return &LazySliceTree{
		sa:       LazySliceFromBytes(data).WithMaxNumberOfElements(256), // max index is 255
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
		st.sa.SetAtIdx(i, str.Bytes())
	}
	return st.sa.Bytes()
}

// takes from cache or creates a subtree, if the data is ns nil
func (st *LazySliceTree) getSubtree(idx byte) *LazySliceTree {
	ret, ok := st.subtrees[idx]
	if ok {
		return ret
	}
	return LazySliceTreeFromBytes(st.sa.At(int(idx)))
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
		st.sa.SetAtIdx(idx, data)
		delete(st.subtrees, idx)
		return
	}
	subtree := st.getSubtree(path[0])
	subtree.SetDataAtPathAtIdx(idx, data, path[1:]...)
	st.subtrees[path[0]] = subtree
	st.sa.invalidateBytes()
}

func (st *LazySliceTree) SetEmptyArrayAtPathAtIdx(idx byte, path ...byte) {
	st.SetDataAtPathAtIdx(idx, emptyArrayPrefix.Bytes(), path...)
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
		return st.sa.At(int(idx))
	}
	subtree := st.getSubtree(path[0])
	ret := subtree.GetDataAtPathAtIdx(idx, path[1:]...)
	st.subtrees[path[0]] = subtree
	return ret
}

// PushNewSubtreeAtPath pushes creates a new LazySlice at the end of the path, if it exists
func (st *LazySliceTree) PushNewSubtreeAtPath(path ...byte) {
	st.PushDataAtPath(LazySliceFromBytes(emptyArrayPrefix.Bytes()).Bytes(), path...)
}

// NumElementsAtPath returns number of elements of the LazySlice at the end of path
func (st *LazySliceTree) NumElementsAtPath(path ...byte) byte {
	if len(path) == 0 {
		return byte(st.sa.NumElements())
	}
	subtree := st.getSubtree(path[0])
	if subtree == nil {
		panic("subtree cannot be nil")
	}
	return subtree.NumElementsAtPath(path[1:]...)
}

// PushLayerTwo is needed when we want to have lists with more than 255 elements.
// We do two leveled tree and address each element with uint16 or two bytes
func (st *LazySliceTree) PushLayerTwo(data []byte) {
	n := st.NumElementsAtPath()
	idx := byte(0)
	for ; idx < n; idx++ {
		if st.NumElementsAtPath(idx) < 255 {
			st.PushDataAtPath(data, idx)
			return
		}
	}
	st.PushNewSubtreeAtPath()
	st.PushDataAtPath(data, idx)
}

func (st *LazySliceTree) AtIdxLayerTwo(idx uint16) []byte {
	return st.BytesAtPath(easyutxo.EncodeInteger(idx)...)
}
