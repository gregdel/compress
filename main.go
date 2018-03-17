package main

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type app struct {
	cli *cli.App
	log *logrus.Logger
}

func main() {
	app := &app{
		cli: cli.NewApp(),
		log: logrus.New(),
	}

	// Log format
	app.log.Formatter = &logrus.TextFormatter{DisableTimestamp: true}

	app.cli.Name = "compress"
	app.cli.Description = "compress"
	app.cli.Usage = app.cli.Description
	app.cli.HideVersion = true
	app.cli.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "log everything",
		},
	}

	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "input, i",
			Value: "",
			Usage: "input file",
		},
		cli.StringFlag{
			Name:  "output, o",
			Value: "out.txt",
			Usage: "output file",
		},
	}

	app.cli.Commands = []cli.Command{
		{
			Name:   "compress",
			Usage:  "compress a file",
			Action: app.compress,
			Flags:  flags,
		},
	}

	app.cli.Run(os.Args)
}

func (app *app) compress(c *cli.Context) {
	verbose := c.GlobalBool("verbose")
	if verbose {
		app.log.SetLevel(logrus.DebugLevel)
	} else {
		app.log.SetLevel(logrus.WarnLevel)
	}

	inputFile := c.String("input")
	if inputFile == "" {
		app.log.Fatal("missing input file")
	}
	outputFile := c.String("output")

	file, err := os.Open(inputFile)
	if err != nil {
		app.log.Fatalf("failed to open file: %s", err)
	}
	defer file.Close()

	app.log.Debugf("reading file: %s", inputFile)

	comp := newCompressor(app.log)

	begin := time.Now()

	if err := comp.analyse(file); err != nil {
		app.log.Fatalf("failed analyse file: %s", err)
	}
	app.log.Debugf("analysing phase done in %s", time.Since(begin))

	start := time.Now()
	if err := comp.buildTree(); err != nil {
		app.log.Fatalf("failed to build tree: %s", err)
	}
	app.log.Debugf("building tree phase done in %s", time.Since(start))

	start = time.Now()
	if err := comp.buildTable(); err != nil {
		app.log.Fatalf("failed to build table: %s", err)
	}
	app.log.Debugf("building table phase done in %s", time.Since(start))

	output, err := os.Create(outputFile)
	if err != nil {
		app.log.Debugf("failed to create output file : ", err)
		os.Exit(1)
	}
	defer output.Close()

	start = time.Now()
	if err := comp.compress(file, output); err != nil {
		app.log.Fatalf("failed to compress file : %s", err)
	}
	app.log.Debugf("compressing file done in %s", time.Since(start))

	app.log.Debugf("input size: %d", comp.inputSize)
	app.log.Debugf("output size: %d", comp.outputSize)

	app.log.Debugf("done in %s", time.Since(begin))
}
