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
		addrED25519FromPubKey(                 // public key
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
const TokensConstraint = `
// Tokens valid if it has exactly 8 non-0 bytes. It is validated both on consumed output and produced output
func tokensValid: and(
	equal(len8($0),8),
	not(isZero($0))
)

func tokensConstraint : tokensValid(selfConstraintData)
`

const TimestampConstraint = `
// Timestamp is 4 bytes of Unix timestamp in seconds. 
// Timestamp must be present in each output and in the TxTimestamp as transaction timestamp
// Timestamp constraint:
// - for transaction timestamp only 4 byte length is checked
// - for produced output the timestamp must be equal to the transaction timestamp
// - for consumed input the timestamp must be strictly less than the transaction timestamp

func timestampValid: and(
	equal(len8($0),4),  // must always be 4 bytes
	or(
		and( isProducedBranch(@), equal($0, txTimestampBytes) ),  // in produced output must be equal to transaction ts
	    and( isConsumedBranch(@), lessThan($0, txTimestampBytes) ) // tx timestamp must be strongly greater than input
	) 
)

func timestampConstraint: timestampValid(selfConstraintData)
`
