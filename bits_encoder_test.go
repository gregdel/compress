package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBitEncoder(t *testing.T) {
	type bits struct {
		bits uint64
		size uint8
	}

	tt := []struct {
		name           string
		expectedResult []byte
		toWrite        []bits
	}{
		{
			name: "zero bit padding",
			// 00000111
			toWrite: []bits{{0x7, 0x3}},
			// 11100000
			expectedResult: []byte{0xe0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "two bytes with bit padding",
			// 10101010 00000011
			toWrite: []bits{{0xaa, 8}, {0x3, 2}},
			// 11100000 11000000
			expectedResult: []byte{0xaa, 0xc0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "concat two bytes",
			// 00101010 00000011
			toWrite: []bits{{0x2a, 6}, {0x3, 2}},
			// 10101011
			expectedResult: []byte{0xab, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "concat three bytes",
			// 00101010 00000110 00000011
			toWrite: []bits{{0x2a, 6}, {0x6, 3}, {0x3, 2}},
			// 10101011 01100000
			expectedResult: []byte{0xab, 0x60, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "two bytes with splitting",
			// 00101010 00000111
			toWrite: []bits{{0x2a, 6}, {0x7, 3}},
			// 10101011 10000000
			expectedResult: []byte{0xab, 0x80, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "four bytes with splitting",
			// 00101010 00000111 11111111 00000111
			toWrite: []bits{{0x2a, 6}, {0x7, 3}, {0xff, 8}, {0x7, 3}},
			// 10101011 1111111 11110000
			expectedResult: []byte{0xab, 0xff, 0xf0, 0, 0, 0, 0, 0},
		},
		{
			name: "uint16",
			// 110101010 00000011
			toWrite: []bits{{0x1aa, 9}, {0x3, 2}},
			// 11010101 01100000
			expectedResult: []byte{0xd5, 0x60, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := &bytes.Buffer{}

			// New encoder with a buffer of size 2
			enc := newBitEncoder(output, 2)

			// Write to the buffer
			for _, b := range tc.toWrite {
				if err := enc.writeBits(b.bits, b.size); err != nil {
					t.Fatalf("failed to write bit: %q", err)
				}
			}

			// Flush the buffer
			if err := enc.flush(); err != nil {
				t.Fatalf("failed to flush buffer: %q", err)
			}

			resultSize := output.Len()
			expectedResultSize := len(tc.expectedResult)
			if resultSize != expectedResultSize {
				t.Fatalf("output buffer should be of size %d but is of size %d",
					expectedResultSize, resultSize)
			}

			if enc.totalEncoded() != expectedResultSize {
				t.Fatalf("total encoded should be %d but is %d",
					expectedResultSize, enc.totalEncoded())
			}

			// Read the byte
			result := output.Bytes()

			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Fatalf("expected %s, got %s",
					bitString(tc.expectedResult),
					bitString(result),
				)
			}
		})
	}
}
