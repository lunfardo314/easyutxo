package engine

import (
	"bytes"
	"fmt"
	"math"

	"github.com/lunfardo314/easyutxo"
)

type Program struct {
	opcodes      Opcodes
	resolve      map[string]int
	instructions []instruction
	label        bool
}

type instruction struct {
	bytes      []byte
	shortJumps map[byte]string
	longJumps  map[byte]string
}

func NewProgram(opcodes Opcodes) *Program {
	return &Program{
		opcodes:      opcodes,
		resolve:      make(map[string]int),
		instructions: make([]instruction, 0),
	}
}

// OP starts new instruction
func (p *Program) OP(c OpCode) *Program {
	if err := p.opcodes.ValidateOpcode(c); err != nil {
		panic(fmt.Errorf("error @ instruction #%d: %v", len(p.instructions), err))
	}
	p.label = false
	p.instructions = append(p.instructions, instruction{
		bytes:      make([]byte, 0, 10),
		shortJumps: make(map[byte]string),
		longJumps:  make(map[byte]string),
	})
	p.append(c.Bytes()...)
	return p
}

func (p *Program) B(b ...byte) *Program {
	if p.label {
		panic("label in wrong position")
	}
	p.append(b...)
	return p
}

// TargetShort jump short
func (p *Program) TargetShort(label string) *Program {
	if p.label {
		panic("label in wrong position")
	}
	p.markToResolveShort(label)
	p.append(0)
	return p
}

func (p *Program) TargetLong(label string) *Program {
	if p.label {
		panic("label in wrong position")
	}
	p.markToResolveLong(label)
	p.append(0, 0)
	return p
}

func (p *Program) Label(label string) {
	if _, already := p.resolve[label]; already {
		panic("repeating label name")
	}
	totalBytes := 0
	for i := range p.instructions {
		totalBytes += len(p.instructions[i].bytes)
	}
	p.resolve[label] = totalBytes
	p.label = true
}

func (p *Program) Compile() ([]byte, error) {
	var buf bytes.Buffer
	currentPosition := 0
	for currentInstruction := range p.instructions {
		// resolve the jump addresses
		if err := p.resolveShort(currentInstruction, currentPosition); err != nil {
			return nil, err
		}
		if err := p.resolveLong(currentInstruction, currentPosition); err != nil {
			return nil, err
		}
		buf.Write(p.instructions[currentInstruction].bytes)
		currentPosition += len(p.instructions[currentInstruction].bytes)
	}
	return buf.Bytes(), nil
}

func (p *Program) MustCompile() []byte {
	ret, err := p.Compile()
	if err != nil {
		panic(err)
	}
	return ret
}

func (p *Program) resolveShort(currentInstruction, instructionAddress int) error {
	for pos, label := range p.instructions[currentInstruction].shortJumps {
		targetPosition, ok := p.resolve[label]
		if !ok {
			return fmt.Errorf("cannot resolve label '%s', instruction #%d@%d", label, currentInstruction, pos)
		}
		if targetPosition < instructionAddress+len(p.instructions[currentInstruction].bytes) {
			// jumps only forward. Loops prevented on purpose
			return fmt.Errorf("cannot jump back: label '%s',  instruction #%d@%d", label, currentInstruction, pos)
		}
		// relative jump forward is counted from the beginning of the current instruction
		relativeJump := targetPosition - instructionAddress
		if relativeJump > math.MaxUint8 {
			return fmt.Errorf("short jump can be bigger than 255 positions heaad: label '%s', instruction #%d@%d", label, currentInstruction, pos)
		}
		p.instructions[currentInstruction].bytes[pos] = byte(relativeJump)
	}
	return nil
}

func (p *Program) resolveLong(currentInstruction, instructionAddress int) error {
	for pos, label := range p.instructions[currentInstruction].longJumps {
		targetPosition, ok := p.resolve[label]
		if !ok {
			return fmt.Errorf("cannot resolve label '%s', instruction #%d@%d", label, currentInstruction, pos)
		}
		if targetPosition < instructionAddress+len(p.instructions[currentInstruction].bytes) {
			// jumps only forward. Loops prevented on purpose
			return fmt.Errorf("cannot jump back: label '%s',  instruction #%d@%d", label, currentInstruction, pos)
		}
		// relative jump forward is counted from the beginning of the current instruction
		relativeJump := targetPosition - instructionAddress
		copy(p.instructions[currentInstruction].bytes[pos:pos+2], easyutxo.EncodeInteger(uint16(relativeJump)))
	}
	return nil
}

func (p *Program) append(b ...byte) {
	last := len(p.instructions) - 1
	p.instructions[last].bytes = append(p.instructions[last].bytes, b...)
}

func (p *Program) markToResolveShort(label string) {
	last := len(p.instructions) - 1
	p.instructions[last].shortJumps[byte(len(p.instructions[last].bytes))] = label
}

func (p *Program) markToResolveLong(label string) {
	last := len(p.instructions) - 1
	p.instructions[last].longJumps[byte(len(p.instructions[last].bytes))] = label
}
