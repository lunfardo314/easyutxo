package ledger

// MainConstraintSource is always in the #0 block of the output. It checks validity of the timestamp and amount
// The constraint is:
// - byte 0 - constraint code 2
// - bytes 1-4 timestamp bytes, big-endian Unix seconds
// - bytes 5-12 amount bytes, big-endian uint64
const MainConstraintSource = `

// mandatory blocks in each output
func mainConstraintBlockIndex: 0
func lockConstraintBlockIndex: 1
func senderConstraintBlockIndex: 2

// amount is valid if it is 8 bytes-long and not all 0
func amountValid: and(
	equal(len8($0),8),
	not(isZero($0))
)

// timestamp is valid if:
// - for consumed output - must be strongly less than the transaction timestamp
// - for produced output - must be equal to the transaction timestamp
func timestampValid: or(
	and( isProducedBranch(@), equal($0, txTimestampBytes) ),
	and( isConsumedBranch(@), lessThan($0, txTimestampBytes) )
)

func amountBytes: tail(selfConstraintData,4)
func timestampBytes: slice(selfConstraintData,0,3)

// constrain valid if both timestamp and amount are valid 
func amountAndTimestampValid : and(
	timestampValid(timestampBytes),
	amountValid(amountBytes)
)

// checks if all mandatory constraints are not nil
// should not check presence of the main constraint itself
func mandatoryConstraintsPresent: selfSiblingBlock(lockConstraintBlockIndex)

func storageDepositEnough: greaterOrEqualThan(
	amountBytes,
	concat(u32/0, mul16_32(#vbCost16,len16(selfOutputBytes)))
)

func mainConstraint : and(
	mandatoryConstraintsPresent,
	amountAndTimestampValid,
	storageDepositEnough
)
`
