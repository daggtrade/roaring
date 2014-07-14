package goroaring

type short uint16

func BitCount(i int64) int {
	x := uint64(i)
	// bit population count, see
	// http://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
	x -= (x >> 1) & 0x5555555555555555
	x = (x>>2)&0x3333333333333333 + x&0x3333333333333333
	x += x >> 4
	x &= 0x0f0f0f0f0f0f0f0f
	x *= 0x0101010101010101
	return int(x >> 56)
}

func FillArrayAND(container []short, bitmap1, bitmap2 []int64) {
	pos := 0
	if len(bitmap1) != len(bitmap2) {
		panic("array lengths don't match")
	}
	for k := 0; k < len(bitmap1); k++ {
		bitset := bitmap1[k] & bitmap2[k]
		for bitset != 0 {
			t := bitset & -bitset
			container[pos] = short((k*64 + BitCount(t-1)))
			pos++
			bitset ^= t
		}
	}
}

func FillArrayANDNOT(container []short, bitmap1, bitmap2 []int64) {
	pos := 0
	if len(bitmap1) != len(bitmap2) {
		panic("array lengths don't match")
	}
	for k := 0; k < len(bitmap1); k++ {
		bitset := bitmap1[k] &^ bitmap2[k]
		for bitset != 0 {
			t := bitset & -bitset
			container[pos] = short((k*64 + BitCount(t-1)))
			pos++
			bitset ^= t
		}
	}
}

func FillArrayXOR(container []short, bitmap1, bitmap2 []int64) {
	pos := 0
	if len(bitmap1) != len(bitmap2) {
		panic("array lengths don't match")
	}
	for k := 0; k < len(bitmap1); k++ {
		bitset := bitmap1[k] ^ bitmap2[k]
		for bitset != 0 {
			t := bitset & -bitset
			container[pos] = short((k*64 + BitCount(t-1)))
			pos++
			bitset ^= t
		}
	}
}

func Highbits(x int) short {
	u := uint(x)
	return short(u >> 16)
}
func Lowbits(x int) short {
	return short(x & 0xFFFF)
}

func MaxLowBit() short {
	return short(0xFFFF)
}

func ToIntUnsigned(x short) int {
	return int(x & 0xFFFF)
}

func AdvanceUntil(
	array []short,
	pos int,
	length int,
	min short) int {
	lower := pos + 1

	if lower >= length || array[lower] >= min {
		return lower
	}

	spansize := 1

	for lower+spansize < length && array[lower+spansize] < min {
		spansize *= 2
	}
	var upper int
	if lower+spansize < length {
		upper = lower + spansize
	} else {
		upper = length - 1
	}

	if array[upper] == min {
		return upper
	}

	if array[upper] < min {
		// means
		// array
		// has no
		// item
		// >= min
		// pos = array.length;
		return length
	}

	// we know that the next-smallest span was too small
	lower += (spansize / 2)

	mid := 0
	for lower+1 != upper {
		mid = (lower + upper) / 2
		if array[mid] == min {
			return mid
		} else if array[mid] < min {
			lower = mid
		} else {
			upper = mid
		}
	}
	return upper

}