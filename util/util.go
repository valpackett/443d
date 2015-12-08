package util

type ByLengthDesc []string

func (a ByLengthDesc) Len() int           { return len(a) }
func (a ByLengthDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLengthDesc) Less(i, j int) bool { return len(a[i]) > len(a[j]) }
