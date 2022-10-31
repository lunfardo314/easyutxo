package library

//
//import (
//	"bytes"
//	"encoding/hex"
//	"fmt"
//
//	"github.com/iotaledger/trie.go/common"
//	"github.com/lunfardo314/easyfl"
//)
//
//type Sender struct{
//	lock            Lock
//	referencedInput byte
//}
//
//const(
//	senderName = "sender"
//	senderTemplate = senderName+
//)
//func NewSender(lock Lock, referencedInput byte) (*Sender, error){
//	if lock.Name() != addressED25519Name{
//		return nil, fmt.Errorf("only addressED25519 sender is suppoted")
//	}
//	return &Sender{
//		lock:            lock,
//		referencedInput: referencedInput,
//	}, nil
//}
//
//
//func (s *Sender) Name() string {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (s *Sender) Bytes() []byte {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (s *Sender) String() string {
//	//TODO implement me
//	panic("implement me")
//}
//
//func SenderConstraintSource(lock []byte, referencedInput byte) string {
//	return fmt.Sprintf("sender(0x%s, %d)", hex.EncodeToString(lock), referencedInput)
//}
//
//func SenderConstraintBin(lock []byte, referencedInput byte) []byte {
//	return mustBinFromSource(SenderConstraintSource(lock, referencedInput))
//}
//
//func initSenderConstraint() {
//	easyfl.MustExtendMany(senderSource)
//
//	addr := AddressED25519Null()
//	example := SenderConstraintBin(addr, 0)
//	sym, prefix, args, err := easyfl.ParseBinaryOneLevel(example, 2)
//	easyfl.AssertNoError(err)
//	common.Assert(sym == "sender" && bytes.Equal(easyfl.StripDataPrefix(args[0]), addr), "inconsistency in 'sender'")
//	registerConstraint("sender", prefix)
//}
//
//// SenderFromConstraint extracts sender address ($0) from the sender script
//func SenderFromConstraint(data []byte) ([]byte, bool) {
//	sym, _, args, err := easyfl.ParseBinaryOneLevel(data, 2)
//	if err != nil {
//		return nil, false
//	}
//	if sym != "sender" {
//		return nil, false
//	}
//	return easyfl.StripDataPrefix(args[0]), true
//}

const senderSource = `

// $0 - lock constraint which is sender
// $1 - referenced consumed output index
func sender: or(
	isConsumedBranch(@),
	and(
		isProducedBranch(@),
		equal($0, consumedLockByOutputIndex($1))
	)
)
`
