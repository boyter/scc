package processor

import "math/rand/v2"

var BloomTable [256]uint64

func init() {
	for i := range BloomTable {
		BloomTable[i] = BloomHash(byte(i))
	}
}

func BloomHash(b byte) uint64 {
	// Since our input is based on ASCII characters (and majority lower case
	// characters) the values are not well distributed through the 0-255 byte
	// range. math/rand gives us a way to generate a value with more well
	// distributed randomness.
	k := rand.New(rand.NewPCG(uint64(b), uint64(b))).Uint64()

	// Mask to slice out a 0-63 value
	var mask64 uint64 = 0b00111111

	// For a bloom filter we only want a few bits set, but distributed
	// through the 64 bit space.
	// The logic here is to slice a value between 0 and 63 from k, and set a
	// single bit in the output hash based on that.
	// Setting three bits this way seems to give the best results. Fewer bits
	// make the hash not unique enough, more leads to overcrowding the bloom
	// filter.
	var hash uint64
	for i := uint64(0); i < 3; i++ {
		n := k >> (i * 8) & mask64
		hash |= 1 << n
	}

	return hash
}
