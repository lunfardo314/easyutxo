package txbuilder

import (
	"fmt"

	"github.com/lunfardo314/easyfl"
	"github.com/lunfardo314/easyutxo/lazyslice"
	"github.com/lunfardo314/easyutxo/ledger/state"
	"golang.org/x/crypto/blake2b"
)

func ValidationContextToString(v *state.TransactionContext) string {
	txid := v.TransactionID()
	ret := fmt.Sprintf("\nTransaction. ID: %s, size: %d\n", txid.String(), len(v.TransactionBytes()))
	tsBin, ts := v.TimestampData()
	ret += fmt.Sprintf("Timestamp: %s (%d)\n", easyfl.Fmt(tsBin), ts)
	ret += fmt.Sprintf("Input commitment: %s\n", easyfl.Fmt(v.InputCommitment()))
	sign := v.Signature()
	ret += fmt.Sprintf("Signature: %s\n", easyfl.Fmt(sign))
	if len(sign) == 96 {
		sender := blake2b.Sum256(sign[64:])
		ret += fmt.Sprintf("     ED25519 sender address: %s\n", easyfl.Fmt(sender[:]))
	}

	ret += "Inputs (consumed outputs): \n"
	for i := byte(0); int(i) < v.NumInputs(); i++ {
		oid := v.InputID(i)
		odata := v.ConsumedOutputData(i)
		ret += fmt.Sprintf("  #%d: %s (%d bytes)\n", i, oid.String(), len(odata))
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
	ret += "Outputs (produced): \n"
	for i := byte(0); int(i) < v.NumProducedOutputs(); i++ {
		odata := v.ProducedOutputData(i)
		ret += fmt.Sprintf("  #%d (%d bytes) :\n", i, len(odata))
		o, err := OutputFromBytes(odata)
		if err != nil {
			ret += fmt.Sprintf("     failed to parse output: %v\n", err)
		} else {
			ret += o.ToString("     ")
		}
	}
	return ret
}
