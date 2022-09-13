package funengine

import (
	"bufio"
	"fmt"
	"strings"
	"unicode"
)

type formula struct {
	funsym string
	params []interface{} // can be literal or formula
}

type literal string

type funDef struct {
	name   string
	params []string
	body   string
}

func parse(s string) error {
	lines := splitLinesStripComments(s)
	fds, err := consolidatedDefs(lines)
	if err != nil {
		return err
	}
	for i, fd := range fds {
		fmt.Printf("%d: %s\n    params: %v\n    body: '%s'\n", i, fd.name, fd.params, fd.body)
	}
	return nil
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
