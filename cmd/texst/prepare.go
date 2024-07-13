package main

import (
	"errors"
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
	output string
	force  bool
}

var cmdPrepare = prepareCmd{
	Prepare: texst.Prepare{
		DefaultIGroup: ' ',
	},
	suffix: texsting.StdSuffix,
}

func (cmd *prepareCmd) usage(flags *flag.FlagSet) func() {
	return func() {
		w := flags.Output()
		fmt.Fprint(w, `Prepare a reference text file from an example subject

Usage: texst prepare [flags] [<subject>...]

Prepare reads from stdin when no subjects are given. When processing more than
one subject without explicit output name, the output file names are generated by
appending the suffix to the subject name.

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
	flags.StringVar(&cmd.output, "o", cmd.output,
		`Set output file name`,
	)
	flags.BoolVar(&cmd.force, "f", cmd.force,
		`Force to overwrite existing reference files`,
	)
	flags.Parse(args[1:])
	return flags.Args()
}

func (cmd *prepareCmd) prepareFiles(files []string) {
	switch len(files) {
	case 0:
		if cmd.output == "" {
			cmd.Text(os.Stdout, os.Stdin)
		} else if out := cmd.createOut(cmd.output); out == nil {
			log.Fatalf("output '%s' already exists", cmd.output)
		} else {
			defer out.Close()
			cmd.Text(out, os.Stdin)
		}
	case 1:
		if cmd.output == "" {
			cmd.prepareFile(files[0], os.Stdout)
		} else if out := cmd.createOut(cmd.output); out == nil {
			log.Fatalf("output '%s' already exists", cmd.output)
		} else {
			defer out.Close()
			cmd.prepareFile(files[0], out)
		}
	default:
		if cmd.output == "" {
			for _, f := range files {
				cmd.prepareFile(f, nil)
			}
		} else if out := cmd.createOut(cmd.output); out == nil {
			log.Fatalf("output '%s' already exists", cmd.output)
		} else {
			defer out.Close()
			for _, file := range files {
				cmd.prepareFile(file, out)
			}
		}
	}
}

func (cmd *prepareCmd) createOut(name string) *os.File {
	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		return out
	} else if err != nil {
		log.Fatal(err)
	}
	if !cmd.force {
		return nil
	}
	out, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func (cmd *prepareCmd) prepareFile(name string, wr *os.File) {
	rd, _ := os.Open(name)
	defer rd.Close()
	if wr == nil {
		if cmd.suffix == "" {
			log.Fatalf("cannot generate texst-file name for %s", name)
		}
		texstfile := name + cmd.suffix
		wr = cmd.createOut(texstfile)
		if wr == nil {
			log.Fatalf("output '%s' already exists", texstfile)
		}
		defer wr.Close()
	}
	cmd.Text(wr, rd)
}
