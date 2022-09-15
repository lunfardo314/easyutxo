package funengine

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/lunfardo314/easyutxo"
)

var functions = []*funDef{
	{sym: "concat", returnType: "S", varParams: true, paramTypes: []string{"S"}},
	{sym: "slice", returnType: "S", varParams: false, paramTypes: []string{"S", "B", "B"}},
	{sym: "bytesAtPath", returnType: "S", varParams: false, paramTypes: []string{"S", "B", "B"}},
	{sym: "bytes", returnType: "S", varParams: true, paramTypes: []string{"B"}},
	{sym: "path", returnType: "S", varParams: false, paramTypes: nil},
	{sym: "if", returnType: "S", varParams: false, paramTypes: []string{"S", "S", "S"}},
	{sym: "equal", returnType: "S", varParams: true, paramTypes: []string{"S"}},
	{sym: "atIndex", returnType: "S", varParams: false, paramTypes: []string{"S", "B"}},
	{sym: "len", returnType: "B", varParams: false, paramTypes: []string{"S"}},
	{sym: "data", returnType: "S", varParams: false, paramTypes: nil},
	{sym: "and", returnType: "S", varParams: true, paramTypes: []string{"S"}},
	{sym: "not", returnType: "S", varParams: false, paramTypes: []string{"S"}},
	{sym: "validSignature", returnType: "S", varParams: false, paramTypes: []string{"S", "S", "S"}},
	{sym: "blake2b", returnType: "S", varParams: true, paramTypes: []string{"S"}},
	{sym: "param", returnType: "S", varParams: false, paramTypes: []string{"B"}},
}

var functionsBySymbol = musPreCompileLibrary(functions)

func musPreCompileLibrary(functs []*funDef) map[string]*funDef {
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

// prefix[0] || prefix[1] || suffix
// prefix[0] bits
// - 7 : 0 is library function, 1 is inline data
// - if inline data: bits 6-0 is size of the inline data, 0-127
// - if library function, prefix[1] byte comes, bytes 14-0
// - bits 14-11 is arity of the call 0-15
// - bits 10-0 is library reference 0 - 2047

const (
	DataMask = 0x80
)

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

func (fd *formula) genCode(lib map[string]*funDef, w io.Writer) error {
	if len(fd.params) == 0 {
		// terminal condition
		n, err := strconv.Atoi(fd.sym)
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
	dscr, found := lib[fd.sym]
	if !found {
		return fmt.Errorf("cannot resolve symbol '%s'", fd.sym)
	}
	prefix, err := makeCallPrefix(dscr.funCode, byte(len(fd.params)))
	if err != nil {
		return err
	}
	if _, err = w.Write(prefix); err != nil {
		return err
	}
	for _, f := range fd.params {
		if err = f.genCode(lib, w); err != nil {
			return err
		}
	}
	return nil
}

func makeCallPrefix(funCode uint16, callArity byte) ([]byte, error) {
	if callArity > 15 {
		return nil, fmt.Errorf("can't be more than 15 call arguments")
	}
	if funCode > 2047 {
		return nil, fmt.Errorf("wrong function code")
	}
	ret := funCode | uint16(callArity)<<11
	return easyutxo.EncodeInteger(ret), nil
}
