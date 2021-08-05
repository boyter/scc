package processor

type SimpleFilter struct {
	BloomMap [256]bool
}

func (filter SimpleFilter) ShouldProcess(currentByte byte) bool {
	return filter.BloomMap[currentByte]
}
