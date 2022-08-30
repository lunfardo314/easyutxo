package transaction

type LedgerState interface {
	GetUTXO(id *OutputID) (OutputData, bool)
	// GetUTXOsForAddress order non-deterministic
	GetUTXOsForAddress(addr []byte) []OutputData
}

type Ledger interface {
	AddTransaction(*Transaction) error
}
