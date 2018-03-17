package main

import (
	"bytes"
	"io"
)

type bitEncoder struct {
	// Buffer to hold the bytes
	buffer []byte
	// Position in the buffer
	n int
	// bit position
	bits uint8
	// Buffer size
	bufSize int
	// Writer to flush the buffer to
	w io.Writer
	// Counter of encoded bytes
	encoded int
}

// New encoder with a specific buffer size and an output writer
func newBitEncoder(w io.Writer, bufSize int) *bitEncoder {
	return &bitEncoder{
		buffer:  make([]byte, bufSize),
		bufSize: bufSize,
		w:       w,
	}
}

// Return the number of byte encoded and flushed to the writer
func (be *bitEncoder) totalEncoded() int {
	return be.encoded
}

// This function writes bits, it flushes the buffer if necessary
func (be *bitEncoder) writeBits(bits, size uint8) error {
	var spaceLeft = 8 - be.bits

	// No space left, flush
	if spaceLeft == 0 {
		if err := be.flush(); err != nil {
			return err
		}
		spaceLeft = 8
	}

	// Not there is not enough space to write the bits
	if size > spaceLeft {
		var remainingBitCount = size - spaceLeft

		// Write in the remaining space
		be.buffer[be.n] = (be.buffer[be.n] << spaceLeft) | (bits >> remainingBitCount)
		be.bits = 8

		// Flush
		if (be.n + 1) > (be.bufSize - 1) {
			if err := be.flush(); err != nil {
				return err
			}
		} else {
			// Use the next byte
			be.n++
		}

		// Write to the remaining bits to the next byte
		be.buffer[be.n] = (bits & ((1 << remainingBitCount) - 1))
		be.bits = remainingBitCount
	} else {
		// Simple case
		be.buffer[be.n] = (be.buffer[be.n] << size) | bits
		be.bits += size
	}

	return nil
}

// Flush the buffer to the writer and return the number of bits written
func (be *bitEncoder) flush() error {
	// Nothing to write
	if be.n == 0 && be.bits == 0 {
		return nil
	}

	// Pad right the remaining space in the byte:
	// bits = 2
	// shift 6 - bits to the left
	// 00000011 => 11000000
	if be.bits != 8 {
		be.buffer[be.n] <<= (8 - be.bits)
		be.bits = 0
	}

	// Write to the output buffer
	buf := bytes.NewBuffer(be.buffer)
	n, err := io.CopyN(be.w, buf, int64(be.n+1))
	if err != nil {
		return err
	}
	be.encoded += int(n)

	// Reset the buffer
	be.n = 0
	be.bits = 0
	be.buffer = make([]byte, be.bufSize)

	return nil
}
