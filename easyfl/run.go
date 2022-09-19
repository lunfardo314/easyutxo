package easyfl

import (
	"encoding/binary"
	"io"
)

type CodeReader struct {
	lib  RuntimeLibrary
	code []byte
	pos  int
}

type funCall struct {
	fun       func()
	callArity int
}

func NewCodeReader(lib RuntimeLibrary, code []byte) *CodeReader {
	return &CodeReader{
		lib:  lib,
		code: code,
	}
}

func (r *CodeReader) Next() (interface{}, error) {
	if r.pos >= len(r.code) {
		return nil, nil
	}
	b0 := r.code[r.pos]
	if b0&FirstByteDataMask != 0 {
		// it is data
		size := int(b0 & FirstByteDataLenMask)
		if len(r.code) < size+1 {
			return nil, io.EOF
		}
		ret := r.code[r.pos+1 : r.pos+1+size]
		r.pos += size + 1
		return ret, nil
	}
	// it is a function call
	if b0&FirstByteLongCallMask == 0 {
		// short call

		r.pos += 1
		return &funCall{
			fun:       r.lib.FunctionByFunCode(uint16(b0), 0),
			callArity: 0,
		}, nil
	}
	// long call
	if r.pos+1 >= len(r.code) {
		return nil, io.EOF
	}
	arity := int((b0 & FirstByteLongCallArityMask) >> 2)
	b1 := r.code[r.pos+1]
	t := binary.BigEndian.Uint16([]byte{b0, b1})
	idx := t & Uint16LongCallCodeMask
	r.pos += 2
	return &funCall{
		fun:       r.lib.FunctionByFunCode(idx, arity),
		callArity: arity,
	}, nil
}

func (r *CodeReader) MustNext() interface{} {
	ret, err := r.Next()
	if err != nil {
		panic(err)
	}
	return ret
}
