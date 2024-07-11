package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/TwiN/go-color"
	"github.com/fractalqb/texst"
	"golang.org/x/term"
)

type compareCmd struct {
	mlim       int
	showRegexp bool
}

var cmdCompare compareCmd

func (cmd *compareCmd) usage(flags *flag.FlagSet) func() {
	return func() {
		w := flags.Output()
		fmt.Fprint(w, `Compare a reference text file to subject files

Usage: texst compare [flags] <reference> <subject>...

FLAGS
`)
		flags.PrintDefaults()
	}
}

func (cmd *compareCmd) run(args []string) {
	args = cmd.flags(args)
	if len(args) == 0 {
		log.Fatal("no reference file")
	}
	cmd.checkFiles(args[0], args[1:])
}

func (cmd *compareCmd) flags(args []string) []string {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.Usage = cmd.usage(flags)
	flags.IntVar(&cmd.mlim, "l", cmd.mlim,
		`Set mismatch limit`,
	)
	flags.BoolVar(&cmd.showRegexp, "m", cmd.showRegexp,
		`Show regular expression of mismatching reference lines`,
	)
	flags.Parse(args[1:])
	return flags.Args()
}

func (cmd *compareCmd) checkFiles(ref string, files []string) {
	if len(files) == 0 {
		cmd.checkRd(ref, "stdin", os.Stdin)
	}
	for _, f := range files {
		cmd.checkFile(ref, f)
	}
}

func (cmd *compareCmd) checkFile(ref, subj string) bool {
	sr, err := os.Open(subj)
	if err != nil {
		log.Fatal(err)
	}
	defer sr.Close()
	return cmd.checkRd(ref, subj, sr)
}

func (cmd *compareCmd) checkRd(ref, sname string, subj io.Reader) bool {
	cmpr := texst.Texst{
		MismatchLimit: cmd.mlim,
		OnMismatch:    cmd.onMismatch,
	}
	rrd, err := texst.OpenRefFile(ref)
	if err != nil {
		log.Println(err)
		return false
	}
	defer rrd.Close()
	if mis, err := cmpr.Check(rrd, subj); err != nil {
		log.Printf("check error: %s", err)
		return false
	} else if mis > 0 {
		log.Printf("%s has %d mismatches with %s", sname, mis, ref)
		return false
	}
	log.Printf("%s matches reference %s\n", sname, ref)
	return true
}

func (cmd *compareCmd) onMismatch(n int, l []byte, ref []*texst.RefLine) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "missmatch in line %d:", n)
	txtCol := sb.Len()
	if term.IsTerminal(int(os.Stderr.Fd())) {
		fmt.Fprintf(&sb, " [%s]", l)
	} else {
		fmt.Fprintf(&sb, " [%s]", l)
	}
	log.Print(sb.String())
	for _, r := range ref {
		sb.Reset()
		fmt.Fprintf(&sb, "ref:%d", r.SourceLine())
		if cmd.showRegexp {
			log.Printf("%s%s '%c' ~ %s",
				strings.Repeat(" ", txtCol-sb.Len()-4),
				sb.String(),
				r.IGroup(),
				r.Regexp(),
			)
		} else {
			log.Printf("%s%s '%c' [%s]",
				strings.Repeat(" ", txtCol-sb.Len()-4),
				sb.String(),
				r.IGroup(),
				withMasks(r, l),
			)
		}
	}
}

func withMasks(rl *texst.RefLine, sl []byte) string {
	segs := rl.Segments()
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return rl.Text()
	}
	var sb strings.Builder
	txt := rl.Text()
	rpt := 0
	for _, seg := range segs {
		if rpt < seg.Start() {
			part := diffPart(string(sl[rpt:]), txt[rpt:seg.Start()])
			sb.WriteString(part)
		}
		rpt = seg.Start() + seg.Len()
		sb.WriteString(color.InGray(txt[seg.Start():rpt]))
	}
	if rpt < len(txt) {
		if rpt < len(sl) {
			part := diffPart(string(sl[rpt:]), txt[rpt:])
			sb.WriteString(part)
		} else {
			sb.WriteString(txt[rpt:])
		}
	}
	return sb.String()
}

func diffPart(sp, rp string) string {
	var (
		isEQ bool
		sb   strings.Builder
	)
	sb.WriteString(color.Bold)
	for i, sr := range sp {
		if rp == "" {
			sb.WriteString(color.Reset)
			return sb.String()
		}
		rr, rsz := utf8.DecodeRuneInString(rp)
		rp = rp[rsz:]
		if i == 0 || isEQ != (sr == rr) {
			isEQ = sr == rr
			if isEQ {
				sb.WriteString(color.Reset)
				sb.WriteString(color.Bold)
				sb.WriteString(color.Green)
			} else {
				sb.WriteString(color.Red)
				sb.WriteString(color.Underline)
			}
		}
		sb.WriteRune(rr)
	}
	sb.WriteString(color.Reset)
	sb.WriteString(color.Bold)
	sb.WriteString(rp)
	sb.WriteString(color.Reset)
	return sb.String()
}
