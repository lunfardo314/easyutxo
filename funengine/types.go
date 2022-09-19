package funengine

const MaxNumShortCall = 64

type funDef struct {
	sym        string
	funCode    uint16
	numParams  int // -1 if variable params, only for embedded
	bodySource string
	formula    *formula
	code       []byte
}

type FunInfo struct {
	Sym        string
	FunCode    uint16
	IsEmbedded bool
	IsShort    bool
	NumParams  int
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
	Exists(sym string) bool
	Resolve(sym string, numParams int) (*FunInfo, error)
}

type RuntimeLibrary interface {
	FunctionByFunCode(funCode uint16, arity int) func()
}
