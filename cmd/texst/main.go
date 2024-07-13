// A command line tool to use text tests
package main

import (
	"flag"
	"fmt"
	"log"
)

func usage() {
	w := flag.CommandLine.Output()
	fmt.Fprint(w, `A command line tool to use text tests

Usage: texst [flags] <command> <arg>...

COMMANDS
   prepare: Prepare a reference file
   compare: Compare a reference file with subjects

TEXST FORMAT

Maks Types:
   . Same length as mask or match a regexp
   * Length 0 or more
   + Length 1 or more
   0 Length 0 up to length of mask
   1 Length 1 up to length of mask
   - At least as long as mask

Preamble Lines:
   %%<interleaving groups>
   *_<global masks> where _ is a mask type

Reference Lines:
   >g<actual reference text> of interleaving group g
    _<mask definitions> where _ is a mask type
    ?m <char class> Set character class for non-regexp masks m
    ~m <regexp> Mask m matches <regexp>
`)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
		return
	}
	switch flag.Arg(0) {
	case "prepare":
		cmdPrepare.run(flag.Args())
	case "compare":
		cmdCompare.run(flag.Args())
	default:
		log.Fatalf("unknown command '%s'", flag.Arg(0))
	}
}
