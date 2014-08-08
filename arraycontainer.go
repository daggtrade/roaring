package roaring

type arrayContainer struct {
	content []uint16
}

func (ac *arrayContainer) fillLeastSignificant16bits(x []int, i, mask int) {
	for k := 0; k < len(ac.content); k++ {
		x[k+i] = toIntUnsigned(ac.content[k]) | mask
	}
}

func (ac *arrayContainer) getShortIterator() shortIterable {
	return &shortIterator{ac.content, 0}
}
func (ac *arrayContainer) getSizeInBytes() int {
	return ac.getCardinality()*2 + 4
}

func (ac *arrayContainer) not(firstOfRange, lastOfRange int) container {
	if firstOfRange > lastOfRange {
		return ac.clone()
	}

	// determine the span of array indices to be affected^M
	startIndex := binarySearch(ac.content, uint16(firstOfRange))
	if startIndex < 0 {
		startIndex = -startIndex - 1
	}
	lastIndex := binarySearch(ac.content, uint16(lastOfRange))
	if lastIndex < 0 {
		lastIndex = -lastIndex - 2
	}
	currentValuesInRange := lastIndex - startIndex + 1
	spanToBeFlipped := lastOfRange - firstOfRange + 1
	newValuesInRange := spanToBeFlipped - currentValuesInRange
	cardinalityChange := newValuesInRange - currentValuesInRange
	newCardinality := len(ac.content) + cardinalityChange

	if newCardinality >= arrayDefaultMaxSize {
		return ac.toBitmapContainer().not(firstOfRange, lastOfRange)
	}
	answer := newArrayContainer()
	answer.content = make([]uint16, newCardinality, newCardinality) //a hack for sure

	copy(answer.content, ac.content[:startIndex])
	outPos := startIndex
	inPos := startIndex
	valInRange := firstOfRange
	for ; valInRange <= lastOfRange && inPos <= lastIndex; valInRange++ {
		if uint16(valInRange) != ac.content[inPos] {
			answer.content[outPos] = uint16(valInRange)
			outPos++
		} else {
			inPos++
		}
	}

	for ; valInRange <= lastOfRange; valInRange++ {
		answer.content[outPos] = uint16(valInRange)
		outPos++
	}

	for i := lastIndex + 1; i < len(ac.content); i++ {
		answer.content[outPos] = ac.content[i]
		outPos++
	}
	answer.content = answer.content[:newCardinality]
	return answer

}

func (ac *arrayContainer) equals(o interface{}) bool {
	srb := o.(*arrayContainer)
	if srb != nil {
		if len(srb.content) != len(ac.content) {
			return false
		}
		for i := 0; i < len(ac.content); i++ {
			if ac.content[i] != srb.content[i] {
				return false
			}
		}
		return true
	}
	return false
}

func (ac *arrayContainer) toBitmapContainer() *bitmapContainer {
	bc := newBitmapContainer()
	bc.loadData(ac)
	return bc

}
func (ac *arrayContainer) add(x uint16) container {
	if len(ac.content) >= arrayDefaultMaxSize {
		a := ac.toBitmapContainer()
		a.add(x)
		return a
	}
	if (len(ac.content) == 0) || (x > ac.content[len(ac.content)-1]) {
		ac.content = append(ac.content, x)
		return ac
	}
	loc := binarySearch(ac.content, x)
	if loc < 0 {
		s := ac.content
		i := -loc - 1
		s = append(s, 0)
		copy(s[i+1:], s[i:])
		s[i] = x
		ac.content = s
	}
	return ac
}

func (ac *arrayContainer) or(a container) container {
	switch a.(type) {
	case *arrayContainer:
		return ac.orArray(a.(*arrayContainer))
	case *bitmapContainer:
		return a.or(ac)
	}
	return nil
}

func (ac *arrayContainer) orArray(value2 *arrayContainer) container {
	value1 := ac
	totalCardinality := value1.getCardinality() + value2.getCardinality()
	if totalCardinality > arrayDefaultMaxSize { // it could be a bitmap!^M
		bc := newBitmapContainer()
		for k := 0; k < len(value2.content); k++ {
			i := uint(toIntUnsigned(value2.content[k])) >> 6
			bc.bitmap[i] |= (1 << (value2.content[k] % 64))
		}
		for k := 0; k < len(ac.content); k++ {
			i := uint(toIntUnsigned(ac.content[k])) >> 6
			bc.bitmap[i] |= (1 << (ac.content[k] % 64))
		}
		bc.cardinality = int(popcntSlice(bc.bitmap))
		if bc.cardinality <= arrayDefaultMaxSize {
			return bc.toArrayContainer()
		}
		return bc
	}
	desiredCapacity := totalCardinality
	answer := newArrayContainerCapacity(desiredCapacity)
	nl := union2by2(value1.content, value2.content, answer.content)
	answer.content = answer.content[:nl] //what is this voodo?
	return answer
}

func (ac *arrayContainer) and(a container) container {
	switch a.(type) {
	case *arrayContainer:
		return ac.andArray(a.(*arrayContainer))
	case *bitmapContainer:
		return a.and(ac)
	}
	return nil
}

func (ac *arrayContainer) xor(a container) container {
	switch a.(type) {
	case *arrayContainer:
		return ac.xorArray(a.(*arrayContainer))
	case *bitmapContainer:
		return a.xor(ac)
	}
	return nil
}

func (ac *arrayContainer) xorArray(value2 *arrayContainer) container {
	value1 := ac
	totalCardinality := value1.getCardinality() + value2.getCardinality()
	if totalCardinality > arrayDefaultMaxSize { // it could be a bitmap!^M
		bc := newBitmapContainer()
		for k := 0; k < len(value2.content); k++ {
			i := uint(toIntUnsigned(value2.content[k])) >> 6
			bc.bitmap[i] ^= (1 << value2.content[k])
		}
		for k := 0; k < len(ac.content); k++ {
			i := uint(toIntUnsigned(ac.content[k])) >> 6
			bc.bitmap[i] ^= (1 << ac.content[k])
		}
		bc.cardinality = int(popcntSlice(bc.bitmap))

		if bc.cardinality <= arrayDefaultMaxSize {
			return bc.toArrayContainer()
		}
		return bc
	}
	desiredCapacity := totalCardinality
	answer := newArrayContainerCapacity(desiredCapacity)
	length := exclusiveUnion2by2(value1.content, value2.content, answer.content)
	answer.content = answer.content[:length]
	return answer

}

func (ac *arrayContainer) andNot(a container) container {
	switch a.(type) {
	case *arrayContainer:
		return ac.andNotArray(a.(*arrayContainer))
	case *bitmapContainer:
		return a.andNot(ac)
	}
	return nil
}

func (ac *arrayContainer) andNotArray(value2 *arrayContainer) container {
	value1 := ac
	desiredcapacity := value1.getCardinality()
	answer := newArrayContainerCapacity(desiredcapacity)
	length := difference(value1.content, value2.content, answer.content)
	answer.content = answer.content[:length]
	return answer
}

func copyOf(array []uint16, size int) []uint16 {
	result := make([]uint16, size)
	for i, x := range array {
		if i == size {
			break
		}
		result[i] = x
	}
	return result
}

func (ac *arrayContainer) inot(firstOfRange, lastOfRange int) container {
	// determine the span of array indices to be affected
	startIndex := binarySearch(ac.content, uint16(firstOfRange))
	if startIndex < 0 {
		startIndex = -startIndex - 1
	}
	lastIndex := binarySearch(ac.content, uint16(lastOfRange))
	if lastIndex < 0 {
		lastIndex = -lastIndex - 1 - 1
	}
	currentValuesInRange := lastIndex - startIndex + 1
	spanToBeFlipped := lastOfRange - firstOfRange + 1

	newValuesInRange := spanToBeFlipped - currentValuesInRange
	buffer := make([]uint16, newValuesInRange)
	cardinalityChange := newValuesInRange - currentValuesInRange
	newCardinality := len(ac.content) + cardinalityChange
	if cardinalityChange > 0 {
		if newCardinality > len(ac.content) {
			if newCardinality >= arrayDefaultMaxSize {
				return ac.toBitmapContainer().inot(firstOfRange, lastOfRange)
			}
			ac.content = copyOf(ac.content, newCardinality)
		}
		base := lastIndex + 1
		//copy(self.content[lastIndex+1+cardinalityChange:], self.content[lastIndex+1:len(self.content)-1-lastIndex])
		copy(ac.content[lastIndex+1+cardinalityChange:], ac.content[base:base+len(ac.content)-1-lastIndex])

		ac.negateRange(buffer, startIndex, lastIndex, firstOfRange, lastOfRange)
	} else { // no expansion needed
		ac.negateRange(buffer, startIndex, lastIndex, firstOfRange, lastOfRange)
		if cardinalityChange < 0 {

			for i := startIndex + newValuesInRange; i < newCardinality; i++ {
				ac.content[i] = ac.content[i-cardinalityChange]
			}
		}
	}
	ac.content = ac.content[:newCardinality]
	return ac
}

func (ac *arrayContainer) negateRange(buffer []uint16, startIndex, lastIndex, startRange, lastRange int) {
	// compute the negation into buffer

	outPos := 0
	inPos := startIndex // value here always >= valInRange,
	// until it is exhausted
	// n.b., we can start initially exhausted.

	valInRange := startRange
	for ; valInRange <= lastRange && inPos <= lastIndex; valInRange++ {
		if uint16(valInRange) != ac.content[inPos] {
			buffer[outPos] = uint16(valInRange)
			outPos++
		} else {
			inPos++
		}
	}

	// if there are extra items (greater than the biggest
	// pre-existing one in range), buffer them
	for ; valInRange <= lastRange; valInRange++ {
		buffer[outPos] = uint16(valInRange)
		outPos++
	}

	if outPos != len(buffer) {
		//panic("negateRange: outPos " + outPos + " whereas buffer.length=" + len(buffer))
		panic("negateRange: outPos  whereas buffer.length=")
	}

	for i, item := range buffer {
		ac.content[i] = item
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func (ac *arrayContainer) andArray(value2 *arrayContainer) *arrayContainer {

	desiredcapacity := min(ac.getCardinality(), value2.getCardinality())
	answer := newArrayContainerCapacity(desiredcapacity)
	length := intersection2by2(
		ac.content,
		value2.content,
		answer.content)
	answer.content = answer.content[:length]
	return answer

}

func (ac *arrayContainer) getCardinality() int {
	return len(ac.content)
}
func (ac *arrayContainer) clone() container {
	ptr := arrayContainer{make([]uint16, len(ac.content))}
	copy(ptr.content, ac.content[:])
	return &ptr
}
func (ac *arrayContainer) contains(x uint16) bool {
	return binarySearch(ac.content, x) >= 0
}

func (ac *arrayContainer) loadData(bitmapContainer *bitmapContainer) {
	ac.content = make([]uint16, bitmapContainer.cardinality, bitmapContainer.cardinality)
	bitmapContainer.fillArray(ac.content)
}
func newArrayContainer() *arrayContainer {
	p := new(arrayContainer)
	return p
}

func newArrayContainerCapacity(size int) *arrayContainer {
	p := new(arrayContainer)
	p.content = make([]uint16, 0, size)
	return p
}

func newArrayContainerSize(size int) *arrayContainer {
	p := new(arrayContainer)
	p.content = make([]uint16, size, size)
	return p
}

func newArrayContainerRange(firstOfRun, lastOfRun int) *arrayContainer {
	valuesInRange := lastOfRun - firstOfRun + 1
	this := newArrayContainerCapacity(valuesInRange)
	for i := 0; i < valuesInRange; i++ {
		this.content = append(this.content, uint16(firstOfRun+i))
	}
	return this
}