package easyfl

const (
	MaxParameters      = 15
	MaxNumShortCall    = 64
	ExtendedCodeOffset = 256
)

type FunInfo struct {
	Sym        string
	FunCode    uint16
	IsEmbedded bool
	IsShort    bool
	NumParams  int
}

type FunParsed struct {
	Sym        string
	NumParams  int
	SourceCode string
}

type FormulaTree struct {
	args    []*FormulaTree
	evalFun EvalFunction
}

type EvalFunction func(glb interface{}, args []*FormulaTree) []byte

type LibraryAccess interface {
	ExistsFunction(sym string) bool
	FunctionByName(sym string, numParams int) (*FunInfo, error)
	FunctionByCode(funCode uint16) (EvalFunction, int, error)
}