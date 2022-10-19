package lazyslice

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/lunfardo314/easyutxo"
)

// Array can be interpreted two ways:
// - as byte slice
// - as serialized append-only array of up to 255 byte slices
// Serialization is optimized by analyzing maximum length of the data element
type Array struct {
	bytes          []byte
	parsed         [][]byte
	userBit0       bool
	userBit1       bool
	maxNumElements int
}

type lenPrefixType uint16

// prefix of the serialized slice array are two bytes interpreted as uint16
// The highest 2 bits are interpreted as 4 possible DataLenBytes (0, 1, 2 and 4 bytes)
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

func ArrayFromBytes(data []byte, maxNumElements ...int) *Array {
	mx := MaxArrayLen
	if len(maxNumElements) > 0 {
		mx = maxNumElements[0]
	}
	return &Array{
		bytes:          data,
		maxNumElements: mx,
	}
}

func EmptyArray(maxNumElements ...int) *Array {
	return ArrayFromBytes(emptyArrayPrefix.Bytes(), maxNumElements...)
}

func (a *Array) SetData(data []byte) {
	a.bytes = data
	a.parsed = nil
}

func (a *Array) SetEmptyArray() {
	a.SetData([]byte{0, 0})
}

func (a *Array) IsEmpty() bool {
	return a.NumElements() == 0
}

func (a *Array) IsFull() bool {
	return a.NumElements() >= a.maxNumElements
}

func (a *Array) Push(data []byte) int {
	if len(a.parsed) >= a.maxNumElements {
		panic("Array.Push: too many elements")
	}
	a.ensureParsed()
	a.parsed = append(a.parsed, data)
	a.bytes = nil // invalidate bytes
	return len(a.parsed) - 1
}

func (a *Array) PutAtIdx(idx byte, data []byte) {
	a.parsed[idx] = data
	a.bytes = nil // invalidate bytes
}

func (a *Array) PushEmptyElements(n int) {
	for i := 0; i < n; i++ {
		a.Push(nil)
	}
}

func (a *Array) ForEach(fun func(i int, data []byte) bool) {
	for i := 0; i < a.NumElements(); i++ {
		if !fun(i, a.At(i)) {
			break
		}
	}
}

func (a *Array) validBytes() bool {
	if len(a.bytes) > 0 {
		return true
	}
	return len(a.parsed) == 0
}

func (a *Array) invalidateBytes() {
	a.bytes = nil
}

func (a *Array) ensureParsed() {
	if a.parsed != nil {
		return
	}
	var err error
	a.parsed, err = parseArray(a.bytes, a.maxNumElements)
	if err != nil {
		panic(err)
	}
}

func (a *Array) ensureBytes() {
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

func (a *Array) At(idx int) []byte {
	a.ensureParsed()
	return a.parsed[idx]
}

func (a *Array) NumElements() int {
	a.ensureParsed()
	return len(a.parsed)
}

func (a *Array) Bytes() []byte {
	a.ensureBytes()
	return a.bytes
}

func (a *Array) AsTree() *Tree {
	return &Tree{
		sa:       a,
		subtrees: make(map[byte]*Tree),
	}
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
	if len(data) != 0 {
		return nil, errors.New("serialization error: not all bytes were consumed")
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

func parseArray(data []byte, maxNumElements int) ([][]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("unexpected EOF")
	}
	prefix := lenPrefixType(easyutxo.DecodeInteger[uint16](data[:2]))
	if prefix.NumElements() > maxNumElements {
		return nil, fmt.Errorf("parseArray: number of elements in the prefix %d is larger than maxNumElements %d ",
			prefix.NumElements(), maxNumElements)
	}
	return decodeData(data[2:], prefix.DataLenBytes(), prefix.NumElements())
}

//------------------------------------------------------------------------------

type Tree struct {
	// bytes
	sa *Array
	// cache of parsed subtrees
	subtrees map[byte]*Tree
}

type TreePath []byte

const MaxElementsLazyTree = 256

func TreeFromBytes(data []byte) *Tree {
	return &Tree{
		sa:       ArrayFromBytes(data, MaxElementsLazyTree),
		subtrees: make(map[byte]*Tree),
	}
}

func TreeEmpty() *Tree {
	return TreeFromBytes(emptyArrayPrefix.Bytes())
}

func Path(p ...interface{}) TreePath {
	return easyutxo.Concat(p...)
}

// PathMakeAppend allocates a new underlying array and appends bytes to it
func PathMakeAppend(p TreePath, b ...byte) TreePath {
	ret := make(TreePath, len(p), len(p)+len(b))
	copy(ret, p)
	return append(ret, b...)
}

func (p TreePath) Bytes() []byte {
	return p
}

func (p TreePath) String() string {
	return fmt.Sprintf("%v", []byte(p))
}

// Bytes recursively updates bytes in the tree from leaves
func (st *Tree) Bytes() []byte {
	if st.sa.validBytes() {
		return st.sa.Bytes()
	}
	for i, subtree := range st.subtrees {
		if !subtree.sa.validBytes() {
			st.sa.PutAtIdx(i, subtree.Bytes())
		}
	}
	return st.sa.Bytes()
}

// takes from cache or creates a subtree
func (st *Tree) getSubtree(idx byte) *Tree {
	ret, ok := st.subtrees[idx]
	if ok {
		return ret
	}
	return TreeFromBytes(st.sa.At(int(idx)))
}

// PushData Array at the end of the global path must exist and must be Array
func (st *Tree) PushData(data []byte, path TreePath) int {
	if len(path) == 0 {
		return st.sa.Push(data)
	}
	subtree := st.getSubtree(path[0])
	ret := subtree.PushData(data, path[1:])
	st.subtrees[path[0]] = subtree
	st.sa.invalidateBytes()
	return ret
}

// PutDataAtIdx Array at the end of the globalpath must exist and must be Array
func (st *Tree) PutDataAtIdx(idx byte, data []byte, path TreePath) {
	if len(path) == 0 {
		st.sa.PutAtIdx(idx, data)
		delete(st.subtrees, idx)
		return
	}
	subtree := st.getSubtree(path[0])
	subtree.PutDataAtIdx(idx, data, path[1:])
	st.subtrees[path[0]] = subtree
	st.sa.invalidateBytes()
}

func (st *Tree) SetEmptyArrayAtIdx(idx byte, path TreePath) {
	st.PutDataAtIdx(idx, emptyArrayPrefix.Bytes(), path)
}

func (st *Tree) Subtree(path TreePath) *Tree {
	if len(path) == 0 {
		return st
	}
	subtree := st.getSubtree(path[0])
	ret := subtree.Subtree(path[1:])
	st.subtrees[path[0]] = subtree
	return ret
}

// BytesAtPath returns serialized for of the element at globalpath
func (st *Tree) BytesAtPath(path TreePath) []byte {
	if len(path) == 0 {
		return st.Bytes()
	}
	if len(path) == 1 {
		return st.sa.At(int(path[0]))
	}
	subtree := st.getSubtree(path[0])
	ret := subtree.BytesAtPath(path[1:])
	st.subtrees[path[0]] = subtree
	return ret
}

func (st *Tree) GetDataAtIdx(idx byte, path TreePath) []byte {
	st.BytesAtPath(path) // updates invalidated bytes
	return st.Subtree(path).sa.At(int(idx))
}

func (st *Tree) PushSubtreeFromBytes(data []byte, path TreePath) int {
	return st.PushData(data, path)
}

// PushEmptySubtrees pushes creates a new Array at the end of the globalpath, if it exists
func (st *Tree) PushEmptySubtrees(n int, path TreePath) {
	for i := 0; i < n; i++ {
		st.PushSubtreeFromBytes(emptyArrayPrefix.Bytes(), path)
	}
}

// PushSubtree pushes data and parsed tree of that data.
// Only correct is pushed tree is read-only (e.g. library)
func (st *Tree) PushSubtree(tr *Tree, path TreePath) int {
	subtree := st.Subtree(path)
	ret := subtree.sa.Push(tr.Bytes())
	subtree.subtrees[byte(subtree.sa.NumElements()-1)] = tr
	return ret
}

func (st *Tree) PutSubtreeAtIdx(tr *Tree, idx byte, path TreePath) {
	subtree := st.Subtree(path)
	subtree.sa.PutAtIdx(idx, tr.Bytes())
	subtree.subtrees[idx] = tr
}

// NumElements returns number of elements of the Array at the end of globalpath
func (st *Tree) NumElements(path TreePath) int {
	return st.Subtree(path).sa.NumElements()
}

func (st *Tree) IsEmpty(path TreePath) bool {
	return st.NumElements(path) == 0
}

func (st *Tree) IsFullAtPath(path TreePath) bool {
	return st.NumElements(path) >= 256
}

func (st *Tree) ForEach(fun func(i byte, data []byte) bool, path TreePath) {
	sub := st.Subtree(path)
	for i := 0; i < sub.sa.NumElements(); i++ {
		fun(byte(i), sub.sa.At(i))
	}
}

func (st *Tree) ForEachIndex(fun func(i byte) bool, path TreePath) {
	for i := 0; i < st.NumElements(path); i++ {
		if !fun(byte(i)) {
			break
		}
	}
}

func (st *Tree) ForEachSubtree(fun func(idx byte, subtree *Tree) bool, path TreePath) {
	subtree := st.Subtree(path)
	subtree.ForEachIndex(func(i byte) bool {
		subtree1 := subtree.Subtree(Path(i))
		return fun(i, subtree1)
	}, nil)
}
