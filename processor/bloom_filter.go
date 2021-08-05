package processor

type BloomFilter struct {
	ProcessBytesMask uint64
}

func (filter BloomFilter) ShouldProcess(currentByte byte) bool {
	k := BloomTable[currentByte]
	if k&filter.ProcessBytesMask != k {
		return false
	}
	return true
}
