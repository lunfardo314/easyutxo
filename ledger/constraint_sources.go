package ledger

// MainConstraint is always in the #0 block of the output. It checks validity of the timestamp and amount
// The constraint is:
// - byte 0 - constraint code 2
// - bytes 1-4 timestamp bytes, big-endian Unix seconds
// - bytes 5-12 amount bytes, big-endian uint64
const MainConstraint = `

// amount is valid if it is 8 bytes-long and not all 0
func amountValid: and(
	equal(len8($0),8),
	not(isZero($0))
)

// timestamp is valid if:
// - for consumed output - must be strongly less tha  transaction timestamp
// - for produced output - must be equal to the transaction timestamp
func outputTimestampValid: or(
	and( isProducedBranch(@), equal($0, txTimestampBytes) ),
	and( isConsumedBranch(@), lessThan($0, txTimestampBytes) )
)

// constrain valid if both timestamp and amount are valid 
func amountAndTimestampValid : and(
	outputTimestampValid(slice($0,0,3)),
	amountValid(tail($0,4))
)

func mainConstraint : amountAndTimestampValid(selfConstraintData)
`

const SigLockConstraint = `

// the function 'selfUnlockedWithReference'' is accessing the transaction context knowing it invocation
// place (output index). Other functions 'selfUnlockBlock', 'selfOutputIndex', 'selfConstraint', 
// 'selfReferencedConstraint' etc are all invocation context specific
// It all and up to embedded functions '@' which gives invocation location and '@Path' which gives data bytes
// for any location inn the transaction specified by any valid path

// 'selfUnlockedWithReference'' specifies validation of the input unlock with the reference
func selfUnlockedWithReference: and(
	equal(len8(selfUnlockBlock), 2),                     // unlock block must be 2 bytes long
	lessThan(byte(selfUnlockBlock,0), selfOutputIndex),  // unlock block must point to another input with strictly 
														 // smaller index	
	equal(selfConstraint, selfReferencedConstraint)      // the referenced constraint bytes must be equal to the
														 // self constrain bytes
)

// 'selfUnlockedWithSigED25519' specifies unlock constraint with the unlock block signature
// the signature must be valid and hash of the public key must be equal to the provided address
func selfUnlockedWithSigED25519: and(
	equal(len8(selfUnlockBlock), 96),                    // unlock block must be 96 bytes long
	validSignatureED25519(
		txEssenceBytes,                        // function 'txEssenceBytes' returns transaction essence btes 
		slice(selfUnlockBlock, 0, 64),         // first 64 bytes is signature
		slice(selfUnlockBlock, 64, 96)         // the rest is public key
	),
	equal(
		selfConstraintData,                    // address in the constraint data must be equal to the has of the  
		addrDataED25519FromPubKey(                 // public key
			slice(selfUnlockBlock, 64, 96)
		)
	)
)

// if it is 'produced' invocation context (constraint invoked in the input), only size of the address is checked
// Otherwise the first will check first condition if it is unlocked by reference, otherwise checks unlocking signature
// Second condition not evaluated if the first is true 

func sigLocED25519: or(
	and(
		isProducedBranch(@), 
		equal( len8(selfConstraintData), 32) 
	),
    and(
		isConsumedBranch(@), 
		or(                                    
			selfUnlockedWithReference,    // if it is unlocked with reference, the signature is not checked
			selfUnlockedWithSigED25519    // otherwise signature is checked
		)
	)
)
`
