package processor

type ByteFilter interface {
	ShouldProcess(currentByte byte) bool
}
