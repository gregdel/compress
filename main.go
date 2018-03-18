package main

import (
	"os"

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
		{
			Name:   "decompress",
			Usage:  "decompress a file",
			Action: app.decompress,
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

	output, err := os.Create(outputFile)
	if err != nil {
		app.log.Fatalf("failed to create output file : %s", err)
	}
	defer output.Close()

	comp := newCompressor(app.log, 1024)
	if err := comp.compress(file, output); err != nil {
		app.log.Fatalf("failed to compress file : %s", err)
	}
}

// decompress a file
func (app *app) decompress(c *cli.Context) {
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

	input, err := os.Open(inputFile)
	if err != nil {
		app.log.Fatalf("failed to open file: %s", err)
	}
	defer input.Close()

	output, err := os.Create(outputFile)
	if err != nil {
		app.log.Fatalf("failed to create output file : %s", err)
	}
	defer output.Close()

	comp := newCompressor(app.log, 8)
	if err := comp.decompress(input, output); err != nil {
		app.log.Fatalf("failed to decompress file : %s", err)
	}

	app.log.Debugf("reading file: %s", inputFile)
}
