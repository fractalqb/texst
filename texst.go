package texst

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"git.fractalqb.de/fractalqb/icontainer"
)

// Line Tags
const (
	// Marks a comment line.
	TagComment = '#'
	// Preamble lines must be the first lines of a reference text specification.
	TagPreamble = '\\'
	// Global segment lines set/clear file-global tags.
	TagGlobalSeg = '*'
	// Reference lines have the text that is compared to the subject text.
	TagRefLine = '>'
	// Argument lines apply to the most recent '>' reference line up to the next
	// non-argument line.
	TagRefArgs = ' '

	// TagGSegPush = '+'
	// TagGSegPop  = '-'
)

// Types of preamble lines
const (
	// Define interleaving groups in the preamble.
	PreIGroups = '%'
)

// MismatchFunc is called for each mismatch in the subject text during
// comparison. It gets the respective line number 'slineno' in the subject
// file, the text line 'sline' and the reference lines of each interleaving
// group that were matched against the subject line.
//
// If the MismatchFunc returns 'abort' == true the comparison terminates
// immediately.
type MismatchFunc func(slineno int, sline string, refs []*RefLine) (abort bool)

// Compare performs the comparison of a subject text against a reference text
// specification. A zero value is valid for use and can be reused for
// more than one comparison. It must not be used concurrently.
type Compare struct {
	// Specifies the number of detected mismatches after which the comparison
	// is aborted. If MismatchLimit == 0, do not abort.
	MismatchLimit int
	// OnMismatch is called on each detected mismatch
	OnMismatch MismatchFunc

	igroupLs  []rune
	igoupRefs map[rune]*icontainer.SBList[*RefLine]
	refLine   int
	sbjLine   int
	gmasks    map[rune]*maskDefns
}

func (cmpr *Compare) isIGroup(r rune) bool {
	for _, g := range cmpr.igroupLs {
		if g == r {
			return true
		}
	}
	return false
}

// MismatchCount is the error used to report the total number of mismatches
// detected during a Compare run.
type MismatchCount int

func (mc MismatchCount) Error() string {
	if mc == 1 {
		return "1 mismatch"
	}
	return fmt.Sprintf("%d mismatches", mc)
}

// RefError is returned for errors during processing of the reference file.
type RefError struct {
	Line int
	err  error
}

func (e RefError) Error() string {
	return fmt.Sprintf("ref %d:%s", e.Line, e.err)
}

func (e RefError) Unwrap() error { return e.err }

// SubjError is returned for errors during processing of the subject file.
type SubjError struct {
	Line int
	err  error
}

func (e SubjError) Error() string {
	return fmt.Sprintf("subj %d:%s", e.Line, e.err)
}

func (e SubjError) Unwrap() error { return e.err }

// Readers compares the reference text and subject text from the io.Readers
// 'ref' and 'subj'. If 'onmiss' is not nil it will be called on each detected
// mismatch. The number of detected mismatches will be returned as
// MismatchCount error or as nil if no mismatch and no other error occurs.
// Errors regarding read operations or syntax errors in 'ref' or 'subj' will
// terminate the comparison immediately and be returned as RefError or
// SubjError, depending on the source of error.
func (cmpr *Compare) Readers(ref, subj io.Reader) error {
	var rr *bufio.Reader
	if tmp, ok := ref.(*bufio.Reader); ok {
		rr = tmp
	} else {
		rr = bufio.NewReader(ref)
	}
	sr := bufio.NewScanner(subj)
	err := cmpr.cmpr(rr, sr)
	return err
}

// Strings compares the reference text and subject text from the strings
// 'ref' and 'subj'. For more detail read Readers documentation.
func (cmpr *Compare) Strings(ref, subj string) error {
	return cmpr.Readers(
		strings.NewReader(ref),
		strings.NewReader(subj),
	)
}

func (cmpr *Compare) RefFile(refname string, subj io.Reader) error {
	rd, err := os.Open(refname)
	if err != nil {
		return err
	}
	defer rd.Close()
	return cmpr.Readers(rd, subj)
}

func (cmpr *Compare) cmpr(ref *bufio.Reader, subj *bufio.Scanner) (err error) {
	cmpr.refLine = 0
	cmpr.sbjLine = 0
	cmpr.igroupLs = nil
	if err = cmpr.readPreamble(ref); err != nil {
		return RefError{Line: cmpr.refLine, err: err}
	}
	cmpr.igoupRefs = make(map[rune]*icontainer.SBList[*RefLine])
	cmpr.gmasks = make(map[rune]*maskDefns)
	defer func() {
		cmpr.igoupRefs = nil
		cmpr.gmasks = nil
	}()
	if err = cmpr.readGlobals(ref); err != nil {
		return RefError{Line: cmpr.refLine, err: err}
	}
	if len(cmpr.igroupLs) == 0 {
		rl := newRefLine()
		if err = rl.read(ref, cmpr.gmasks, &cmpr.refLine); err != nil {
			return RefError{Line: cmpr.refLine, err: err}
		}
		cmpr.igroupLs = []rune{rl.igroup}
		cmpr.igoupRefs[rl.igroup] = icontainer.NewSBList[*RefLine](rl)
	}
	misses := 0
SCAN_NEXT_LINE:
	for subj.Scan() {
		if err = subj.Err(); err != nil {
			return SubjError{Line: cmpr.sbjLine, err: err}
		}
		cmpr.sbjLine++
		sline := subj.Text()
		for _, r := range cmpr.igroupLs {
			rl, err := cmpr.nextRefLine(r, ref)
			if err != nil {
				return RefError{Line: cmpr.refLine, err: err}
			}
			if rl != nil && rl.matches(sline) == nil {
				cmpr.dropRefLine(rl)
				continue SCAN_NEXT_LINE
			}
		}
		misses++
		if cmpr.OnMismatch != nil {
			if cmpr.OnMismatch(cmpr.sbjLine, sline, cmpr.currentRefs()) {
				break SCAN_NEXT_LINE
			}
		}
		if cmpr.MismatchLimit > 0 && misses >= cmpr.MismatchLimit {
			break SCAN_NEXT_LINE
		}
	}
	if misses > 0 {
		return MismatchCount(misses)
	}
	return nil
}

func (cmpr *Compare) currentRefs() []*RefLine {
	var res []*RefLine
	for _, nm := range cmpr.igroupLs {
		rls := cmpr.igoupRefs[nm]
		if rls != nil && rls.Len() > 0 {
			res = append(res, rls.Front())
		}
	}
	return res
}

func (cmpr *Compare) nextRefLine(igroup rune, rd *bufio.Reader) (*RefLine, error) {
	ls := cmpr.igoupRefs[igroup]
	if ls == nil || ls.Len() == 0 {
	READ_LOOP:
		for {
			tag, err := nextTag(rd)
			if err != nil {
				return nil, err
			}
			switch tag {
			case TagRefLine:
				rl := newRefLine()
				if err = rl.read(rd, cmpr.gmasks, &cmpr.refLine); err != nil {
					return nil, err
				}
				if ls = cmpr.igoupRefs[rl.igroup]; ls == nil {
					ls = icontainer.NewSBList(rl)
					cmpr.igoupRefs[rl.igroup] = ls
				} else {
					ls.PushBack(rl)
				}
				if rl.igroup == igroup {
					break READ_LOOP
				}
			case 0:
				return nil, nil
			default:
				return nil, fmt.Errorf("syntax error: unexpected line tag '%c'", tag)
			}
		}
	}
	res := ls.Front()
	return res, nil
}

func (cmpr *Compare) dropRefLine(rl *RefLine) {
	ls := cmpr.igoupRefs[rl.igroup]
	if ls == nil {
		panic("dropRefLine: no igroup list")
	}
	if ls.Front() != rl {
		panic("dropRefLine: igroup list-front missmatch")
	}
	ls.DropFront(1)
}

func (cmpr *Compare) readPreamble(ref *bufio.Reader) error {
	return eachTagLine(ref, &cmpr.refLine, tags(TagPreamble), func(line string) error {
		if len(line) < 2 {
			return errors.New("incomplete preamble line")
		}
		switch line[1] {
		case PreIGroups:
			cmpr.igroupLs = []rune(line[2:])
		default:
			return fmt.Errorf("unknown preamble tag: '%c'", line[1])
		}
		return nil
	})
}

func (cmpr *Compare) readGlobals(ref *bufio.Reader) error {
	return eachTagLine(ref, &cmpr.refLine, globalsFilter, func(line string) error {
		l := utf8.RuneCountInString(line)
		if l < 2 {
			return errors.New("incomplete global line")
		}
		tag, rsz := utf8.DecodeRuneInString(line)
		switch tag {
		case TagGlobalSeg:
			segs := cmpr.gmasks[0]
			if segs == nil {
				segs = new(maskDefns)
				cmpr.gmasks[0] = segs
			}
			if rep, err := segs.set(line[rsz:]); err != nil {
				return err
			} else if rep != "" {
				return errors.New("redefining global mask line") // TODO more specific?
			}
		default:
			if !cmpr.isIGroup(tag) {
				return fmt.Errorf("unknown global tag: '%c'", line[1])
			}
			segs := cmpr.gmasks[tag]
			if segs == nil {
				segs = new(maskDefns)
				cmpr.gmasks[tag] = segs
			}
			if rep, err := segs.set(line[rsz:]); err != nil {
				return err
			} else if rep != "" {
				return errors.New("redefining global mask line") // TODO more specific?
			}
		}
		return nil
	})
}

type maskDefns struct {
	exact  string
	optn   string
	vari   string
	regexp []string
}

// set requires the line length to be valid, i.e. at least one rune for mask mode
func (segs *maskDefns) set(line string) (replaces string, err error) {
	mode, rsz := utf8.DecodeRuneInString(line)
	switch mode {
	case ArgMaskExact:
		if segs.exact != "" {
			replaces = segs.exact
		}
		segs.exact = line[rsz:]
	case ArgMaskOpt:
		if segs.optn != "" {
			replaces = segs.optn
		}
		segs.optn = line[rsz:]
	case ArgMaskVar:
		if segs.vari != "" {
			replaces = segs.vari
		}
		segs.vari = line[rsz:]
	case ArgRegexp:
		segs.regexp = append(segs.regexp, line[rsz:])
	default:
		err = fmt.Errorf("unknown mask mode '%c'", mode)
	}
	return replaces, err
}

func (segs *maskDefns) applyTo(rl *RefLine) error {
	if segs == nil {
		return nil
	}
	if segs.exact != "" {
		if err := rl.masksPattern(segs.exact, ArgMaskExact); err != nil {
			return err
		}
	}
	if segs.optn != "" {
		if err := rl.masksPattern(segs.optn, ArgMaskOpt); err != nil {
			return err
		}
	}
	if segs.vari != "" {
		if err := rl.masksPattern(segs.vari, ArgMaskVar); err != nil {
			return err
		}
	}
	for _, regexp := range segs.regexp {
		if err := rl.regexpArg(regexp); err != nil {
			return err
		}
	}
	return nil
}

func globalsFilter(t byte) bool { return t != TagRefLine }

// tags creates a line tag filter for several tags
func tags(tags ...byte) func(byte) bool {
	return func(t byte) bool {
		for _, u := range tags {
			if t == u {
				return true
			}
		}
		return false
	}
}

func eachTagLine(
	rd *bufio.Reader,
	lno *int,
	filter func(tag byte) bool,
	do func(line string) error,
) error {
	for {
		p, err := rd.Peek(1)
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case p[0] == TagComment:
			if _, err = rd.ReadBytes('\n'); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			continue
		case !filter(p[0]):
			return nil
		}
		line, err := readLine(rd, lno)
		if err != nil && err != io.EOF {
			return err
		}
		if err = do(line); err != nil {
			return err
		}
	}
}

// nextTag peeks into rd to detect next line tag
func nextTag(rd *bufio.Reader) (byte, error) {
	p, err := rd.Peek(1)
	switch {
	case err == io.EOF:
		return 0, nil
	case err != nil:
		return 0, err
	}
	return p[0], nil
}

// readLine makes keeping line numbers consistent more easy
func readLine(rd *bufio.Reader, lno *int) (line string, err error) {
	for {
		line, err = rd.ReadString('\n')
		line = strings.TrimRight(line, "\n\r")
		*lno += 1
		if err != nil || (len(line) > 0 && line[0] != TagComment) {
			break
		}
	}
	return line, err
}
