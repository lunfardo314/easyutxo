package ledger

const SigLockConstraint = `
func unlockBlock: _atPath(
	concat(0x0000, _slice(@, 2, 5))
)

func referencedConstraint: _atPath(
	concat(	_slice(@,0,2), unlockBlock)
)

func referencedUnlockBlock: _atPath(
	concat(	0, _slice(@,1,2), unlockBlock) 
)

// unlock block 2 bytes of index 1 bytes of index of the referenced address block. The constraints should be identical
func checkED25519RefUnlock: and(
	_not(_equal(_len8(unlockBlock),3)),
	_equal(
		referencedConstraint,
		referencedUnlockBlock
	)
)
func addrED25519FromPubKey: blake2b($0)

func validSigED25519: and(
	validSignature(
		txEssenceBytes, 
		_slice(unlockBlock(), 0, 64), 
		_slice(unlockBlock(), 64, 96)
	),
	_equal(
		invocationData, 
		addrED25519FromPubKey(
			_slice(unlockBlock(), 64, 96)
		)
	)
)

func sigLocED25519: _if(
    _equal(
		_slice(@,0,1), 
		1
	),    // consumed output
    _equal(_len8(invocationData), 32),         // ok if len of the data is 32, otherwise fail
	_if(
		_equal(_len8(unlockBlock), 3),    // references
		checkED25519RefUnlock,
		validSigED25519
	)
)
`
