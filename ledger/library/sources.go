package library

const SigLockConstraint = `
def unlockBlock(0) = _atPath(
	concat(0x0000, _slice(_path, 2, 5))
)

def referencedConstraint(0) = _atPath(
	concat(	_slice(_path,0,2), unlockBlock)
)

def referencedUnlockBlock(0) = _atPath(
	concat(	0, _slice(_path,1,2), unlockBlock) 
)

// unlock block 2 bytes of index 1 bytes of index of the referenced address block. The constraints should be identical
def checkED25519RefUnlock(0) = and(
	_not(_equal(_len8(unlockBlock),3)),
	_equal(
		referencedConstraint,
		referencedUnlockBlock
	)
)

def essence(0) = concat(
	_atPath(0x0001), 
	_atPath(0x0002), 
	_atPath(0x0003), 
	_atPath(0x0004)
)

def addrED25519FromPubKey(1) = blake2b($0)

def validSigED25519(0) = and(
	validSignature(
		essence, 
		_slice(unlockBlock(), 0, 64), 
		_slice(unlockBlock(), 64, 96)
	),
	_equal(
		_data, 
		addrED25519FromPubKey(
			_slice(unlockBlock(), 64, 96)
		)
	)
)

def sigLocED25519(3) = _if(
    _equal(
		_slice(_path,0,1), 
		1
	),    // consumed output
    _equal(_len8(_data), 32),         // ok if len of the data is 32, otherwise fail
	_if(
		_equal(_len8(unlockBlock), 3),    // references
		checkED25519RefUnlock,
		validSigED25519
	)
)
`
