package constraint

// TODO propper calculation of the storage deposit

func MinimumStorageDeposit(outputByteSize, extraWeight uint32) uint64 {
	return uint64(outputByteSize)
}
