package easyfl

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type parsedFormula struct {
	sym    string
	params []*parsedFormula
}

// ParseFunctions parses many function definitions
func ParseFunctions(s string) ([]*FunParsed, error) {
	lines := splitLinesStripComments(s)
	ret, err := parseDefs(lines)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func splitLinesStripComments(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "//")
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func parseDefs(lines []string) ([]*FunParsed, error) {
	ret := make([]*FunParsed, 0)
	var current *FunParsed
	for lineno, line := range lines {
		if strings.HasPrefix(line, "func ") {
			if current != nil {
				current.SourceCode = stripSpaces(current.SourceCode)
				ret = append(ret, current)
			}
			sym, body, found := strings.Cut(strings.TrimPrefix(line, "func "), ":")
			if !found {
				return nil, fmt.Errorf("':' expectected @ line %d", lineno)
			}
			current = &FunParsed{
				Sym:        strings.TrimSpace(sym),
				SourceCode: body,
			}
		} else {
			if len(stripSpaces(line)) == 0 {
				continue
			}
			if current == nil {
				return nil, fmt.Errorf("unexpectected symbols @ line %d", lineno)
			}
			current.SourceCode += line
		}
	}
	if current != nil {
		current.SourceCode = stripSpaces(current.SourceCode)
		ret = append(ret, current)
	}
	return ret, nil
}

func stripSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// if the character is a space, drop it
			return -1
		}
		// else keep it in the string
		return r
	}, str)
}

func parseFormula(s string) (*parsedFormula, error) {
	name, rest, foundOpen := strings.Cut(s, "(")
	f := &parsedFormula{
		sym:    name,
		params: make([]*parsedFormula, 0),
	}
	if !foundOpen {
		if strings.Contains(rest, ")") || strings.Contains(rest, ",") {
			return nil, fmt.Errorf("unexpected ')': '%s'", s)
		}
		return f, nil
	}
	spl, err := splitArgs(rest)
	if err != nil {
		return nil, err
	}
	for _, call := range spl {
		ff, err := parseFormula(call)
		if err != nil {
			return nil, err
		}
		f.params = append(f.params, ff)
	}
	return f, nil
}

// parseArgs expects ','-delimited list of calls, which ends with ')'
func splitArgs(argsStr string) ([]string, error) {
	ret := make([]string, 0)
	var buf bytes.Buffer
	level := 0
	for _, c := range []byte(argsStr) {
		if level < 0 {
			return nil, fmt.Errorf("unbalanced paranthesis: '%s'", argsStr)
		}
		switch c {
		case ',':
			if level == 0 {
				p := make([]byte, len(buf.Bytes()))
				copy(p, buf.Bytes())
				ret = append(ret, string(p))
				buf.Reset()
			} else {
				buf.WriteByte(c)
			}
		case '(':
			buf.WriteByte(c)
			level++
		case ')':
			level--
			if level >= 0 {
				buf.WriteByte(c)
			}
		default:
			buf.WriteByte(c)
		}
	}
	if level != -1 {
		return nil, fmt.Errorf("unclosed '(': '%s'", argsStr)
	}
	if len(buf.Bytes()) > 0 {
		p := make([]byte, len(buf.Bytes()))
		copy(p, buf.Bytes())
		ret = append(ret, string(p))
	}
	return ret, nil
}

// prefix[0] || prefix[1] || suffix
// prefix[0] bits
// - bit 7 (FirstByteDataMask) : 0 is library function, 1 is inline data
// - if inline data: bits 6-0 is size of the inline data, 0-127
// - if library function:
//  - if bit 6 (FirstByteLongCallMask) is 0, it is inline parameter only byte prefix[0] is used
//  - bits 5-0 are interpreted inline (values 0-63) call to short embedded function with fixed arity
//    some values are used, some are reserved
//  - if bit 6 (FirstByteLongCallMask) is 1, it is long call.
//  -- bits 5-2 are interpreted as arity of the long call (values 0-15)
//  -- the prefix[0] byte is extended with prefix[1] the prefix 1-2 is interpreted as uint16 bigendian
//  -- bits 9-0 of uint16 is the long code of the called function (values 0-1023)

const (
	FirstByteDataMask          = byte(0x01) << 7
	FirstByteDataLenMask       = ^FirstByteDataMask
	FirstByteLongCallMask      = byte(0x01) << 6
	FirstByteLongCallArityMask = byte(0x0f) << 2
	Uint16LongCallCodeMask     = ^(uint16(FirstByteDataMask|FirstByteLongCallMask|FirstByteLongCallArityMask) << 8)
)

func (f *parsedFormula) binaryFromParsedFormula(lib LibraryAccess, w io.Writer) (int, error) {
	if len(f.params) == 0 {
		// write inline data
		n, err := strconv.Atoi(f.sym)
		if err == nil {
			// it is a number
			if n < 0 || n >= 256 {
				return 0, fmt.Errorf("integer constant value not uint8: %s", f.sym)
			}
			// it is a 1 byte value
			if _, err = w.Write([]byte{FirstByteDataMask | byte(1), byte(n)}); err != nil {
				return 0, err
			}
			return 0, nil
		}
		if strings.HasPrefix(f.sym, "0x") {
			// it is hexadecimal constant
			b, err := hex.DecodeString(f.sym[2:])
			if err != nil {
				return 0, fmt.Errorf("%v: '%s'", err, f.sym)
			}
			if len(b) > 127 {
				return 0, fmt.Errorf("hexadecimal constant longer than 127 bytes: '%s'", f.sym)
			}
			if _, err = w.Write([]byte{FirstByteDataMask | byte(len(b))}); err != nil {
				return 0, err
			}
			if _, err = w.Write(b); err != nil {
				return 0, err
			}
			return 0, nil
		}
		if strings.HasPrefix(f.sym, "u16/") {
			// it is u16 constant big endian
			n, err = strconv.Atoi(strings.TrimPrefix(f.sym, "u16/"))
			if err != nil {
				return 0, fmt.Errorf("%v: '%s'", err, f.sym)
			}
			if n < 0 || n > math.MaxUint16 {
				return 0, fmt.Errorf("wrong u16 constant: '%s'", f.sym)
			}
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(n))
			if _, err = w.Write([]byte{FirstByteDataMask | byte(2)}); err != nil {
				return 0, err
			}
			if _, err = w.Write(b); err != nil {
				return 0, err
			}
			return 0, nil
		}
		if strings.HasPrefix(f.sym, "u32/") {
			// it is u16 constant big endian
			n, err = strconv.Atoi(strings.TrimPrefix(f.sym, "u32/"))
			if err != nil {
				return 0, fmt.Errorf("%v: '%s'", err, f.sym)
			}
			if n < 0 || n > math.MaxUint32 {
				return 0, fmt.Errorf("wrong u32 constant: '%s'", f.sym)
			}
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, uint32(n))
			if _, err = w.Write([]byte{FirstByteDataMask | byte(4)}); err != nil {
				return 0, err
			}
			if _, err = w.Write(b); err != nil {
				return 0, err
			}
			return 0, nil
		}
		if strings.HasPrefix(f.sym, "u64/") {
			// it is u16 constant big endian
			un, err := strconv.ParseUint(strings.TrimPrefix(f.sym, "u64/"), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("%v: '%s'", err, f.sym)
			}
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, un)
			if _, err = w.Write([]byte{FirstByteDataMask | byte(8)}); err != nil {
				return 0, err
			}
			if _, err = w.Write(b); err != nil {
				return 0, err
			}
			return 0, nil
		}
		// TODO other types of literals
	}
	// either has arguments or not literal
	// try if it is a short call
	fi, err := lib.FunctionByName(f.sym, len(f.params))
	if err != nil {
		return 0, err
	}
	var callBytes []byte
	numArgs := 0

	if fi.IsShort {
		if strings.HasPrefix(fi.Sym, "$") {
			n, err := strconv.Atoi(fi.Sym[1:])
			if err != nil {
				return 0, err
			}
			if n < 0 || n > MaxParameters {
				return 0, fmt.Errorf("wrong argument reference '%s'", fi.Sym)
			}
			if numArgs < n+1 {
				numArgs = n + 1
			}
		}
		callBytes = []byte{byte(fi.FunCode)}
	} else {
		firstByte := FirstByteLongCallMask | (byte(len(f.params)) << 2)
		u16 := (uint16(firstByte) << 8) | fi.FunCode
		callBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(callBytes, u16)
	}
	// write call bytes
	if _, err = w.Write(callBytes); err != nil {
		return 0, err
	}
	// generate code for call parameters
	for _, ff := range f.params {
		n, err := ff.binaryFromParsedFormula(lib, w)
		if err != nil {
			return 0, err
		}
		if n > numArgs {
			numArgs = n
		}
	}
	return numArgs, nil
}

func FormulaSourceToBinary(lib LibraryAccess, formulaSource string) ([]byte, int, error) {
	f, err := parseFormula(formulaSource)
	if err != nil {
		return nil, 0, err
	}

	var buf bytes.Buffer
	numArgs, err := f.binaryFromParsedFormula(lib, &buf)
	if err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), numArgs, nil
}

func FormulaTreeFromBinary(lib LibraryAccess, code []byte) (*FormulaTree, error) {
	ret, remaining, err := formulaTreeFromBinary(lib, code)
	if err != nil {
		return nil, err
	}
	if len(remaining) != 0 {
		return nil, fmt.Errorf("not all bytes have been consumed")
	}
	return ret, nil
}

func formulaTreeFromBinary(lib LibraryAccess, code []byte) (*FormulaTree, []byte, error) {
	if len(code) == 0 {
		return nil, nil, io.EOF
	}
	if code[0]&FirstByteDataMask != 0 {
		// it is data
		size := int(code[0] & FirstByteDataLenMask)
		if len(code) < size+1 {
			return nil, nil, io.EOF
		}
		return DataFormulas(code[1 : 1+size])[0], code[1+size:], nil
	}
	// function call expected
	ret := &FormulaTree{
		Args:     make([]*FormulaTree, 0),
		EvalFunc: nil,
	}
	var evalFun EvalFunction
	var numParams, arity int
	var err error

	if code[0]&FirstByteLongCallMask == 0 {
		// short call
		evalFun, arity, err = lib.FunctionByCode(uint16(code[0]))
		if err != nil {
			return nil, nil, err
		}
		code = code[1:]
	} else {
		// long call
		if len(code) < 2 {
			return nil, nil, io.EOF
		}
		arity = int((code[0] & FirstByteLongCallArityMask) >> 2)
		t := binary.BigEndian.Uint16(code[:2])
		idx := t & Uint16LongCallCodeMask
		evalFun, numParams, err = lib.FunctionByCode(idx)
		if err != nil {
			return nil, nil, err
		}
		if numParams > 0 && numParams != arity {
			return nil, nil, fmt.Errorf("wrong number of call params")
		}
		code = code[2:]
	}

	// collect call Args
	var p *FormulaTree
	for i := 0; i < arity; i++ {
		p, code, err = formulaTreeFromBinary(lib, code)
		if err != nil {
			return nil, nil, err
		}
		ret.Args = append(ret.Args, p)
	}
	ret.EvalFunc = evalFun
	return ret, code, nil
}

func CompileFormula(lib LibraryAccess, formulaSource string) (*FormulaTree, int, []byte, error) {
	code, numParams, err := FormulaSourceToBinary(lib, stripSpaces(formulaSource))
	if err != nil {
		return nil, 0, nil, err
	}
	ret, err := FormulaTreeFromBinary(lib, code)
	if err != nil {
		return nil, 0, nil, err
	}
	return ret, numParams, code, nil
}

func DataFormulas(data ...[]byte) []*FormulaTree {
	ret := make([]*FormulaTree, len(data))
	for i, d := range data {
		d1 := d
		ret[i] = &FormulaTree{
			EvalFunc: func(_ EvalContext) []byte {
				return d1
			},
		}
	}
	return ret
}
