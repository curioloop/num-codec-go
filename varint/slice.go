package varint

// AppendUvarint32s appends the Uvarint encodings of every value in vs to dst.
func AppendUvarint32s(dst []byte, vs []uint32) []byte {
	for _, v := range vs {
		dst = AppendUvarint32(dst, v)
	}
	return dst
}

// AppendUvarint64s appends the Uvarint encodings of every value in vs to dst.
func AppendUvarint64s(dst []byte, vs []uint64) []byte {
	for _, v := range vs {
		dst = AppendUvarint64(dst, v)
	}
	return dst
}

// AppendZigZag32s appends the ZigZag+Uvarint encodings of every value in vs to dst.
func AppendZigZag32s(dst []byte, vs []int32) []byte {
	for _, v := range vs {
		dst = AppendZigZag32(dst, v)
	}
	return dst
}

// AppendZigZag64s appends the ZigZag+Uvarint encodings of every value in vs to dst.
func AppendZigZag64s(dst []byte, vs []int64) []byte {
	for _, v := range vs {
		dst = AppendZigZag64(dst, v)
	}
	return dst
}

// AppendDecodeUvarint32s appends every Uvarint decoded from data to dst.
func AppendDecodeUvarint32s(dst []uint32, data []byte) ([]uint32, error) {
	for len(data) > 0 {
		v, n, err := Uvarint32(data)
		if err != nil {
			return dst, err
		}
		dst = append(dst, v)
		data = data[n:]
	}
	return dst, nil
}

// AppendDecodeUvarint64s appends every Uvarint decoded from data to dst.
func AppendDecodeUvarint64s(dst []uint64, data []byte) ([]uint64, error) {
	for len(data) > 0 {
		v, n, err := Uvarint64(data)
		if err != nil {
			return dst, err
		}
		dst = append(dst, v)
		data = data[n:]
	}
	return dst, nil
}

// AppendDecodeZigZag32s appends every ZigZag decoded from data to dst.
func AppendDecodeZigZag32s(dst []int32, data []byte) ([]int32, error) {
	for len(data) > 0 {
		v, n, err := ZigZag32(data)
		if err != nil {
			return dst, err
		}
		dst = append(dst, v)
		data = data[n:]
	}
	return dst, nil
}

// AppendDecodeZigZag64s appends every ZigZag decoded from data to dst.
func AppendDecodeZigZag64s(dst []int64, data []byte) ([]int64, error) {
	for len(data) > 0 {
		v, n, err := ZigZag64(data)
		if err != nil {
			return dst, err
		}
		dst = append(dst, v)
		data = data[n:]
	}
	return dst, nil
}
