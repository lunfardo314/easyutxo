package constraint

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
)

type UnknownConstraint []byte

func NewUnknownConstraint(data []byte) UnknownConstraint {
	return data
}

func (u UnknownConstraint) Name() string {
	return "unknown_constraint"
}

func (u UnknownConstraint) Bytes() []byte {
	return u
}

func (u UnknownConstraint) String() string {
	return fmt.Sprintf("unknown_constraint(%s)", easyfl.Fmt(u))
}
