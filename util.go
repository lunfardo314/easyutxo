package easyutxo

import (
	"bytes"
	"fmt"
)

func CatchPanic(f func()) (err error) {
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
		f()
	}()
	return err
}

func Concat(data ...[]byte) []byte {
	var buf bytes.Buffer
	for _, d := range data {
		buf.Write(d)
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
