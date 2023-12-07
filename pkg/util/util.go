package util

import (
	"fmt"
	"strings"
)

// ThousandSep format int with thousands divided by sep
func ThousandSep(n int64, sep string) string {
	s := fmt.Sprintf("%v", n)

	// divide by 3 symbols from end
	size := len(s)
	res := make([]string, 0, 1+size/3)
	for i := 0; i < size; i += 3 {
		from := size - i - 3
		if from < 0 {
			from = 0
		}
		res = append(res, s[from:size-i])
	}

	ReverseInPlace(res)

	return strings.Join(res, sep)
}

// ReverseInPlace reverses any slice in place
func ReverseInPlace[T any](arr []T) {
	if len(arr) < 2 {
		return
	}
	size := len(arr)
	for i := 0; i < size/2; i++ {
		arr[i], arr[size-i-1] = arr[size-i-1], arr[i]
	}
}
