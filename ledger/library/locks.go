package library

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
)

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

func IndexableFromBytes(data []byte) (Accountable, error) {
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
