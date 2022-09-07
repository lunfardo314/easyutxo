package opcodes

import (
	"bufio"
	"fmt"
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
		ret, compileErr = p.Compile()
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
		} else {
			opcode, params, _ := strings.Cut(instr, " ")
			opcode = strings.TrimSpace(opcode)
			params = strings.TrimSpace(params)
			par := strings.Split(params, ",")
			fmt.Printf("%2d: opcode: '%s', params: %v\n", lineno, opcode, par)
		}
	}
	return nil, nil
}

func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
