package easyfl

import (
	"fmt"
)

type evalArgs []*Expression

type RunContext struct {
	globalCtx interface{}
	params    []*Expression
	depth     int
	expr      *Expression
	//evalStack     []*Expression
	//evalStackNext int
}

const MaxStackDepth = 50

func NewRunContext(glb interface{}, expr *Expression, params []*Expression, depth int) *RunContext {
	return &RunContext{
		expr:      expr,
		globalCtx: glb,
		params:    params,
		depth:     depth,
		//evalStack: make([]*Expression, MaxStackDepth),
	}
}

func (ctx *RunContext) Arity() byte {
	return byte(len(ctx.expr.Args))
	//return byte(len(ctx.evalStack[ctx.evalStackNext-1].Args))
}

func (ctx *RunContext) Arg(n byte) []byte {
	if traceYN {
		fmt.Printf("Arg(%d) depth: %d -- IN\n",
			n, ctx.depth)
	}

	//ret := evalExpression(ctx.globalCtx, ctx.evalStack[ctx.evalStackNext-1][n], nil)
	ret := ctx.Eval(ctx.expr.Args[n])
	if traceYN {
		fmt.Printf("Arg(%d) depth: %d -- OUT ret: %v\n",
			n, ctx.depth, ret)
	}
	return ret
}

// evalParam used by $0-$15 functions
func (ctx *RunContext) evalParam(n byte) []byte {
	if traceYN {
		fmt.Printf("evalParam $%d, depth: %d -- IN\n", n, ctx.depth)
	}
	ret := ctx.Eval(ctx.params[n])
	if traceYN {
		fmt.Printf("evalParam $%d, depth: %d -- OUT, ret: %v\n",
			n, ctx.depth, ret)
	}
	return ret
}

func (ctx *RunContext) Eval(f *Expression) []byte {
	//ctx.push(f)
	//defer ctx.pop()

	if traceYN {
		fmt.Printf("Eval expression. Depth %d\n", ctx.depth)
	}
	ctxNew := NewRunContext(ctx.globalCtx, f, ctx.params, ctx.depth+1)
	return f.EvalFunc(ctxNew)
}

func (ctx *RunContext) DataContext() interface{} {
	return ctx.globalCtx
}

//func (ctx *RunContext) push(f *Expression) {
//	if traceYN {
//		fmt.Printf("push args: %d depth: %d\n", len(f.Args), ctx.depth)
//	}
//	ctx.evalStack[ctx.evalStackNext] = f
//	ctx.evalStackNext++
//}
//
//func (ctx *RunContext) pop() {
//	if traceYN {
//		fmt.Printf("pop depth: %d, stack: %d\n", ctx.depth, ctx.evalStackNext)
//	}
//	ctx.evalStackNext--
//	ctx.evalStack[ctx.evalStackNext] = nil
//}

func evalExpression(glb interface{}, f *Expression, args evalArgs) []byte {
	ctx := NewRunContext(glb, f, args, 0)
	return ctx.Eval(f)
}

func callFunction(glb interface{}, f *Expression, args ...[]byte) []byte {
	argsForData := dataFormulas(args...)
	return evalExpression(glb, f, argsForData)
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
