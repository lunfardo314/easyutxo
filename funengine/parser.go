package funengine

import (
	"bufio"
	"bytes"
	"fmt"
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
	paramTypes []string
	returnType string
	varParams  bool
	bodySource string
	formula    *formula
}

func parseDefinitions(s string) ([]*funDef, error) {
	lines := splitLinesStripComments(s)
	ret, err := parseDefs(lines)
	if err != nil {
		return nil, err
	}
	for i, fd := range ret {
		fmt.Printf("%d: '%s'\n    paramTypes: %v\n    bodySource: '%s'\n", i, fd.sym, fd.paramTypes, fd.bodySource)
	}
	for _, fd := range ret {
		fd.formula, err = parseFormula(fd.bodySource)
		if err != nil {
			return nil, err
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
			current = &funDef{
				paramTypes: make([]string, 0),
			}
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
	return fd.parseParamAndReturnTypes(paramStr, lineno)
}

// B - one byte
// S - slice of bytes
// B... - many bytes
// S... - many slices
func (fd *funDef) parseParamAndReturnTypes(s string, lineno int) error {
	paramTypes, returnType, found := strings.Cut(s, "->")
	if !found {
		return fmt.Errorf("'->' expected @ line %d", lineno)
	}
	switch returnType {
	case "B", "S":
		fd.returnType = returnType
	default:
		return fmt.Errorf("wrong return type '%s' @ line %d", returnType, lineno)
	}
	if len(paramTypes) > 0 {
		if !strings.Contains(paramTypes, ",") {
			switch s {
			case "B...":
				fd.paramTypes = append(fd.paramTypes, "B")
				fd.varParams = true
			case "S...":
				fd.paramTypes = append(fd.paramTypes, "S")
				fd.varParams = true
			case "B", "S":
				fd.paramTypes = append(fd.paramTypes, paramTypes)
			default:
				return fmt.Errorf("argument type '%s' not supported @ line %d", s, lineno)
			}
			return nil
		}
		spl := strings.Split(paramTypes, ",")
		for _, p := range spl {
			switch p {
			case "B", "S":
			default:
				return fmt.Errorf("argument type '%s' not supported @ line %d", p, lineno)
			}
			fd.paramTypes = append(fd.paramTypes, p)
		}
	}
	return nil
}

func (fd *funDef) resolveAndAddToLibrary(lib libraryFun) error {
	if _, already := lib[fd.sym]; already {
		return fmt.Errorf("repeating function name '%s'", fd.sym)
	}
	returnType, err := fd.formula.validate(lib)
	if err != nil {
		return err
	}
	if fd.returnType != returnType {
		return fmt.Errorf("return type dpes not match the type of the formula '%s'", fd.sym)

	}
	return nil
}

func (f *formula) validate(lib libraryFun) (string, error) {
	if len(f.params) == 0 {
		if checkLiteral(f.sym) {
			return "", nil
		}
		fd, found := lib[f.sym]
		if !found {
			return "", fmt.Errorf("can't resolve name '%s'", f.sym)
		}
		return fd.returnType, nil
	}
	fd, found := lib[f.sym]
	if !found {
		return "", fmt.Errorf("can't resolve name '%s'", f.sym)
	}
	// check parameters
	return fd.returnType, nil
}

func checkLiteral(s string) bool {
	i, err := strconv.Atoi(s)
	if err != nil || i < 0 || i > 255 {
		return false
	}
	return true
}

// call ::= sym(call1, .., callN)

func parseFormula(s string) (*formula, error) {
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
