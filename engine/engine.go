package engine

type Engine struct {
	stack         [][]byte
	tx            ContextAccess
	remainingCode []byte
}

type DEAS []byte

type ContextAccess interface {
	GetElement(deas DEAS) ([]byte, bool)
}

func NewEngine() *Engine {
	return &Engine{
		stack: make([][]byte, 0),
	}
}

// Run executes the script. If it returns, script is successful.
// If it panics, transaction is invalid
func (e *Engine) Run(code []byte, tx ContextAccess) {
	e.stack = e.stack[:0]
	e.tx = tx
	e.remainingCode = code
	for e.run1Cycle() {
	}
}

func (e *Engine) run1Cycle() bool {
	var instrRunner instructionRunner
	instrRunner, e.remainingCode = parseInstruction(e.remainingCode)
	return instrRunner(e.tx)
}
