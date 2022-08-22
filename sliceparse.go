package easyutxo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// SliceArray can be interpreted two ways:
// - as byte slice
// - as serialized append-only array of up to 255 byte slices
// Serialization is optimized by analyzing maximum length of the data element
type SliceArray struct {
	bytes  []byte
	parsed [][]byte
}

func SliceArrayFromBytes(data []byte) *SliceArray {
	return &SliceArray{
		bytes: data,
	}
}

func (a *SliceArray) Push(data []byte) {
	if len(a.parsed) >= 255 {
		panic("SliceArray.PushDataAtPath: can't contain more than 256 values")
	}
	a.parsed = append(a.parsed, data)
	a.bytes = nil // invalidate bytes
}

func (a *SliceArray) SetAt(idx byte, data []byte) {
	a.parsed[idx] = data
	a.bytes = nil // invalidate bytes
}

func (a *SliceArray) IsLeaf() bool {
	return len(a.bytes) > 0 // bytes not invalidated
}

func (a *SliceArray) InvalidateBytes() {
	a.bytes = nil
}

func (a *SliceArray) At(idx byte) []byte {
	a.ensureParsed()
	return a.parsed[idx]
}

func (a *SliceArray) NumElements() byte {
	a.ensureParsed()
	return byte(len(a.parsed))
}

func (a *SliceArray) Bytes() []byte {
	a.ensureBytes()
	return a.bytes
}

func (a *SliceArray) ensureParsed() {
	if a.parsed == nil {
		var err error
		a.parsed, err = parseArray(a.bytes)
		if err != nil {
			panic(err)
		}
	}
}

func (a *SliceArray) ensureBytes() {
	if a.bytes == nil {
		var buf bytes.Buffer
		if err := encodeArray(a.parsed, &buf); err != nil {
			panic(err)
		}
		a.bytes = buf.Bytes()
	}
}

type maxDataLen byte

// prefix of the serializad slize array are two bytes
// 0 byte with ArrayMaxData.. code, the number of bits reserved for element data length
// 1 byte is number of elements in the array
const (
	ArrayMaxDataLen0  = maxDataLen(0)
	ArrayMaxDataLen8  = maxDataLen(8)
	ArrayMaxDataLen16 = maxDataLen(16)
	ArrayMaxDataLen32 = maxDataLen(32)
)

// data encoding is generic for three maximal element size: byte | uint16 | uint32
type dataLenType interface {
	byte | uint16 | uint32
}

func encodeData[L dataLenType](d [][]byte, w io.Writer) error {
	for _, d := range d {
		sz := L(len(d))
		if err := WriteInteger(w, sz); err != nil {
			return err
		}
		if _, err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

// decodeElement 'reads' element without memory allocation, just taking slice
// from data. Suitable for immutable data
func decodeElement(buf []byte, dl maxDataLen) ([]byte, []byte, error) {
	var sz int
	switch dl {
	case ArrayMaxDataLen0:
	case ArrayMaxDataLen8:
		if len(buf) < 1 {
			return nil, nil, errors.New("unexpected EOF")
		}
		sz = int(buf[0])
		buf = buf[1:]
	case ArrayMaxDataLen16:
		if len(buf) < 2 {
			return nil, nil, errors.New("unexpected EOF")
		}
		sz = int(binary.LittleEndian.Uint16(buf[:2]))
		buf = buf[2:]
	case ArrayMaxDataLen32:
		if len(buf) < 4 {
			return nil, nil, errors.New("unexpected EOF")
		}
		sz = int(binary.LittleEndian.Uint32(buf[:4]))
		buf = buf[4:]
	default:
		return nil, nil, errors.New("wrong maxDataLen value")
	}
	if len(buf) < sz {
		return nil, nil, errors.New("unexpected EOF")
	}
	return buf[sz:], buf[:sz], nil
}

// decodeData decodes by splitting into slices, reusing the same underlying array
func decodeData(data []byte, dl maxDataLen, n byte) ([][]byte, error) {
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
		return [2]byte{byte(ArrayMaxDataLen0), 0}, nil
	}
	dl := maxDataLen(0)
	var t maxDataLen
	for _, d := range data {
		switch {
		case len(d) > math.MaxUint32:
			return [2]byte{}, errors.New("data can't be longer that MaxInt32")
		case len(d) > math.MaxUint16:
			t = ArrayMaxDataLen32
		case len(d) > math.MaxUint8:
			t = ArrayMaxDataLen16
		case len(d) > 0:
			t = ArrayMaxDataLen8
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
	switch maxDataLen(prefix[0]) {
	case ArrayMaxDataLen8:
		err = encodeData[byte](data, w)
	case ArrayMaxDataLen16:
		err = encodeData[uint16](data, w)
	case ArrayMaxDataLen32:
		err = encodeData[uint32](data, w)
	}
	return err
}

func parseArray(data []byte) ([][]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("unexpected EOF")
	}
	return decodeData(data[2:], maxDataLen(data[0]), data[1])
}

type SliceTree struct {
	sa     *SliceArray
	cached map[byte]*SliceTree
}

func SliceTreeFromBytes(data []byte) *SliceTree {
	return &SliceTree{
		sa:     SliceArrayFromBytes(data),
		cached: make(map[byte]*SliceTree),
	}
}

// Bytes recursively updates bytes in the tree
func (st *SliceTree) Bytes() []byte {
	if st.sa.IsLeaf() {
		return st.sa.Bytes()
	}
	for i, tr := range st.cached {
		st.sa.SetAt(i, tr.Bytes())
	}
	return st.sa.Bytes()
}

func (st *SliceTree) getChild(idx byte) *SliceTree {
	ret, ok := st.cached[idx]
	if !ok {
		ret = SliceTreeFromBytes(st.sa.At(idx))
		st.cached[idx] = ret
	}
	return ret
}
func (st *SliceTree) BytesAtPath(path ...byte) []byte {
	if len(path) == 0 {
		return st.Bytes()
	}
	return st.getChild(path[0]).BytesAtPath(path[1:]...)
}

// PushDataAtPath SliceArray at the end of the path must exist and must be SliceArray
func (st *SliceTree) PushDataAtPath(data []byte, path ...byte) {
	if len(path) == 0 {
		st.sa.Push(data)
		return
	}
	if path[0] < st.sa.NumElements() {
		tr := st.getChild(path[0])
		tr.PushDataAtPath(data, path[1:]...)
		st.sa.InvalidateBytes()
		return
	}
	panic("path does not exist")
}

// SetDataAtPathAt SliceArray at the end of the path must exist and must be SliceArray
func (st *SliceTree) SetDataAtPathAt(data []byte, idx byte, path ...byte) {
	if len(path) == 0 {
		st.sa.SetAt(idx, data)
		delete(st.cached, idx)
		return
	}
	if path[0] < st.sa.NumElements() {
		tr := st.getChild(path[0])
		tr.SetDataAtPathAt(data, idx, path[1:]...)
		st.sa.InvalidateBytes()
		return
	}
	panic("path does not exist")
}

// PushNewArrayAtPath pushes creates a new SliceArray at the end of the path, if it exists
func (st *SliceTree) PushNewArrayAtPath(path ...byte) {
	st.PushDataAtPath(SliceArrayFromBytes(nil).Bytes(), path...)
}

// NumElementsAtPath returns number of elements of the SliceArray at the end of path
func (st *SliceTree) NumElementsAtPath(path ...byte) byte {
	if len(path) == 0 {
		return st.sa.NumElements()
	}
	return st.getChild(path[0]).NumElementsAtPath(path[1:]...)
}
