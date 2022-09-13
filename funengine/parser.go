package funengine

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

type formula struct {
	sym    string
	params []interface{} // can be literal or formula
}

type funDef struct {
	name   string
	params []string
	body   string
}

func parse(s string) ([]*formula, error) {
	lines := splitLinesStripComments(s)
	fds, err := consolidatedDefs(lines)
	if err != nil {
		return nil, err
	}
	for i, fd := range fds {
		fmt.Printf("%d: '%s'\n    params: %v\n    body: '%s'\n", i, fd.name, fd.params, fd.body)
	}
	ret := make([]*formula, 0)
	for _, fd := range fds {
		res, err := parseCall(fd.body)
		if err != nil {
			return nil, err
		}
		ret = append(ret, res)
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

func consolidatedDefs(lines []string) ([]*funDef, error) {
	ret := make([]*funDef, 0)
	var current *funDef
	for lineno, line := range lines {
		if strings.HasPrefix(line, "def ") {
			if current != nil {
				current.body = stripSpaces(current.body)
				ret = append(ret, current)
			}
			current = &funDef{
				params: make([]string, 0),
			}
			signature, body, foundEq := strings.Cut(strings.TrimPrefix(line, "def "), "=")
			if !foundEq {
				return nil, fmt.Errorf("'=' expectected @ line %d", lineno)
			}
			if err := current.parseSignature(stripSpaces(signature), lineno); err != nil {
				return nil, err
			}
			current.body += body
		} else {
			if len(stripSpaces(line)) == 0 {
				continue
			}
			if current == nil {
				return nil, fmt.Errorf("unexpectected symbols @ line %d", lineno)
			}
			current.body += line
		}
	}
	if current != nil {
		current.body = stripSpaces(current.body)
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
	fd.name = name
	if !found {
		if len(name) == 0 {
			return fmt.Errorf("empty name not allowed @ line %d", lineno)
		}
		return nil
	}
	paramStr, rest, found := strings.Cut(rest, ")")
	if !found || len(rest) != 0 {
		return fmt.Errorf("closing ')' expected @ line %d", lineno)
	}
	return fd.parseParams(paramStr, lineno)
}

func (fd *funDef) parseParams(s string, lineno int) error {
	if len(s) == 0 {
		return nil
	}
	spl := strings.Split(s, ",")
	for _, p := range spl {
		switch p {
		case "S", "V":
		default:
			return fmt.Errorf("argument type '%s' not supported @ line %d", p, lineno)
		}
		fd.params = append(fd.params, p)
	}
	return nil
}

// call ::= name(call1, .., callN)

func parseCall(s string) (*formula, error) {
	name, rest, foundOpen := strings.Cut(s, "(")
	f := &formula{
		sym:    name,
		params: make([]interface{}, 0),
	}
	if !foundOpen {
		if strings.Contains(rest, ")") {
			return nil, fmt.Errorf("unexpected ')': '%s'", s)
		}
		if strings.Contains(name, ",") {
			for _, a := range strings.Split(name, ",") {
				f.params = append(f.params, a)
			}
		}
		return f, nil
	}
	spl, err := splitArgs(rest)
	if err != nil {
		return nil, err
	}
	for _, call := range spl {
		ff, err := parseCall(call)
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
