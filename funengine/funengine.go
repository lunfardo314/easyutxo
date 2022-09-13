package funengine

/*
	Functional validation of the transaction

	Each output at #0 contains evaluation to the function. The function receives global environment plus local parameters:
	- global context tree
	- invocation path <output path>!!0
	-- if <output path>[0] == 0 it is tx output context
	-- if <output path>[0] == 1 it is consumed output context
	-- invocation data
	- #0 element of each output is invocation:
	-- <funcode><data>: global library evaluation
	-- <inline funcode><inline code>: inline evaluation
	-- <local funcode><code hash><data>: local library evaluation
	- <function evaluation> := <funcode>(<params>)
	- <params> ::= nil | <function evaluation>[,<params>] | <literal>[,<params>]
	- <literal> ::= nil|bytes(<hex>)|bytes([<n>,...])||<n>||u16/<n>||u32/<n>||u64/<n>
	- all functions with parameter and return types are defined in funcodes
	- param types: 0 - one byte value, 1 - u16, 2 - u32, 3 - u64, V - byte array
*/

var sigLockConstraint = `
def sigLocED25519(V) = if(
    equal8(atIndex(path(),0), 1),    // consumed output
    equal8(len(data(), 32),        // ok if len of the data is 32, otherwise fail
	if(
		equal8(len(unlockBlock()), 3),    // references
		checkED25519RefUnlock(),
		validSigED25519()
	)
)

def unlockBlock() = bytesAtPath(concat(bytes(0,0),slice(path(), 2, 5)))

def validSigED25519() = and(
	validSignature(essenceBytes(), slice(unlockBlock(), 0, 64), slice(unlockBlock(), 64, 96)),
	equalBytes(data(), addrED25519FromPuKey(slice(unlockBlock(), 64, 96)))
)

def referencedUnlockBlock() = bytesAtPath(concat(0, concat(slice(path(),1,2)), unlockBlock())

def referencedConstraint() = bytesAtPath(slice(concat(path(),0,2), unlockBlock()))

// unlock block 2 bytes of index 1 bytes of index of the referenced address block. The constraints should be identical
def checkED25519RefUnlock() = and(
	equalBytes(, referencedUnlockBlock()),
	not(equal8(len(),3)
)

essence() = concat(bytesAtPath(bytes(0,1)), concat(bytesAtPath(bytes(0,2)), concat(bytesAtPath(bytes(0,3)), bytesAtPath(bytes(0,4))))
`
