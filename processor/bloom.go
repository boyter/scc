package processor

// Prime number less than 256
const BloomPrime = 251

var BloomTable [256]uint64

func init() {
	for i := range BloomTable {
		BloomTable[i] = BloomHash(byte(i))
	}
}

func BloomHash(b byte) uint64 {
	i := uint64(b)

	k := (i^BloomPrime) * i

	k1 := k & 0x3f
	k2 := k >> 1 & 0x3f
	k3 := k >> 2 & 0x3f

	return (1 << k1) | (1 << k2) | (1 << k3)
}
