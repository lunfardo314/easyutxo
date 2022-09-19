package easyfl

const (
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

type FunCompiled struct {
	Sym        string
	NumParams  int
	BinaryCode []byte
}

type FunDef struct {
	sym        string
	funCode    uint16
	numParams  int // -1 if variable params, only for embedded
	bodySource string
	formula    *formula
	code       []byte
}

type CompilerLibrary interface {
	ExistsFun(sym string) bool
	Resolve(sym string, numParams int) (*FunInfo, error)
}

type RuntimeLibrary interface {
	FunctionByFunCode(funCode uint16, arity int) func()
}
