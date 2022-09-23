package ledger

const SigLockConstraint = `

// if unlock block is 3 bytes (b0, b1, b2), where (b0, b1) long index of output and b2 block index
// and it references constraint in the consumed output with the same constraint (address), it is unlocked
// (b0, b1) must point to the place strongly preceding the current one. It prevents cycle and guarantees
// at least one (normally) of sig-locked outputs is unlocked with signature

func unlockedWithReference: and(
	equal(len8(unlockBlock), 3),
	lessThan(slice(unlockBlock,0,2), outputIndex),
	equal(constraint, referencedConstraint)
)

// otherwise, the signature must be valid and hash of the public key must be equal to the provided address

func unlockedWithSigED25519: and(
	validSignatureED25519(
		txEssenceBytes, 
		slice(unlockBlock, 0, 64), 
		slice(unlockBlock, 64, 96)
	),
	equal(
		constraintData, 
		addrED25519FromPubKey(
			slice(unlockBlock, 64, 96)
		)
	)
)

// the interpreter first will check first condition and if it is true, won't evaluate the second one 

func sigLocED25519: or(
	unlockedWithReference,
	unlockedWithSigED25519
)
`
