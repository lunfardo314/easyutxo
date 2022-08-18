package easyutxo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// always assume littleendian
var byteOrder = binary.LittleEndian

type integerIntern interface {
	uint8 | int8 | uint16 | int16 | uint32 | int32 | uint64 | int64
}

func ReadInteger[T integerIntern](r io.Reader, pval *T) error {
	return binary.Read(r, byteOrder, pval)
}

func WriteInteger[T integerIntern](w io.Writer, val T) error {
	return binary.Write(w, byteOrder, val)
}

// r/w utility functions

func ReadBytes16(r io.Reader) ([]byte, error) {
	var length uint16
	err := ReadInteger(r, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes16(w io.Writer, data []byte) error {
	if len(data) > math.MaxUint16 {
		panic(fmt.Sprintf("WriteBytes16: too long data (%v)", len(data)))
	}
	err := WriteInteger(w, uint16(len(data)))
	if err != nil {
		return err
	}
	if len(data) != 0 {
		_, err = w.Write(data)
	}
	return err
}

func ReadBytes32(r io.Reader) ([]byte, error) {
	var length uint32
	err := ReadInteger(r, &length)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	ret := make([]byte, length)
	_, err = r.Read(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func WriteBytes32(w io.Writer, data []byte) error {
	if len(data) > math.MaxUint32 {
		panic(fmt.Sprintf("WriteBytes32: too long data (%v)", len(data)))
	}
	err := WriteInteger(w, uint32(len(data)))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

type byteCounter int

func (b *byteCounter) Write(p []byte) (n int, err error) {
	*b = byteCounter(int(*b) + len(p))
	return 0, nil
}

// MustBytes most common way of serialization
func MustBytes(o interface{ Write(w io.Writer) error }) []byte {
	var buf bytes.Buffer
	if err := o.Write(&buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func MustSize(o interface{ Write(w io.Writer) error }) int {
	counter := new(byteCounter)
	if err := o.Write(counter); err != nil {
		panic(err)
	}
	return int(*counter)
}
