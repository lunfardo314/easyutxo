package library

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/trie.go/common"
	"github.com/lunfardo314/easyfl"
)

type (
	Constraint interface {
		Name() string
		Bytes() []byte
		String() string
	}

	AccountID []byte

	Accountable interface {
		Constraint
		AccountID() AccountID
	}

	Lock interface {
		Constraint
		IndexableTags() []Accountable
	}

	Parser func([]byte) (Constraint, error)

	constraintRecord struct {
		name   string
		prefix []byte
		parser Parser
	}
)

var (
	constraintByPrefix = make(map[string]*constraintRecord)
	constraintNames    = make(map[string]struct{})
)

func registerConstraint(name string, prefix []byte, parser Parser) {
	_, already := constraintNames[name]
	easyfl.Assert(!already, "repeating constraint name '%s'", name)
	_, already = constraintByPrefix[string(prefix)]
	easyfl.Assert(!already, "repeating constraint prefix %s with name '%s'", easyfl.Fmt(prefix), name)
	easyfl.Assert(0 < len(prefix) && len(prefix) <= 2, "wrong constraint prefix %s, name: %s", easyfl.Fmt(prefix), name)
	constraintByPrefix[string(prefix)] = &constraintRecord{
		name:   name,
		prefix: easyfl.Concat(prefix),
		parser: parser,
	}
	constraintNames[name] = struct{}{}
}

func NameByPrefix(prefix []byte) (string, bool) {
	if ret, found := constraintByPrefix[string(prefix)]; found {
		return ret.name, true
	}
	return "", false
}

func parserByPrefix(prefix []byte) (Parser, bool) {
	if ret, found := constraintByPrefix[string(prefix)]; found {
		return ret.parser, true
	}
	return nil, false
}

func mustBinFromSource(src string) []byte {
	_, _, binCode, err := easyfl.CompileExpression(src)
	easyfl.AssertNoError(err)
	return binCode
}

func Equal(l1, l2 Constraint) bool {
	if common.IsNil(l1) != common.IsNil(l2) {
		return false
	}
	if common.IsNil(l1) || common.IsNil(l2) {
		return false
	}
	return bytes.Equal(l1.Bytes(), l2.Bytes())
}

func FromBytes(data []byte) (Constraint, error) {
	prefix, err := easyfl.ParseCallPrefixFromBinary(data)
	if err != nil {
		return nil, err
	}
	parser, ok := parserByPrefix(prefix)
	if ok {
		return parser(data)
	}
	return NewGeneralScript(data), nil
}

func (acc AccountID) Bytes() []byte {
	return acc
}

func LockFromBytes(data []byte) (Lock, error) {
	prefix, err := easyfl.ParseCallPrefixFromBinary(data)
	if err != nil {
		return nil, err
	}
	name, ok := NameByPrefix(prefix)
	if !ok {
		return nil, fmt.Errorf("unknown constraint with prefix '%s'", easyfl.Fmt(prefix))
	}
	switch name {
	case addressED25519Name:
		return AddressED25519FromBytes(data)
	case deadlineLockName:
		return DeadlineLockFromBytes(data)
	}
	return nil, fmt.Errorf("not a lock constraint '%s'", name)
}

func AccountableFromBytes(data []byte) (Accountable, error) {
	prefix, err := easyfl.ParseCallPrefixFromBinary(data)
	if err != nil {
		return nil, err
	}
	name, ok := NameByPrefix(prefix)
	if !ok {
		return nil, fmt.Errorf("unknown constraint with prefix '%s'", easyfl.Fmt(prefix))
	}
	switch name {
	case addressED25519Name:
		return AddressED25519FromBytes(data)
	}
	return nil, fmt.Errorf("not a indexable constraint '%s'", name)
}

func UnlockParamsByReference(ref byte) []byte {
	return []byte{ref}
}
