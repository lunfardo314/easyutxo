package easyfl

import "fmt"

type evalArgs []*Expression

type RunContext struct {
	evalStack     []evalArgs
	evalStackNext int
	globalCtx     interface{}
}

const MaxStackDepth = 50

func NewRunContext(glb interface{}) *RunContext {
	return &RunContext{
		evalStack: make([]evalArgs, MaxStackDepth),
		globalCtx: glb,
	}
}

func (ctx *RunContext) Arity() byte {
	return byte(len(ctx.evalStack[ctx.evalStackNext-1]))
}

func (ctx *RunContext) Arg(n byte) []byte {
	return ctx.Eval(ctx.evalStack[ctx.evalStackNext-1][n])
}

// evalParam used by $0-$15 functions
func (ctx *RunContext) evalParam(n byte, levelBack int) []byte {
	return ctx.Eval(ctx.evalStack[ctx.evalStackNext-levelBack-1][n])
}

func (ctx *RunContext) Eval(f *Expression) []byte {
	ctx.pushEvalArgs(f.Args)
	defer ctx.popEvalArgs()

	return f.EvalFunc(ctx)
}

func (ctx *RunContext) DataContext() interface{} {
	return ctx.globalCtx
}

func (ctx *RunContext) pushEvalArgs(args evalArgs) {
	ctx.evalStack[ctx.evalStackNext] = args
	ctx.evalStackNext++
}

func (ctx *RunContext) popEvalArgs() {
	ctx.evalStackNext--
	ctx.evalStack[ctx.evalStackNext] = nil
}

func evalExpression(glb interface{}, f *Expression, args evalArgs) []byte {
	ctx := NewRunContext(glb)
	expr := Expression{
		Args: args,
		EvalFunc: func(glb *RunContext) []byte {
			return glb.Eval(f)
		},
	}
	return ctx.Eval(&expr)
}

func callFunction(glb interface{}, f *Expression, args ...[]byte) []byte {
	return evalExpression(glb, f, dataFormulas(args...))
}

func EvalExpression(glb interface{}, source string, args ...[]byte) ([]byte, error) {
	f, requiredNumArgs, _, err := CompileFormula(source)
	if err != nil {
		return nil, err
	}
	if requiredNumArgs != len(args) {
		return nil, fmt.Errorf("required number of parameters is %d, got %d", requiredNumArgs, len(args))
	}
	return callFunction(glb, f, args...), nil
}
