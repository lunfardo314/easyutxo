package funengine

var sigLockConstraint = `
def unlockBlock(->S) = bytesAtPath(concat(bytes(0,0),slice(path, 2, 5)))


def referencedConstraint(->S) = bytesAtPath(slice(concat(path(),0,2), unlockBlock()))

def referencedUnlockBlock(->S) = bytesAtPath(
	concat(
		0, 
		concat(slice(path(),1,2))
	), 
	unlockBlock()
)

// unlock block 2 bytes of index 1 bytes of index of the referenced address block. The constraints should be identical
def checkED25519RefUnlock(->S) = and(
	equal(
		referencedConstraint(),
		referencedUnlockBlock()
	),
	not(equal(len(),3))
)

def essence(->S) = concat(
	bytesAtPath(bytes(0,1)), 
	concat(
		bytesAtPath(bytes(0,2)), 
		concat(
			bytesAtPath(bytes(0,3)), 
			bytesAtPath(bytes(0,4))
		)
	)
)

def addrED25519FromPubKey(S,S->S) = blake2b(param(0))

def validSigED25519(->S) = and(
	validSignature(essence(), slice(unlockBlock(), 0, 64), slice(unlockBlock(), 64, 96)),
	equal(data(), addrED25519FromPubKey(slice(unlockBlock(), 64, 96)))
)

def sigLocED25519(S,S,S->S) = if(
    equal(atIndex(path(),0), 1),    // consumed output
    equal(len(data()), 32),        // ok if len of the data is 32, otherwise fail
	if(
		equal(len(unlockBlock()), 3),    // references
		checkED25519RefUnlock(unlockBlock()),
		validSigED25519()
	)
)
`
