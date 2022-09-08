package opcodes

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lunfardo314/easyutxo"
	"github.com/lunfardo314/easyutxo/engine"
)

func GenProgram(fun func(p *engine.Program)) ([]byte, error) {
	p := engine.NewProgram(All)
	var compileErr error
	var ret []byte
	err := easyutxo.CatchPanic(func() {
		fun(p)
		ret, compileErr = p.Assemble()
	})
	if err != nil {
		return nil, err
	}
	if compileErr != nil {
		return nil, compileErr
	}
	return ret, nil
}

func MustGenProgram(fun func(p *engine.Program)) []byte {
	ret, err := GenProgram(fun)
	if err != nil {
		panic(err)
	}
	return ret
}

func CompileSource(sourceCode string) ([]byte, error) {
	prog := engine.NewProgram(All)
	lines := splitLines(sourceCode)
	for lineno, line := range lines {
		instr, _, _ := strings.Cut(line, ";")
		l := strings.TrimSpace(instr)
		if len(l) == 0 {
			continue
		}
		instr = strings.TrimSpace(instr)
		if strings.HasPrefix(instr, ">") {
			instr = strings.TrimPrefix(instr, ">")
			instr = strings.TrimSpace(instr)
			fmt.Printf("%2d: label: '%s'\n", lineno, instr)
			prog.Label(instr)
		} else {
			opcode, params, _ := strings.Cut(instr, " ")
			opcode = strings.TrimSpace(opcode)
			dscr, found := allSymLookup[opcode]
			if !found {
				return nil, fmt.Errorf("wrong opcode '%s' @ line %d: '%s'", opcode, lineno, strings.TrimSpace(line))
			}
			prog.Opcode(dscr.opcode)
			params = strings.TrimSpace(params)
			par := strings.Split(params, ",")
			fmt.Printf("%2d: opcode: '%s', params: %v\n", lineno, opcode, par)
			parCleaned := par[:0]
			for _, p := range par {
				p1 := strings.TrimSpace(p)
				if len(p1) > 0 {
					parCleaned = append(parCleaned, p1)
				}
			}
			if err := assembleParams(prog, parCleaned, dscr.params); err != nil {
				return nil, fmt.Errorf("%v @ line %d: '%s'", err, lineno, strings.TrimSpace(line))
			}
		}
	}
	return prog.Assemble()
}

func MustCompileSource(sourceCode string) []byte {
	ret, err := CompileSource(sourceCode)
	if err != nil {
		panic(err)
	}
	return ret
}

func assembleParams(prog *engine.Program, params []string, templates []paramsTemplateCompiled) error {
	if len(params) != len(templates) {
		return fmt.Errorf("expected %d parameters, got %d", len(templates), len(params))
	}
	for i, p := range params {
		switch templates[i].paramType {
		case paramType8:
			r, err := strconv.Atoi(p)
			if err != nil {
				return err
			}
			if r < 0 || r > math.MaxUint8 {
				return errors.New("must be byte value")
			}
			prog.ParamBytes(byte(r))
		case paramType16:
			r, err := strconv.Atoi(p)
			if err != nil {
				return err
			}
			if r < 0 || r > math.MaxUint16 {
				return errors.New("must be uint16 value")
			}
			prog.ParamBytes(easyutxo.EncodeInteger(uint16(r))...)
		case paramTypeVariable:
			r, err := parseVarLenParameter(p)
			if err != nil {
				return err
			}
			prog.ParamBytes(byte(len(r)))
			prog.ParamBytes(r...)
		case paramTypeShortTarget:
			prog.TargetShort(p)
		case paramTypeLongTarget:
			prog.TargetLong(p)
		default:
			panic("assembleParams: wrong param template")
		}
	}
	return nil
}

func parseVarLenParameter(p string) ([]byte, error) {
	var ret []byte
	var err error
	var n int
	switch {
	case p == "nil":
		ret = nil
	case strings.HasPrefix(p, "hex/"):
		ret, err = hex.DecodeString(strings.TrimPrefix(p, "hex/"))
		if err != nil {
			return nil, err
		}
	case strings.HasPrefix(p, "u8/"):
		n, err = strconv.Atoi(strings.TrimPrefix(p, "u8/"))
		if err != nil {
			return nil, err
		}
		if n < 0 || n > math.MaxUint8 {
			return nil, errors.New("wrong u8 value")
		}
		ret = []byte{byte(n)}
	case strings.HasPrefix(p, "u16/"):
		n, err = strconv.Atoi(strings.TrimPrefix(p, "u16/"))
		if err != nil {
			return nil, err
		}
		if n < 0 || n > math.MaxUint16 {
			return nil, errors.New("wrong u16 value")
		}
		ret = easyutxo.EncodeInteger(uint16(n))
	case strings.HasPrefix(p, "u32/"):
		n, err = strconv.Atoi(strings.TrimPrefix(p, "u32/"))
		if err != nil {
			return nil, err
		}
		if n < 0 || n > math.MaxUint16 {
			return nil, errors.New("wrong u32 value")
		}
		ret = easyutxo.EncodeInteger(uint32(n))
	default:
		// treat it as string
		ret = []byte(p)

	}
	if len(ret) > math.MaxUint8 {
		return nil, errors.New("too long value")
	}
	return ret, nil
}

func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
