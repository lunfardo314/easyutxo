package easyutxo

import (
	"bytes"
	"errors"
	"io"
)

// DataArray append-only non-empty array of non-empty data slices with length <= 256 bytes
type DataArray struct {
	a [][]byte
}

func NewDataArray() *DataArray {
	return &DataArray{
		a: make([][]byte, 0),
	}
}

func ParseDataArray(data []byte) (*DataArray, error) {
	ret := NewDataArray()
	if err := ret.Read(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return ret, nil
}

func (a *DataArray) Push(data []byte) {
	if len(data) == 0 || len(data) > 256 {
		panic("DataArray: wrong data length")
	}
	if len(a.a) >= 256 {
		panic("DataArray: overflow")
	}
	a.a = append(a.a, data)
}

func (a *DataArray) At(idx byte) []byte {
	return a.a[idx]
}

func (a *DataArray) Len() int {
	return len(a.a)
}

func (a *DataArray) Write(w io.Writer) error {
	if len(a.a) == 0 {
		return errors.New("cannot serialize empty DataArray")
	}
	// write number of elements
	if _, err := w.Write([]byte{byte(len(a.a) - 1)}); err != nil {
		return err
	}
	// write elements
	for _, d := range a.a {
		if _, err := w.Write([]byte{byte(len(d))}); err != nil {
			return err
		}
		if _, err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

func (a *DataArray) Read(r io.Reader) error {
	var sz [1]byte
	if _, err := r.Read(sz[:]); err != nil {
		return err
	}
	a.a = a.a[:0]
	num := int(sz[0]) + 1
	for i := 0; i < num; i++ {
		if _, err := r.Read(sz[:]); err != nil {
			return err
		}
		if sz[0] == 0 {
			return errors.New("cannot parse DataArray: data size cannot be 0")
		}
		d := make([]byte, sz[0])
		n, err := r.Read(d)
		if err != nil {
			return err
		}
		if n != len(d) {
			return errors.New("wrong data size")
		}
		a.a = append(a.a, d)
	}
	return nil
}
