package easyutxo

import (
	"bytes"
	"errors"
	"io"
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
	if len(data) > 256 {
		panic("Params: wrong bytes length")
	}
	if len(a.parsed) >= 256 {
		panic("Params: overflow")
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
		if err := writeArray(a.parsed, &buf); err != nil {
			panic(err)
		}
		a.bytes = buf.Bytes()
	}
}

func (a *Params) Read(r io.Reader) error {
	var sz uint16
	if err := ReadInteger(r, &sz); err != nil {
		return err
	}
	a.bytes = make([]byte, sz)
	a.parsed = nil
	_, err := r.Read(a.bytes)
	return err
}

func (a *Params) Write(w io.Writer) (err error) {
	a.ensureBytes()
	if err = WriteInteger(w, uint16(len(a.bytes))); err != nil {
		return
	}
	_, err = w.Write(a.bytes)
	return
}

func writeArray(d [][]byte, w io.Writer) error {
	if len(d) == 0 {
		return nil
	}
	// write number of elements
	if _, err := w.Write([]byte{byte(len(d) - 1)}); err != nil {
		return err
	}
	// write elements
	for _, d := range d {
		if _, err := w.Write([]byte{byte(len(d))}); err != nil {
			return err
		}
		if _, err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

func parseArray(r io.Reader) ([][]byte, error) {
	var sz [1]byte
	if _, err := r.Read(sz[:]); err != nil {
		return nil, err
	}
	num := int(sz[0]) + 1
	ret := make([][]byte, num)
	for i := range ret {
		if _, err := r.Read(sz[:]); err != nil {
			return nil, err
		}
		if sz[0] == 0 {
			return nil, errors.New("cannot parse Params: bytes size cannot be 0")
		}
		d := make([]byte, sz[0])
		n, err := r.Read(d)
		if err != nil {
			return nil, err
		}
		if n != len(d) {
			return nil, errors.New("wrong bytes size")
		}
		ret[i] = d
	}
	return ret, nil
}
