package easyutxo

type Engine struct {
	stack         [][]byte
	par           *Params
	tx            *Transaction
	remainingCode []byte
}

func NewEngine() *Engine {
	return &Engine{
		stack: make([][]byte, 0),
	}
}

// Run executes the script. If it returns, script is successful.
// If it panics, transaction is invalid
func (e *Engine) Run(code []byte, par *Params, tx *Transaction) {
	e.stack = e.stack[:0]
	e.par = par
	e.tx = tx
	e.remainingCode = code
	for e.run1Cycle() {
	}
}

func (e *Engine) run1Cycle() bool {
	var instrRunner instructionRunner
	instrRunner, e.remainingCode = parseInstruction(e.remainingCode)
	return instrRunner(e.tx, e.par)
}
