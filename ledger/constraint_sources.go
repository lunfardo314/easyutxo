package ledger

const SigLockConstraint = `

// if unlock block is 3 bytes (b0, b1, b2), where (b0, b1) long index of output and b2 block index
// and it references constraint in the consumed output with the same constraint (address), it is unlocked
// (b0, b1) must point to the place strongly preceding the current one. It prevents cycle and guarantees
// at least one (normally) of sig-locked outputs is unlocked with signature

func selfUnlockedWithReference: and(
	equal(len8(selfUnlockBlock), 3),
	lessThan(slice(selfUnlockBlock,0,2), selfOutputIndex),
	equal(selfConstraint, selfReferencedConstraint)
)

// otherwise, the signature must be valid and hash of the public key must be equal to the provided address

func selfUnlockedWithSigED25519: and(
	validSignatureED25519(
		txEssenceBytes, 
		slice(selfUnlockBlock, 0, 64), 
		slice(selfUnlockBlock, 64, 96)
	),
	equal(
		selfConstraintData, 
		addrED25519FromPubKey(
			slice(selfUnlockBlock, 64, 96)
		)
	)
)

// if it is 'consumed' invocation context, only size of the address is checked
// Otherwise the first will check first condition if it is unlocked by reference, otherwise checks unlocking signature
// Second condition not evaluated if the first is true 

func sigLocED25519: if(
	selfConsumedContext,
    equal( len8(selfConstraintData), 32),
	or(
		selfUnlockedWithReference,
		selfUnlockedWithSigED25519
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
