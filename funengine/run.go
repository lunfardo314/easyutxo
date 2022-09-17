package funengine

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/lunfardo314/easyutxo/lazyslice"
)

type RunContext struct {
	localLibrary map[uint16]*funDef
	//globalContext ledger.GlobalContext
}

type InvocationContext struct {
	runContext *RunContext
	path       lazyslice.TreePath
	data       []byte
	callStack  interface{}
}

type CodeReader struct {
	lib  map[uint16]*funDef
	code []byte
	pos  int
}

type funCall struct {
	funDef    *funDef
	callArity int
}

func NewCodeReader(code []byte, lib map[uint16]*funDef) *CodeReader {
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
		ret := embeddedShortByCode[b0]
		if ret == nil {
			return nil, fmt.Errorf("short code %d not found", b0)
		}
		r.pos += 1
		return &funCall{funDef: ret, callArity: ret.numParams}, nil
	}
	// long call
	if r.pos+1 >= len(r.code) {
		return nil, io.EOF
	}
	arity := (b0 & FirstByteLongCallArityMask) >> 4
	t := binary.BigEndian.Uint16([]byte{b0, r.code[r.pos+1]})
	idx := t & Uint16LongCallCodeMask
	ret := embeddedLongByCode[idx]
	if ret == nil {
		return nil, fmt.Errorf("long code %d not found", b0)
	}
	r.pos += 2
	return &funCall{funDef: ret, callArity: int(arity)}, nil
}

func (r *CodeReader) MustNext() interface{} {
	ret, err := r.Next()
	if err != nil {
		panic(err)
	}
	return ret
}
