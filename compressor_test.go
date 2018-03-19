package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

// Ensure that we can rebuild the same tree with the headers only
func TestBuildTreeFromHeaders(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	comp := newCompressor(log, 1)

	// Input stats
	comp.stats = map[byte]uint64{
		0x1: 10,
		0x2: 10,
		0x3: 10,
		0x4: 5,
		0x6: 2,
		0x7: 1,
	}
	if err := comp.buildTree(); err != nil {
		t.Fatalf("failed to build tree: %s", err)
	}
	outputTree := comp.treeRoot

	if err := comp.buildTable(); err != nil {
		t.Fatalf("failed to build table: %s", err)
	}
	outputTable := comp.table

	// Build headers
	comp.buildHeaders()

	// Empty the tree and the table
	comp.treeRoot = nil
	comp.table = map[byte]encodedChar{}

	// Build the tree from the headers
	if err := comp.buildTreeFromHeaders(); err != nil {
		t.Fatalf("failed to build tree from headers: %s", err)
	}

	// Build the table
	if err := comp.buildTable(); err != nil {
		t.Fatalf("failed to build table: %s", err)
	}

	// Compare the trees
	if !reflect.DeepEqual(comp.treeRoot, outputTree) {
		t.Fatalf("failed to rebuild tree, want %v, got %v",
			outputTree, comp.treeRoot)
	}

	// Compare the tables
	if !reflect.DeepEqual(comp.table, outputTable) {
		t.Fatalf("failed to rebuild tree, want %v, got %v",
			outputTable, comp.table)
	}
}

func TestGetCharFromTree(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	comp := newCompressor(log, 1)

	// Tree values
	// 1 => 00
	// 2 => 01
	// 3 => 10
	// 4 => 110
	// 5 => 111
	comp.treeRoot = &node{
		lChild: &node{
			lChild: &node{char: 1, leaf: true},
			rChild: &node{char: 2, leaf: true},
		},
		rChild: &node{
			lChild: &node{char: 3, leaf: true},
			rChild: &node{
				lChild: &node{char: 4, leaf: true},
				rChild: &node{char: 5, leaf: true},
			},
		},
	}

	// 12345 => 00011011 01110000
	input := bytes.NewBuffer([]byte{
		0x1b, 0x70,
	})
	expectedOutput := []byte{1, 2, 3, 4, 5}

	output := &bytes.Buffer{}
	// Expected output size: 5
	bitDecoder := newBitDecoder(input, 2, 5)

	if err := bitDecoder.decode(output, comp); err != nil {
		t.Fatalf("failed to decode: %s", err)
	}

	if !reflect.DeepEqual(output.Bytes(), expectedOutput) {
		t.Fatalf("unexpected output, wanted %v, got %v",
			expectedOutput, output.Bytes())
	}
}

func TestCompressDecompress(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	// First sentense of the bible
	inputStr := `
      1:1 In the beginning God created the heaven and the earth.

      1:2 And the earth was without form, and void; and darkness was upon
      the face of the deep. And the Spirit of God moved upon the face of the
      waters.
	`

	input := bytes.NewReader([]byte(inputStr))

	compressed := &bytes.Buffer{}
	comp := newCompressor(log, 8)
	if err := comp.compress(input, compressed); err != nil {
		t.Fatalf("failed to compress input: %s", err)
	}

	decompressed := &bytes.Buffer{}
	dec := newCompressor(log, 8)
	if err := dec.decompress(compressed, decompressed); err != nil {
		t.Fatalf("failed to decompress input: %s", err)
	}

	if decompressed.String() != inputStr {
		t.Fatalf("failed to decompress data\nwanted:'%s'\ngot:'%s'",
			inputStr, decompressed.String())
	}
}
