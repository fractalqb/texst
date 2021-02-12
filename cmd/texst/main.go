package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fractalqb/texst"
)

type errLog string

var (
	fPrepare bool
	fClean   bool
	fRefFile string
)

func prepare(rd io.Reader, wr io.Writer) {
	prefix := []byte{'>', ' '}
	scn := bufio.NewScanner(rd)
	for scn.Scan() {
		wr.Write(prefix)
		wr.Write(scn.Bytes())
		fmt.Fprintln(wr)
	}
}

func prepareFile(name string) {
	rd, _ := os.Open(name)
	defer rd.Close()
	wr, _ := os.Create(name + ".texst")
	defer wr.Close()
	prepare(rd, wr)
}

func prepareFiles(args []string) {
	if len(args) == 0 {
		prepare(os.Stdin, os.Stdout)
	} else {
		for _, arg := range args {
			prepareFile(arg)
		}
	}
}

func checkFile(ref, subj string) bool {
	rr, err := os.Open(ref)
	if err != nil {
		log.Fatal(err)
	}
	defer rr.Close()
	sr, err := os.Open(subj)
	if err != nil {
		log.Fatal(err)
	}
	defer sr.Close()
	var cmpr texst.Compare
	err = cmpr.Readers(rr, sr, func(n int, l string, rs []*texst.RefLine) bool {
		log.Printf("missmatch #%d:%s\n", n, l)
		for _, r := range rs {
			log.Printf("- ref '%c':%s\n", r.IGroup(), r.Text())
		}
		return false
	})
	if err == nil {
		log.Printf("%s matches reference %s\n", subj, ref)
		return true
	} else {
		log.Printf("%s mismatch with %s: %s", subj, ref, err)
		return false
	}
}

func usage() {
	wr := flag.CommandLine.Output()
	fmt.Fprintf(wr, "Usage of %s (v%d.%d.%d-%s+%d):\n",
		os.Args[0],
		texst.VMajor, texst.VMinor, texst.VPatch,
		texst.VQuality,
		texst.VBuild,
	)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.BoolVar(&fPrepare, "p", false, "Prepare reference files")
	flag.BoolVar(&fClean, "c", false, "Clean reference files")
	flag.StringVar(&fRefFile, "r", "", "Set reference file")
	flag.Parse()
	if fPrepare {
		prepareFiles(flag.Args())
		return
	} else if fClean {
		return
	}
	miss := 0
	for _, arg := range flag.Args() {
		if !checkFile(fRefFile, arg) {
			miss++
		}
	}
	if miss > 0 {
		os.Exit(1)
	}
}
