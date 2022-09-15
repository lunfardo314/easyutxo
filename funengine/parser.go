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
		fmt.Printf("%d: '%s'\n    numParams: %d\n    bodySource: '%s'\n", i, fd.sym, fd.numParams, fd.bodySource)
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
