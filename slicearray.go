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
	if len(a.parsed) >= math.MaxUint16 {
		panic("SliceArray.Push: overflow")
	}
	a.parsed = append(a.parsed, data)
	a.bytes = nil
}

func (a *SliceArray) At(idx int) []byte {
	a.ensureParsed()
	return a.parsed[idx]
}

func (a *SliceArray) NumElements() int {
	a.ensureParsed()
	return len(a.parsed)
}

func (a *SliceArray) Bytes() []byte {
	a.ensureBytes()
	return a.bytes
}

func (a *SliceArray) ensureParsed() {
	if a.parsed == nil {
		var err error
		a.parsed, err = parseArray(bytes.NewReader(a.bytes))
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
	ArrayMaxDataLen0     = maxDataLen(0)
	ArrayMaxDataLen8     = maxDataLen(8)
	ArrayMaxDataLen16    = maxDataLen(16)
	ArrayMaxDataLen32    = maxDataLen(32)
	ArrayMaxDataLenWrong = maxDataLen(64)

	ArrayNumElementUint16 = 0x80
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

func decodeData[L dataLenType](r io.Reader, n int) ([][]byte, error) {
	ret := make([][]byte, n)
	var sz L
	for i := range ret {
		if err := ReadInteger(r, &sz); err != nil {
			return nil, err
		}
		ret[i] = make([]byte, sz)
		if _, err := r.Read(ret[i]); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func calcMaxDataLen(data [][]byte) maxDataLen {
	if len(data) == 0 {
		return ArrayMaxDataLen0
	}
	dl := ArrayMaxDataLen8
	for _, d := range data {
		switch {
		case len(d) > math.MaxUint32:
			return ArrayMaxDataLenWrong // not supported
		case len(d) > math.MaxUint16:
			return ArrayMaxDataLen32 // max 4 bytes for data len is supported
		case len(d) > math.MaxUint8:
			dl = ArrayMaxDataLen16
		}
	}
	return dl
}

type lenPrefix struct {
	maxDataLen        maxDataLen
	numElements       int
	numElementsUint16 bool
}

// calcPrefix encodes sizes and num elements into 2 or 3 bytes
func calcPrefix(data [][]byte) (lenPrefix, error) {
	ret := lenPrefix{
		numElements: len(data),
	}
	ret.maxDataLen = calcMaxDataLen(data)
	if ret.maxDataLen == ArrayMaxDataLenWrong {
		return lenPrefix{}, errors.New("wrong data length")
	}
	if len(data) > math.MaxUint16 {
		return lenPrefix{}, errors.New("too manny elements")
	}
	if len(data) > math.MaxUint8 {
		ret.numElementsUint16 = true
	}
	return ret, nil
}

func (p *lenPrefix) Write(w io.Writer) error {
	var pref [3]byte
	prefix := pref[:0]
	prefix = append(prefix, byte(p.maxDataLen))
	if p.numElementsUint16 {
		prefix[0] |= ArrayNumElementUint16
		var sz [2]byte
		binary.LittleEndian.PutUint16(sz[:], uint16(p.numElements))
		prefix = append(prefix, sz[0], sz[1])
	} else {
		prefix = append(prefix, byte(p.numElements))
	}
	_, err := w.Write(prefix)
	return err
}

func (p *lenPrefix) Read(r io.Reader) error {
	var t [1]byte
	if _, err := r.Read(t[:]); err != nil {
		return err
	}
	p.numElementsUint16 = (t[0] & ArrayNumElementUint16) != 0
	p.maxDataLen = maxDataLen(t[0] & ^byte(ArrayNumElementUint16))
	if p.numElementsUint16 {
		var sz [2]byte
		if _, err := r.Read(sz[:]); err != nil {
			return err
		}
		p.numElements = int(binary.LittleEndian.Uint16(sz[:]))
	} else {
		var sz [1]byte
		if _, err := r.Read(sz[:]); err != nil {
			return err
		}
		p.numElements = int(sz[0])
	}
	return nil
}

func encodeArray(data [][]byte, w io.Writer) error {
	prefix, err := calcPrefix(data)
	if err != nil {
		return err
	}
	if err = prefix.Write(w); err != nil {
		return err
	}
	switch prefix.maxDataLen {
	case ArrayMaxDataLen8:
		err = encodeData[byte](data, w)
	case ArrayMaxDataLen16:
		err = encodeData[uint16](data, w)
	case ArrayMaxDataLen32:
		err = encodeData[uint32](data, w)
	}
	return err
}

func parseArray(r io.Reader) ([][]byte, error) {
	var prefix lenPrefix
	if err := prefix.Read(r); err != nil {
		return nil, err
	}
	switch prefix.maxDataLen {
	case ArrayMaxDataLen0:
		return nil, nil
	case ArrayMaxDataLen8:
		return decodeData[byte](r, prefix.numElements)
	case ArrayMaxDataLen16:
		return decodeData[uint16](r, prefix.numElements)
	case ArrayMaxDataLen32:
		return decodeData[uint32](r, prefix.numElements)
	}
	return nil, errors.New("wrong data len code")
}
