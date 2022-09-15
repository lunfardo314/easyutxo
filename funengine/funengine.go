package funengine

var sigLockConstraint = `
def unlockBlock(0) = bytesAtPath(concat(bytes(0,0),slice(_path, 2, 5)))


def referencedConstraint(0) = bytesAtPath(slice(concat(_path,0,2), unlockBlock()))

def referencedUnlockBlock(0) = bytesAtPath(
	concat(
		0, 
		concat(slice(_path,1,2))
	), 
	unlockBlock()
)

// unlock block 2 bytes of index 1 bytes of index of the referenced address block. The constraints should be identical
def checkED25519RefUnlock(0) = and(
	equal(
		referencedConstraint(),
		referencedUnlockBlock()
	),
	not(equal(len(),3))
)

def essence(0) = concat(
	bytesAtPath(bytes(0,1)), 
	concat(
		bytesAtPath(bytes(0,2)), 
		concat(
			bytesAtPath(bytes(0,3)), 
			bytesAtPath(bytes(0,4))
		)
	)
)

def addrED25519FromPubKey(1) = blake2b(_param(0))

def validSigED25519(0) = and(
	validSignature(essence(), slice(unlockBlock(), 0, 64), slice(unlockBlock(), 64, 96)),
	equal(_data, addrED25519FromPubKey(slice(unlockBlock(), 64, 96)))
)

def sigLocED25519(3) = if(
    equal(atIndex(_path,0), 1),    // consumed output
    equal(len(_data), 32),        // ok if len of the data is 32, otherwise fail
	if(
		equal(len(unlockBlock()), 3),    // references
		checkED25519RefUnlock(unlockBlock()),
		validSigED25519()
	)
)
`
