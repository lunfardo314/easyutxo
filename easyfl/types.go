package easyfl

const (
	EmbeddedReservedUntil = 15
	MaxNumEmbeddedShort   = 64
	FirstEmbeddedLongFun  = MaxNumEmbeddedShort
	MaxNumEmbeddedLong    = 256
	FirstExtendedFun      = FirstEmbeddedLongFun + MaxNumEmbeddedLong
	MaxFunCode            = 1023
	MaxNumExtended        = MaxFunCode - FirstExtendedFun

	MaxParameters = 15
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
	SourceCode string
}

type FormulaTree struct {
	Args     []*FormulaTree
	EvalFunc EvalFunction
}

type EvalFunction func(glb *RunContext) []byte

type LibraryAccess interface {
	ExistsFunction(sym string) bool
	FunctionByName(sym string) (*FunInfo, error)
	FunctionByCode(funCode uint16) (EvalFunction, int, error)
}
