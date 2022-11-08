package txbuilder

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/state"
)

func ValidationContextToString(v *state.ValidationContext) string {
	txid := v.TransactionID()
	ret := fmt.Sprintf("TransactionID: %s\n", txid.String())
	tsBin, ts := v.TimestampData()
	ret += fmt.Sprintf("Timestamp: %s (%d)\n", easyfl.Fmt(tsBin), ts)

	ret += "inputs: \n"
	for i := byte(0); int(i) < v.NumInputs(); i++ {
		oid := v.InputID(i)
		ret += fmt.Sprintf("  #%d: %s\n", i, oid.String())
		odata := v.ConsumedOutputData(i)
		o, err := OutputFromBytes(odata)
		if err != nil {
			ret += fmt.Sprintf("     failed to parse output: %v\n", err)
		} else {
			ret += o.ToString("     ")
		}
		unlockBin := v.UnlockData(i)
		arr := lazyslice.ArrayFromBytes(unlockBin)
		ret += fmt.Sprintf("     Unlock data: %s\n", arr.ParsedString())
	}
	ret += "outputs: \n"
	for i := byte(0); int(i) < v.NumProducedOutputs(); i++ {
		ret += fmt.Sprintf("  #%d:\n", i)
		odata := v.ProducedOutputData(i)
		o, err := OutputFromBytes(odata)
		if err != nil {
			ret += fmt.Sprintf("     failed to parse output: %v\n", err)
		} else {
			ret += o.ToString("     ")
		}
	}
	return ret
}
