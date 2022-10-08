package easyutxo

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func CatchPanicOrError(f func() error) error {
	var err error
	func() {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("%v", r)
			}
		}()
		err = f()
	}()
	return err
}

func Concat(data ...interface{}) []byte {
	var buf bytes.Buffer
	for _, d := range data {
		switch d := d.(type) {
		case byte:
			buf.WriteByte(d)
		case []byte:
			buf.Write(d)
		case interface{ Bytes() []byte }:
			buf.Write(d.Bytes())
		default:
			panic("must be byte or []byte")
		}
	}
	return buf.Bytes()
}

func EmptySlices(s ...[]byte) bool {
	for _, sl := range s {
		if len(sl) != 0 {
			return false
		}
	}
	return true
}

func RequireErrorWith(t *testing.T, err error, s string) {
	require.Error(t, err)
	require.Contains(t, err.Error(), s)
}

func RequirePanicOrErrorWith(t *testing.T, f func() error, s string) {
	RequireErrorWith(t, CatchPanicOrError(f), s)
}

func Assert(cond bool, format string, args ...interface{}) {
	if !cond {
		panic(fmt.Sprintf("assertion failed:: "+format, args...))
	}
}

func AssertNoError(err error) {
	Assert(err == nil, "error: %v", err)
}

func All0(d []byte) bool {
	for _, e := range d {
		if e != 0 {
			return false
		}
	}
	return true
}
