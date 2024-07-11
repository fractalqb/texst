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
