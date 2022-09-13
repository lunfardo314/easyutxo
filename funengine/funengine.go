package funengine

var sigLockConstraint = `
def unlockBlock() = bytesAtPath(concat(bytes(0,0),slice(path(), 2, 5)))

def sigLocED25519(V,S,S) = if(
    equal8(atIndex(path(),0), 1),    // consumed output
    equal8(len(data()), 32),        // ok if len of the data is 32, otherwise fail
	if(
		equal8(len(unlockBlock()), 3),    // references
		checkED25519RefUnlock(),
		validSigED25519()
	)
)

def validSigED25519() = and(
	validSignature(essenceBytes(), slice(unlockBlock(), 0, 64), slice(unlockBlock(), 64, 96)),
	equalBytes(data(), addrED25519FromPuKey(slice(unlockBlock(), 64, 96)))
)

def referencedUnlockBlock() = bytesAtPath(
	concat(
		0, 
		concat(slice(path(),1,2))
	), 
	unlockBlock()
)

def referencedConstraint() = bytesAtPath(slice(concat(path(),0,2), unlockBlock()))

// unlock block 2 bytes of index 1 bytes of index of the referenced address block. The constraints should be identical
def checkED25519RefUnlock() = and(
	equalBytes(
		referencedConstraint(),
		referencedUnlockBlock()
	),
	not(equal8(len(),3))
)

def essence() = concat(
	bytesAtPath(bytes(0,1)), 
	concat(
		bytesAtPath(bytes(0,2)), 
		concat(
			bytesAtPath(bytes(0,3)), 
			bytesAtPath(bytes(0,4))
		)
	)
)
`
