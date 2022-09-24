package easyfl

type evalArgs []*FormulaTree

type RunContext struct {
	evalStack     []evalArgs
	evalStackNext int
	callStack     []evalArgs
	callStackTop  int
	globalCtx     interface{}
}

const MaxStack = 30

func NewRunContext(glb interface{}) *RunContext {
	return &RunContext{
		evalStack: make([]evalArgs, 30),
		callStack: make([]evalArgs, 30),
		globalCtx: glb,
	}
}

func (ctx *RunContext) pushEvalArgs(args evalArgs) {
	ctx.evalStack[ctx.evalStackNext] = args
	ctx.evalStackNext++
}

func (ctx *RunContext) popEvalArgs() {
	ctx.evalStackNext--
	ctx.evalStack[ctx.evalStackNext] = nil
}

func (ctx *RunContext) Arity() byte {
	return byte(len(ctx.evalStack[ctx.evalStackNext-1]))
}

func (ctx *RunContext) Arg(n byte) []byte {
	return ctx.Eval(ctx.evalStack[ctx.evalStackNext-1][n])
}

func (ctx *RunContext) Eval(f *FormulaTree) []byte {
	ctx.pushEvalArgs(f.Args)
	defer ctx.popEvalArgs()

	return f.EvalFunc(ctx)
}

func (ctx *RunContext) Global() interface{} {
	return ctx.globalCtx
}
