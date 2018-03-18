package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestReadBit(t *testing.T) {
	input := []byte{
		// 10101010 00000011 00000111
		0xaa, 0x3, 0x7,
	}
	expectedOutput := []byte{
		// 10101010
		1, 0, 1, 0, 1, 0, 1, 0,
		// 00000011
		0, 0, 0, 0, 0, 0, 1, 1,
		// 00000111
		0, 0, 0, 0, 0, 1, 1, 1,
	}

	inputBuffer := bytes.NewBuffer(input)
	dec := newBitDecoder(inputBuffer, 2)

	output := []byte{}
	for {
		bit, err := dec.readBit()
		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("failed to read bit: %s", err)
		}

		output = append(output, byte(bit))
	}

	if !reflect.DeepEqual(output, expectedOutput) {
		t.Fatalf("failed to read bits, expected %v, got %v",
			expectedOutput, output)
	}
}
