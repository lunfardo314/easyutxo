package ledger

func (v *TransactionContext) Validate() error {
	if err := v.ValidateProducedOutputs(); err != nil {
		return err
	}
	return nil
}
func (v *TransactionContext) ValidateProducedOutputs() error {
	return nil
}
