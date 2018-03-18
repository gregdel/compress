package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"sort"

	"github.com/sirupsen/logrus"
)

type encodedChar struct {
	char uint64
	// Max 9
	size uint8
}

type headerMember struct {
	Char  uint8
	Count uint64
}

type compressor struct {
	// Size of the buffers to use
	buffersSize int

	inputSize  int
	outputSize int
	headerSize int

	stats   map[byte]uint64
	headers []headerMember

	table map[byte]encodedChar

	treeRoot *node

	// logger
	log *logrus.Logger
}

type node struct {
	weight uint64
	// A node might contain a byte
	char byte
	leaf bool
	// Value
	value       uint64
	valueLength uint8
	// A node might have child nodes
	lChild *node
	rChild *node
}

func newCompressor(log *logrus.Logger, buffersSize int) *compressor {
	return &compressor{
		log:         log,
		stats:       map[byte]uint64{},
		table:       map[byte]encodedChar{},
		buffersSize: buffersSize,
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
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].weight == nodes[j].weight {
				return nodes[i].char < nodes[j].char
			}
			return nodes[i].weight < nodes[j].weight
		})

		switch len(nodes) {
		case 0:
			return fmt.Errorf("invalid node length")
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
			char: root.value,
			size: root.valueLength,
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

func (c *compressor) buildHeaders() {
	c.headers = []headerMember{}
	for b, v := range c.stats {
		c.headers = append(c.headers, headerMember{
			Char:  b,
			Count: v,
		})
	}
}

func (c *compressor) buildTreeFromHeaders() error {
	// Builds the stats
	for _, v := range c.headers {
		c.stats[v.Char] = v.Count
	}

	// Build the tree
	return c.buildTree()
}

func (c *compressor) compress(r io.ReadSeeker, w io.Writer) error {
	// Analyse the file
	if err := c.analyse(r); err != nil {
		return err
	}

	// Build the tree
	if err := c.buildTree(); err != nil {
		return err
	}

	// Build the table
	if err := c.buildTable(); err != nil {
		return err
	}

	// Build the headers
	c.buildHeaders()

	// Encode the headers
	var headers bytes.Buffer
	enc := gob.NewEncoder(&headers)
	if err := enc.Encode(c.headers); err != nil {
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
	c.log.Debugf("header size indicator: %d", n)

	// Write the table
	s, err := headers.WriteTo(w)
	if err != nil {
		return err
	}
	c.outputSize += int(s)
	if s != int64(headersLen) {
		return fmt.Errorf("len missmatch %d / %d", s, headersLen)
	}
	c.log.Debugf("header size: %d", s)

	// Reset the file seek
	if _, err := r.Seek(io.SeekStart, io.SeekStart); err != nil {
		return err
	}

	// Bit encoder
	inputBuffer := make([]byte, 1)
	bitEncoder := newBitEncoder(w, c.buffersSize)
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
			c.table[inputBuffer[0]].char, c.table[inputBuffer[0]].size)
		if err != nil {
			return err
		}
	}
}

func (c *compressor) decompress(r io.Reader, w io.Writer) error {
	// Read a uint64 fot the header size
	// Read header size and decompress
	// Read the rest of the file and go through the tree
	// Write the table length in a uint64
	headersLenBuffer := make([]byte, 8)
	n, err := r.Read(headersLenBuffer)
	if err != nil {
		return err
	}
	c.outputSize += n
	headersLen := binary.LittleEndian.Uint64(headersLenBuffer)

	c.log.Debugf("header size indicator: %d", headersLen)

	statsBuffer := make([]byte, headersLen)
	n, err = r.Read(statsBuffer)
	if err != nil {
		return err
	}
	c.outputSize += n

	// Encode the headers
	dec := gob.NewDecoder(bytes.NewBuffer(statsBuffer))
	if err := dec.Decode(&c.headers); err != nil {
		return err
	}

	// Build the tree
	if err := c.buildTreeFromHeaders(); err != nil {
		return err
	}

	return c.decode(r, w)
}

func (c *compressor) decode(r io.Reader, w io.Writer) error {
	// Go through the tree and decode
	bitDecoder := newBitDecoder(r, c.buffersSize)
	for {
		// Read a bit from the input
		char, err := c.getCharFromTree(bitDecoder)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		// Write the char
		_, err = w.Write([]byte{char})
		if err != nil {
			return err
		}
	}
}

func (c *compressor) getCharFromTree(bd *bitDecoder) (byte, error) {
	node := c.treeRoot
	bit, err := bd.readBit()
	if err != nil {
		return 0, err
	}
	if bit == 0 {
		return 0, io.EOF
	}

	for {
		bit, err = bd.readBit()
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
