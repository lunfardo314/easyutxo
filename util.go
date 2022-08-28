package easyutxo

import (
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
