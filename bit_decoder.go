package main

import (
	"io"
)

type bitDecoder struct {
	// Buffer to hold the bytes
	buffer []byte
	// Buffer size
	bufSize int
	// max reads from the buffer
	maxRead int
	// Position in the buffer
	n int
	// bit position
	bit uint8
	// Reader to read the bytes from
	r io.Reader
	// Counter of decoded bytes
	decoded int
}

// New decoder with a specific buffer size and an output writer
func newBitDecoder(r io.Reader, bufSize int) *bitDecoder {
	return &bitDecoder{
		buffer:  make([]byte, bufSize),
		bufSize: bufSize,
		r:       r,
		n:       bufSize + 1,
		maxRead: bufSize,
	}
}

func (bd *bitDecoder) readBit() (uint8, error) {
	var needRead = bd.n > (bd.maxRead - 1)

	// Check if we're done reading
	if needRead && bd.maxRead != bd.bufSize {
		return 0, io.EOF
	}

	// Check if we need to read from the buffer
	if needRead {
		bd.n = 0

		// Read into the buffer
		bd.buffer = make([]byte, bd.bufSize)
		n, err := bd.r.Read(bd.buffer)
		if err != nil {
			return 0, err
		}
		bd.maxRead = n
	}

	// Return
	bd.bit++
	bit := (bd.buffer[bd.n] >> (8 - bd.bit)) & 0x1

	if bd.bit == 8 {
		bd.bit = 0
		bd.n++
	}

	return bit, nil
}
