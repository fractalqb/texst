package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fractalqb/texst"
	"github.com/fractalqb/texst/texsting"
)

type prepareCmd struct {
	texst.Prepare
	suffix string
	force  bool
}

var cmdPrepare = prepareCmd{
	suffix: texsting.StdSuffix,
}

func (cmd *prepareCmd) usage(flags *flag.FlagSet) func() {
	return func() {
		w := flags.Output()
		fmt.Fprint(w, `Prepare a reference text file from an example subject

Usage: texst prepare [flags] <subject>...

FLAGS
`)
		flags.PrintDefaults()
	}
}

func (cmd *prepareCmd) run(args []string) {
	args = cmd.flags(args)
	cmd.prepareFiles(args)
}

func (cmd *prepareCmd) flags(args []string) []string {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.Usage = cmd.usage(flags)
	flags.StringVar(&cmd.suffix, "s", cmd.suffix,
		`Set file suffix for created reference text files`,
	)
	flags.BoolVar(&cmd.force, "f", cmd.force,
		`Force to overwrite existing reference files`,
	)
	flags.Parse(args[1:])
	return flags.Args()
}

func (cmd *prepareCmd) prepareFiles(files []string) {
	if len(files) == 0 {
		cmd.Text(os.Stdout, os.Stdin)
	} else {
		for _, f := range files {
			cmd.prepareFile(f)
		}
	}
}

func (cmd *prepareCmd) prepareFile(name string) {
	texstfile := name + cmd.suffix
	if _, err := os.Stat(texstfile); !os.IsNotExist(err) {
		if !cmd.force {
			log.Fatalf("%s already exists", texstfile)
		}
	}
	rd, _ := os.Open(name)
	defer rd.Close()
	wr, _ := os.Create(texstfile)
	defer wr.Close()
	cmd.Text(wr, rd)
}
