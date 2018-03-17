package main

import (
	"strconv"
)

// Helper to print bits
func bitString(input []byte) string {
	output := "-"

	for _, v := range input {
		output += strconv.FormatUint(uint64(v), 2) + "-"
	}

	return output
}
