package funengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var embeddedShort = []*funDef{
	{sym: "$0", numParams: 0},
	{sym: "$1", numParams: 0},
	{sym: "$2", numParams: 0},
	{sym: "$3", numParams: 0},
	{sym: "$4", numParams: 0},
	{sym: "$5", numParams: 0},
	{sym: "$6", numParams: 0},
	{sym: "$7", numParams: 0},
	{sym: "$8", numParams: 0},
	{sym: "$9", numParams: 0},
	{sym: "$10", numParams: 0},
	{sym: "$11", numParams: 0},
	{sym: "$12", numParams: 0},
	{sym: "$13", numParams: 0},
	{sym: "$14", numParams: 0},
	{sym: "$15", numParams: 0},
	{sym: "_data", numParams: 0},
	{sym: "_path", numParams: 0},
	{sym: "_slice", numParams: 3},
	{sym: "_atPath", numParams: 1},
	{sym: "_if", numParams: 3},
	{sym: "_equal", numParams: 2},
	{sym: "_len", numParams: 1},
	{sym: "_not", numParams: 1},
	// 9 left
}

var embeddedLong = []*funDef{
	{sym: "concat", numParams: -1},
	{sym: "and", numParams: -1},
	{sym: "or", numParams: -1},
	{sym: "blake2b", numParams: -1},
	{sym: "validSignature", numParams: 3},
}

const FirstUserFunCode = 1024

var (
	embeddedShortByName = mustPreCompileEmbeddedShortBySym()
	embeddedLongByName  = mustPreCompileEmbeddedLongBySym()
)

func mustMakeMapBySym(defs []*funDef) map[string]*funDef {
	ret := make(map[string]*funDef)
	for i, fd := range defs {
		fd.funCode = uint16(i)
		if _, already := ret[fd.sym]; already {
			panic(fmt.Errorf("repeating symbol '%s'", fd.sym))
		}
		ret[fd.sym] = fd
	}
	return ret
}

func mustPreCompileEmbeddedShortBySym() map[string]*funDef {
	if len(embeddedShort) > 32 {
		panic("failed: len(embeddedShort) <= 32")
	}
	ret := mustMakeMapBySym(embeddedShort)
	for _, fd := range ret {
		if fd.numParams < 0 {
			panic(fmt.Errorf("embedded short must be fixed number of parameters: '%s'", fd.sym))
		}
	}
	return ret
}

func mustPreCompileEmbeddedLongBySym() map[string]*funDef {
	ret := mustMakeMapBySym(embeddedLong)
	// offset fun codes by 32
	for _, fd := range ret {
		fd.funCode += 32
	}
	return ret
}

func compileToLibrary(source string, codeOffset int) (map[string]*funDef, error) {
	lib := make(map[string]*funDef)
	fdefs, err := parseDefinitions(source)
	if err != nil {
		return nil, err
	}
	totalCode := 0
	for _, fd := range fdefs {
		if _, already := embeddedShortByName[fd.sym]; already {
			return nil, fmt.Errorf("repeated symbol '%s'", fd.sym)
		}
		if _, already := embeddedLongByName[fd.sym]; already {
			return nil, fmt.Errorf("repeated symbol '%s'", fd.sym)
		}
		if err = fd.genCode(lib); err != nil {
			return nil, err
		}
		fd.funCode = uint16(codeOffset + len(lib))
		lib[fd.sym] = fd
		fmt.Printf("'%s' code len = %d\n", fd.sym, len(fd.code))
		totalCode += len(fd.code)
	}
	fmt.Printf("total bytes: %d\n", totalCode)
	return lib, nil
}

func (fd *funDef) genCode(lib map[string]*funDef) error {
	var buf bytes.Buffer
	if err := fd.formula.genCode(fd.numParams, lib, &buf); err != nil {
		return err
	}
	fd.code = buf.Bytes()
	return nil
}

func (f *formula) genCode(numArgs int, lib map[string]*funDef, w io.Writer) error {
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
	prefix, shortCall, err := makeCallEmbeddedShortPrefix(f.sym, numArgs, len(f.params))
	if err != nil {
		return err
	}
	if shortCall {
		if _, err = w.Write(prefix); err != nil {
			return err
		}
	} else {
		fd, found := embeddedLongByName[f.sym]
		if !found {
			fd, found = lib[f.sym]
		}
		if !found {
			return fmt.Errorf("can't resolve symbol '%s'", f.sym)
		}
		if fd.numParams >= 0 && fd.numParams != len(f.params) {
			return fmt.Errorf("function '%s' require %d arguments", f.sym, fd.numParams)
		}
	}
	for _, ff := range f.params {
		if err = ff.genCode(numArgs, lib, w); err != nil {
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

func makeCallEmbeddedShortPrefix(sym string, numArgs, numParams int) ([]byte, bool, error) {
	fd, found := embeddedShortByName[sym]
	if !found {
		return nil, false, nil
	}
	if numParams != fd.numParams {
		return nil, false, fmt.Errorf("'%s' takes exactly %d parameters", fd.sym, fd.numParams)
	}
	if strings.HasPrefix(sym, "$") {
		n, _ := strconv.Atoi(sym[1:])
		if n < 0 || n >= numArgs {
			return nil, false, fmt.Errorf("wrong argument reference '%s'", fd.sym)
		}
	}
	return []byte{byte(fd.funCode)}, true, nil
}
