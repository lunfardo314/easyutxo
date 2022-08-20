package easyutxo

import (
	"bytes"
	"errors"
	"io"
	"math"
)

// Params can be interpreted two ways:
// - as byte slice
// - as serialized append-only non-empty array of byte slices with length <= 256 bytes each
// Is deserialized/serialized only if accessed as array
type Params struct {
	bytes  []byte
	parsed [][]byte
}

func NewParams(data []byte) *Params {
	return &Params{
		bytes: data,
	}
}

func (a *Params) Push(data []byte) {
	if len(a.parsed) >= 256 {
		panic("Params.Push: overflow")
	}
	a.parsed = append(a.parsed, data)
	a.bytes = nil
}

func (a *Params) At(idx byte) []byte {
	a.ensureParsed()
	return a.parsed[idx]
}

func (a *Params) NumElements() int {
	a.ensureParsed()
	return len(a.parsed)
}

func (a *Params) Bytes() []byte {
	a.ensureBytes()
	return a.bytes
}

func (a *Params) ensureParsed() {
	if a.parsed == nil {
		var err error
		a.parsed, err = parseArray(bytes.NewReader(a.bytes))
		if err != nil {
			panic(err)
		}
	}
}

func (a *Params) ensureBytes() {
	if a.bytes == nil {
		var buf bytes.Buffer
		if err := encodeArray(a.parsed, &buf); err != nil {
			panic(err)
		}
		a.bytes = buf.Bytes()
	}
}

type maxDataLen byte

const (
	ArrayMaxDataLen0     = maxDataLen(0)
	ArrayMaxDataLen8     = maxDataLen(8)
	ArrayMaxDataLen16    = maxDataLen(16)
	ArrayMaxDataLen32    = maxDataLen(32)
	ArrayMaxDataLenWrong = maxDataLen(64)
)

type dataLen interface {
	byte | uint16 | uint32
}

func encodeData[T dataLen](d [][]byte, w io.Writer) error {
	for _, d := range d {
		sz := T(len(d))
		if err := WriteInteger(w, sz); err != nil {
			return err
		}
		if _, err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

func decodeData[T dataLen](r io.Reader, n byte) ([][]byte, error) {
	ret := make([][]byte, n)
	var sz T
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

func maxLen(data [][]byte) maxDataLen {
	if len(data) == 0 {
		return ArrayMaxDataLen0
	}
	dl := ArrayMaxDataLen8
	for _, d := range data {
		switch {
		case len(d) > math.MaxUint32:
			return ArrayMaxDataLenWrong
		case len(d) > math.MaxUint16:
			return ArrayMaxDataLen32
		case len(d) > math.MaxUint8:
			dl = ArrayMaxDataLen16
		}
	}
	return dl
}

func encodeArray(data [][]byte, w io.Writer) error {
	if len(data) > math.MaxUint8 {
		return errors.New("array cannot contain more that 255 elements")
	}
	ml := maxLen(data)
	if ml == ArrayMaxDataLenWrong {
		return errors.New("wrong data length")
	}
	prefix := [2]byte{byte(ml), byte(len(data))}
	if _, err := w.Write(prefix[:]); err != nil {
		return err
	}
	var err error
	switch ml {
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
	var prefix [2]byte
	if _, err := r.Read(prefix[:]); err != nil {
		return nil, err
	}
	ml := maxDataLen(prefix[0])
	n := prefix[1]
	switch ml {
	case ArrayMaxDataLen0:
		return nil, nil
	case ArrayMaxDataLen8:
		return decodeData[byte](r, n)
	case ArrayMaxDataLen16:
		return decodeData[uint16](r, n)
	case ArrayMaxDataLen32:
		return decodeData[uint32](r, n)
	}
	return nil, errors.New("wrong data len code")
}
