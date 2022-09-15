package funengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

var functions = []*funDef{
	{sym: "_data", numParams: 0},
	{sym: "_param", numParams: 1},
	{sym: "_path", numParams: -1},
	{sym: "concat", numParams: -1},
	{sym: "slice", numParams: 3},
	{sym: "bytesAtPath", numParams: 1},
	{sym: "bytes", numParams: -1},
	{sym: "if", numParams: 3},
	{sym: "equal", numParams: 2},
	{sym: "atIndex", numParams: 2},
	{sym: "len", numParams: 1},
	{sym: "and", numParams: 1},
	{sym: "not", numParams: 1},
	{sym: "validSignature", numParams: 1},
	{sym: "blake2b", numParams: 1},
}

var functionsBySymbol = mustPreCompileLibrary()

func mustPreCompileLibrary() map[string]*funDef {
	ret := make(map[string]*funDef)
	for i, fd := range functions {
		fd.funCode = uint16(i)
		if _, already := ret[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		ret[fd.sym] = fd
	}
	return ret
}

func compileToLibrary(source string, lib map[string]*funDef) error {
	fdefs, err := parseDefinitions(source)
	if err != nil {
		return err
	}
	totalCode := 0
	for _, fd := range fdefs {
		if err = fd.genCode(lib); err != nil {
			return err
		}
		fd.funCode = uint16(len(lib))
		lib[fd.sym] = fd
		fmt.Printf("'%s' code len = %d\n", fd.sym, len(fd.code))
		totalCode += len(fd.code)
	}
	fmt.Printf("total bytes: %d\n", totalCode)
	return nil
}

func (fd *funDef) genCode(lib map[string]*funDef) error {
	var buf bytes.Buffer
	if err := fd.formula.genCode(lib, &buf); err != nil {
		return err
	}
	fd.code = buf.Bytes()
	return nil
}

func (f *formula) genCode(lib map[string]*funDef, w io.Writer) error {
	if len(f.params) == 0 {
		// terminal condition
		n, err := strconv.Atoi(f.sym)
		if err == nil {
			// it is a number
			if n < 0 || n >= 256 {
				return fmt.Errorf("constant value not uint8")
			}
			// it is a byte value
			_, err = w.Write([]byte{DataMask, byte(n)})
			return err
		}
		// not a number
		// TODO other types of literals
	}
	// lookup into the library
	dscr, found := lib[f.sym]
	if !found {
		return fmt.Errorf("cannot resolve symbol '%s'", f.sym)
	}
	prefix, shortCall, err := dscr.makeCallPrefix(f.params)
	if err != nil {
		return err
	}
	if _, err = w.Write(prefix); err != nil {
		return err
	}
	if shortCall {
		return nil
	}
	for _, f := range f.params {
		if err = f.genCode(lib, w); err != nil {
			return err
		}
	}
	return nil
}

const (
	DataMask          = byte(0x80)
	ShortCallMask     = byte(0x40)
	ShortCallCodeMask = ^(DataMask | ShortCallMask)
)

var (
	InvocationDataCallPrefix = []byte{31}
	InvocationPathCallPrefix = []byte{30}
)

// prefix[0] || prefix[1] || suffix
// prefix[0] bits
// - 7 : 0 is library function, 1 is inline data
// - if inline data: bits 6-0 is size of the inline data, 0-127
// - if library function:
//  - if bit 6 is 0, it is inline parameter only byte prefix[0] is used
//  - bits 5-0 ar interpreted inline (values 0-31)
//      value 30 is call to function _path()
//      value 31 is call to function _data()
//      value n = 0-15 it is call to function _param(n)
//      values 16-29 reserved
//  - if bit 6 is 1 (bit 14), it is long and byte prefix[1] is used (total 16 bits)
// - bits 13-10 is arity of the call 0-15
// - bits 9-0 is library reference 0 - 1023

func (fd *funDef) makeCallPrefix(params []*formula) ([]byte, bool, error) {
	if fd.sym == "_data" {
		if len(params) > 0 {
			return nil, false, fmt.Errorf("_data call does not take parameters")
		}
		return InvocationDataCallPrefix, true, nil
	}
	if fd.sym == "_path" {
		if len(params) > 0 {
			return nil, false, fmt.Errorf("_path call does not take parameters")
		}
		return InvocationPathCallPrefix, true, nil
	}
	if fd.sym == "_param" {
		if len(params) != 1 {
			return nil, false, fmt.Errorf("_param call takes exactly 1 parameter")
		}
		parNum, err := strconv.Atoi(params[0].sym)
		if err != nil || parNum < 0 || parNum > 15 {
			return nil, false, fmt.Errorf("wrong parameter number in the _param call")
		}
		if fd.numParams >= 0 && parNum > fd.numParams {
			return nil, false, fmt.Errorf(" _param call is out of range")
		}
		return []byte{byte(parNum)}, true, nil
	}
	if len(params) > 15 {
		return nil, false, fmt.Errorf("can't be more than 15 call arguments")
	}
	if fd.funCode > 1023 {
		return nil, false, fmt.Errorf("wrong function code")
	}
	if fd.numParams >= 0 && len(params) != fd.numParams {
		return nil, false, fmt.Errorf("must be exactly %d argments in the call to '%s': '%s'", fd.numParams, fd.sym, fd.bodySource)
	}
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], fd.funCode|uint16(len(params))<<10)
	return b[:], false, nil
}
