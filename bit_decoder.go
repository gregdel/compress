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
	// number of decoded bytes
	decoded uint64
	// expected ouput size
	expectedOuputSize uint64
}

// New decoder with a specific buffer size and an output writer
func newBitDecoder(r io.Reader, bufSize int, expectedOuputSize uint64) *bitDecoder {
	return &bitDecoder{
		buffer:            make([]byte, bufSize),
		bufSize:           bufSize,
		r:                 r,
		n:                 bufSize + 1,
		maxRead:           bufSize,
		expectedOuputSize: expectedOuputSize,
	}
}

// Return the number of byte decoded
func (bd *bitDecoder) totalDecoded() uint64 {
	return bd.decoded
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

func (bd *bitDecoder) decode(w io.Writer, c *compressor) error {
	for {
		// Read a bit from the input
		char, err := bd.getCharFromTree(c)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		// Write the char
		n, err := w.Write([]byte{char})
		if err != nil {
			return err
		}

		bd.decoded += uint64(n)
	}
}

func (bd *bitDecoder) getCharFromTree(c *compressor) (byte, error) {
	node := c.treeRoot
	for {
		if bd.decoded >= bd.expectedOuputSize {
			return 0, io.EOF
		}

		bit, err := bd.readBit()
		if err != nil {
			return 0, err
		}

		if bit == 1 {
			node = node.rChild
		} else {
			node = node.lChild
		}

		if node.leaf {
			return node.char, nil
		}
	}
}
