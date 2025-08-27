package hash

func CalculateDJB2(key string) uint32 {
	var hash uint32 = 5381
	for _, c := range []byte(key) {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}
