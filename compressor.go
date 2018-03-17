package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"sort"
)

type encodedChar struct {
	Size uint8
	Char uint8
}

type compressor struct {
	inputSize  int
	outputSize int
	headerSize int

	stats map[byte]int

	table map[byte]encodedChar

	treeRoot *node
}

type node struct {
	weight int
	// A node might contain a byte
	char byte
	leaf bool
	// Value
	value       uint8
	valueLength uint8
	// A node might have child nodes
	lChild *node
	rChild *node
}

func newCompressor() *compressor {
	return &compressor{
		stats: map[byte]int{},
		table: map[byte]encodedChar{},
	}
}

func (c *compressor) analyse(r io.Reader) error {
	buffer := make([]byte, 1)
	for {
		n, err := r.Read(buffer)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read buffer : %s", err)
		}

		c.stats[buffer[0]]++
		c.inputSize += n
	}
}

func (c *compressor) buildTree() error {
	nodes := make([]*node, len(c.stats))

	i := 0
	for b, c := range c.stats {
		nodes[i] = &node{
			weight: c,
			char:   b,
			leaf:   true,
		}
		i++
	}

	for {
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].weight < nodes[j].weight })

		switch len(nodes) {
		case 0:
			return fmt.Errorf("invliad node length")
		case 1:
			c.treeRoot = nodes[0]
			return nil
		default:
			l := nodes[0]
			r := nodes[1]
			newNode := &node{
				weight: l.weight + r.weight,
				rChild: r,
				lChild: l,
			}
			nodes[1] = newNode
			nodes = nodes[1:]
		}
	}
}

func (c *compressor) buildTable() error {
	c.treeRoot.value = 0x1
	c.treeRoot.valueLength = 1
	return c.exploreSubtree(c.treeRoot)
}

func (c *compressor) exploreSubtree(root *node) error {
	// Leaf
	if root.leaf {
		c.table[root.char] = encodedChar{
			Char: root.value,
			Size: root.valueLength,
		}
		return nil
	}

	// Explore left
	root.lChild.value = root.value << 1
	root.lChild.valueLength = root.valueLength + 1
	if err := c.exploreSubtree(root.lChild); err != nil {
		return err
	}

	// Explore right
	root.rChild.value = (root.value << 1) | 1
	root.rChild.valueLength = root.valueLength + 1
	return c.exploreSubtree(root.rChild)
}

func (c *compressor) compress(r io.ReadSeeker, w io.Writer) error {
	// Encode the headers
	var headers bytes.Buffer
	enc := gob.NewEncoder(&headers)
	if err := enc.Encode(c.table); err != nil {
		return err
	}
	headersLen := headers.Len()

	// Write the table length in a uint64
	h := make([]byte, 8)
	binary.LittleEndian.PutUint64(h, uint64(headersLen))
	n, err := w.Write(h)
	if err != nil {
		return err
	}
	c.outputSize += n
	fmt.Printf("header size indicator: %d\n", n)

	// Write the table
	s, err := headers.WriteTo(w)
	if err != nil {
		return err
	}
	c.outputSize += int(s)
	if s != int64(headersLen) {
		return fmt.Errorf("len missmatch %d / %d", s, headersLen)
	}
	fmt.Printf("header size: %d\n", s)

	// Reset the file seek
	if _, err := r.Seek(io.SeekStart, io.SeekStart); err != nil {
		return err
	}

	// Bit encoder
	inputBuffer := make([]byte, 1)
	bitEncoder := newBitEncoder(w, 4)
	for {
		_, err := r.Read(inputBuffer)
		if err == io.EOF {
			if err := bitEncoder.flush(); err != nil {
				return err
			}
			c.outputSize += bitEncoder.totalEncoded()

			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read buffer : %s", err)
		}

		err = bitEncoder.writeBits(
			c.table[inputBuffer[0]].Char, c.table[inputBuffer[0]].Size)
		if err != nil {
			return err
		}
	}
}
