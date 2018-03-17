package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("missing file name")
		os.Exit(1)
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("failed to open file: ", err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Printf("reading file: %s\n", fileName)

	c := newCompressor()

	begin := time.Now()

	if err := c.analyse(file); err != nil {
		fmt.Println("failed to analyse file: ", err)
		os.Exit(1)
	}
	fmt.Printf("analysing phase done in %s\n", time.Since(begin))

	start := time.Now()
	if err := c.buildTree(); err != nil {
		fmt.Println("failed to build tree: ", err)
		os.Exit(1)
	}
	fmt.Printf("building tree phase done in %s\n", time.Since(start))

	start = time.Now()
	if err := c.buildTable(); err != nil {
		fmt.Println("failed to build table: ", err)
		os.Exit(1)
	}
	fmt.Printf("building table phase done in %s\n", time.Since(start))

	output, err := os.Create("output.gc")
	if err != nil {
		fmt.Println("failed to create output file : ", err)
		os.Exit(1)
	}
	defer output.Close()

	start = time.Now()
	if err := c.compress(file, output); err != nil {
		fmt.Println("failed to compress file : ", err)
		os.Exit(1)
	}
	fmt.Printf("compressing file done in %s\n", time.Since(start))

	fmt.Printf("input size: %d\n", c.inputSize)
	fmt.Printf("output size: %d\n", c.outputSize)

	compressionFactor := (float64(c.outputSize) * 100) / float64(c.inputSize)
	fmt.Printf("compression factor: %.02f%%\n", compressionFactor)

	fmt.Printf("done in %s\n", time.Since(begin))
}
