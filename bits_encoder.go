package main

import (
	"encoding/binary"
	"io"
)

type bitEncoder struct {
	// Buffer to hold the bytes
	buffer []uint64
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
		buffer:  make([]uint64, bufSize),
		bufSize: bufSize,
		w:       w,
	}
}

// Return the number of byte encoded and flushed to the writer
func (be *bitEncoder) totalEncoded() int {
	return be.encoded
}

// This function writes bits, it flushes the buffer if necessary
func (be *bitEncoder) writeBits(bits uint64, size uint8) error {
	var spaceLeft = 64 - be.bits

	// No space left, flush
	if spaceLeft == 0 {
		if err := be.flush(); err != nil {
			return err
		}
		spaceLeft = 64
	}

	// Not there is not enough space to write the bits
	if size > spaceLeft {
		var remainingBitCount = size - spaceLeft

		// Write in the remaining space
		be.buffer[be.n] = (be.buffer[be.n] << spaceLeft) | (bits >> remainingBitCount)
		be.bits = 64

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
	if be.bits != 64 {
		be.buffer[be.n] <<= (64 - be.bits)
		be.bits = 0
	}

	// Write to the output buffer
	if err := binary.Write(be.w, binary.BigEndian, be.buffer[:be.n+1]); err != nil {
		return err
	}
	be.encoded += (be.n + 1) * 8

	// Reset the buffer
	be.n = 0
	be.bits = 0
	be.buffer = make([]uint64, be.bufSize)

	return nil
}
