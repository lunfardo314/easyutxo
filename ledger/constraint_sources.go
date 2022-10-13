package ledger

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

// if it is 'consumed' invocation context (constraint invoked in the input), only size of the address is checked
// Otherwise the first will check first condition if it is unlocked by reference, otherwise checks unlocking signature
// Second condition not evaluated if the first is true 

func sigLocED25519: if(
	isConsumedBranch(@),
    equal( len8(selfConstraintData), 32 ),
	or(                                    
		selfUnlockedWithReference,    // if it is unlocked with reference, the signature is not checked
		selfUnlockedWithSigED25519    // otherwise signature is checked
	)
)
`
const MainConstraint = `

func tokensValid: and(
	equal(len8($0),8),
	not(isZero($0))
)

func outputTimestampValid: or(
	and( isProducedBranch(@), equal($0, txTimestampBytes) ),  // in produced output must be equal to transaction ts
	and( isConsumedBranch(@), lessThan($0, txTimestampBytes) ) // tx timestamp must be strongly greater than input
)

func amountAndTimestampValid : and(
	outputTimestampValid(slice($0,0,3)),
	tokensValid(tail($0,3))
)

func mainConstraint : amountAndTimestampValid(selfConstraintData)
`
