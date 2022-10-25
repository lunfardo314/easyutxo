package ledger

const TimeLockConstraintSource = `

// enforces output can be unlocked only after specified time
func timeLock: or(
	and( isProducedBranch(@), equal(len8(selfConstraintData), 4) ), // must be 4 bytes long
	and( isConsumedBranch(@), lessThan(selfConstraintData, txTimestampBytes) ) // transaction must be strongly after timelock
)
`
