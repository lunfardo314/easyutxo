package library

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
)

type GeneralScript []byte

func NewGeneralScript(data []byte) GeneralScript {
	return data
}

func (u GeneralScript) Name() string {
	return "GeneralScript"
}

func (u GeneralScript) Bytes() []byte {
	return u
}

func (u GeneralScript) String() string {
	return fmt.Sprintf("GeneralScript(%s)", easyfl.Fmt(u))
}