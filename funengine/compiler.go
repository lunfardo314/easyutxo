package funengine

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type formula struct {
	sym    string
	params []*formula
}

type funDef struct {
	sym        string
	funCode    uint16
	numParams  int // -1 if variable params, only for embedded
	bodySource string
	formula    *formula
	code       []byte
}

func parseDefinitions(s string) ([]*funDef, error) {
	lines := splitLinesStripComments(s)
	ret, err := parseDefs(lines)
	if err != nil {
		return nil, err
	}
	for i, fd := range ret {
		fmt.Printf("%d: '%s'\n    callArity: %d\n    bodySource: '%s'\n", i, fd.sym, fd.numParams, fd.bodySource)
	}
	for _, fd := range ret {
		fd.formula, err = fd.parseFormula(fd.bodySource)
		if err != nil {
			return nil, err
		}
		if len(fd.formula.params) > 15 {
			return nil, fmt.Errorf("too many call parameters in call to '%s': '%s'", fd.sym, fd.bodySource)
		}
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

func parseDefs(lines []string) ([]*funDef, error) {
	ret := make([]*funDef, 0)
	var current *funDef
	for lineno, line := range lines {
		if strings.HasPrefix(line, "def ") {
			if current != nil {
				current.bodySource = stripSpaces(current.bodySource)
				ret = append(ret, current)
			}
			current = &funDef{}
			signature, body, foundEq := strings.Cut(strings.TrimPrefix(line, "def "), "=")
			if !foundEq {
				return nil, fmt.Errorf("'=' expectected @ line %d", lineno)
			}
			if err := current.parseSignature(stripSpaces(signature), lineno); err != nil {
				return nil, err
			}
			current.bodySource += body
		} else {
			if len(stripSpaces(line)) == 0 {
				continue
			}
			if current == nil {
				return nil, fmt.Errorf("unexpectected symbols @ line %d", lineno)
			}
			current.bodySource += line
		}
	}
	if current != nil {
		current.bodySource = stripSpaces(current.bodySource)
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

func (fd *funDef) parseSignature(s string, lineno int) error {
	name, rest, found := strings.Cut(s, "(")
	fd.sym = name
	if !found {
		return fmt.Errorf("argument/return types expected @ line %d", lineno)
	}
	paramStr, rest, found := strings.Cut(rest, ")")
	if !found || len(rest) != 0 {
		return fmt.Errorf("closing ')' expected @ line %d", lineno)
	}
	if paramStr == "..." {
		fd.numParams = -1
		return nil
	}
	n, err := strconv.Atoi(paramStr)
	if err != nil || n < 0 || n > 15 {
		return fmt.Errorf("number of parameters must be '...' or number from 0 to 15 @ line %d", lineno)
	}
	fd.numParams = n
	return nil
}

func (fd *funDef) parseFormula(s string) (*formula, error) {
	name, rest, foundOpen := strings.Cut(s, "(")
	f := &formula{
		sym:    name,
		params: make([]*formula, 0),
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
		ff, err := fd.parseFormula(call)
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
	MaxLongCallCode            = int(Uint16LongCallCodeMask)
	MaxNumShortCall            = 64
)

func (f *formula) genCode(numArgs int, lib map[string]*funDef, w io.Writer) error {
	if len(f.params) == 0 {
		// write inline data
		n, err := strconv.Atoi(f.sym)
		if err == nil {
			// it is a number
			if n < 0 || n >= 256 {
				return fmt.Errorf("integer constant value not uint8: %s", f.sym)
			}
			// it is a 1 byte value
			if _, err = w.Write([]byte{FirstByteDataMask | byte(1), byte(n)}); err != nil {
				return err
			}
			return nil
		}
		if strings.HasPrefix(f.sym, "0x") {
			// it is hexadecimal constant
			b, err := hex.DecodeString(f.sym[2:])
			if err != nil {
				return fmt.Errorf("%v: '%s'", err, f.sym)
			}
			if len(b) > 127 {
				return fmt.Errorf("hexadecimal constant longer than 127 bytes: '%s'", f.sym)
			}
			if _, err = w.Write([]byte{FirstByteDataMask | byte(len(b))}); err != nil {
				return err
			}
			if _, err = w.Write(b); err != nil {
				return err
			}
			return nil
		}
		// TODO other types of literals
	}
	// either has arguments or not literal
	// try if it is a short call
	callBytes, shortCall, err := makeShortCallBytes(f.sym, numArgs, len(f.params))
	if err != nil {
		return err
	}
	if !shortCall {
		// not short call, try the long call
		callBytes, err = makeLongCallBytes(lib, f.sym, len(f.params))
		if err != nil {
			return err
		}
	}
	// generate code for call parameters
	for _, ff := range f.params {
		if err = ff.genCode(numArgs, lib, w); err != nil {
			return err
		}
	}
	// write call bytes
	if _, err = w.Write(callBytes); err != nil {
		return err
	}
	return nil
}

func makeShortCallBytes(sym string, numArgs, numParams int) ([]byte, bool, error) {
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
	if fd.funCode >= MaxNumShortCall {
		panic("too big short call code")
	}
	return []byte{byte(fd.funCode)}, true, nil
}

func makeLongCallBytes(lib map[string]*funDef, sym string, numParams int) ([]byte, error) {
	if numParams > 15 {
		return nil, fmt.Errorf("too many arguments in the call '%s'", sym)
	}
	fd, found := embeddedLongByName[sym]
	if !found {
		if fd, found = lib[sym]; !found {
			return nil, fmt.Errorf("can't resolve symbol '%s'", sym)
		}
	}
	if fd.numParams >= 0 && fd.numParams != numParams {
		return nil, fmt.Errorf("function '%s' require %d arguments", sym, fd.numParams)
	}
	if int(fd.funCode) > MaxLongCallCode {
		panic("too large function code")
	}
	firstByte := FirstByteLongCallMask | (byte(numParams) << 2)
	u16 := (uint16(firstByte) << 8) | fd.funCode
	ret := make([]byte, 2)
	binary.BigEndian.PutUint16(ret, u16)

	return ret, nil
}
