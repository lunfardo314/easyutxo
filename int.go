package easyutxo

import (
	"bytes"
	"encoding/binary"
	"io"
)

// assumed endian-ness
var byteOrder = binary.BigEndian

type integerIntern interface {
	uint8 | int8 | uint16 | int16 | uint32 | int32 | uint64 | int64
}

func ReadInteger[T integerIntern](r io.Reader, pval *T) error {
	return binary.Read(r, byteOrder, pval)
}

func WriteInteger[T integerIntern](w io.Writer, val T) error {
	return binary.Write(w, byteOrder, val)
}

func EncodeInteger[T integerIntern](v T) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, byteOrder, v); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func DecodeInteger[T integerIntern](data []byte) T {
	var ret T
	if err := binary.Read(bytes.NewReader(data), byteOrder, &ret); err != nil {
		panic(err)
	}
	return ret
}
