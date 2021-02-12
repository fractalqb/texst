package texst

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"git.fractalqb.de/fractalqb/icontainer/islist"
)

// Types of argument lines
const (
	// The matching part of the subject must have the same length as the
	// reference line segment.
	ArgSegExact = '='
	// The matching part of the subject can be of any length, even zero.
	ArgSegOpt = '*'
	// The matching part of the subject can be of any length greater than zero.
	ArgSegVar = '+'
)

type segment struct {
	name             rune
	start, end       int
	refStart, refEnd int
	mode             byte
}

func (s *segment) len() int { return s.end - s.start }

func (s *segment) empty() bool { return s.end <= s.start }

func (s *segment) sub(m *segment) (split *segment) {
	if s.start < m.start {
		if s.end > m.end {
			split = &segment{
				name:  s.name,
				start: m.end, end: s.end,
				refStart: m.refEnd, refEnd: s.refEnd,
				mode: s.mode,
			}
			s.end = m.start
			s.refEnd = m.refStart
			return split
		}
		if s.end > m.start {
			s.end = m.start
			s.refEnd = m.refStart
		}
	} else if s.end <= m.end {
		s.end = s.start
		s.refEnd = s.refStart
	} else if m.end > s.start {
		s.start = m.end
		s.refStart = m.refEnd
	}
	return nil
}

type RefLine struct {
	igroup   rune
	text     string
	segs     []segment
	srcLn    int
	islsNext *RefLine
}

func newRefLine() *RefLine {
	res := new(RefLine)
	return res
}

// IGroup returns the name of the line's interleaving group
func (rl *RefLine) IGroup() rune { return rl.igroup }

// Text returns the verbatim reference text
func (rl *RefLine) Text() string { return rl.text }

func (rl *RefLine) addSegment(name rune, mode byte, start, end int) {
	seg := segment{
		name:  name,
		mode:  mode,
		start: start, end: end,
		refStart: utf8Start(rl.text, start),
	}
	seg.refEnd = seg.refStart + utf8Start(rl.text[seg.refStart:], end-start)
	if len(rl.segs) == 0 {
		rl.segs = []segment{seg}
		return
	}
	var nsegs []segment
	inseg := true
	for i := range rl.segs {
		rseg := &rl.segs[i]
		before := rseg.start < seg.start
		split := rseg.sub(&seg)
		if inseg {
			if before {
				if !rseg.empty() {
					nsegs = append(nsegs, *rseg)
				}
				if split != nil {
					nsegs = append(nsegs, seg, *split)
					inseg = false
				}
			} else {
				nsegs = append(nsegs, seg)
				inseg = false
				if !rseg.empty() {
					nsegs = append(nsegs, *rseg)
				}
			}
		} else if !rseg.empty() {
			nsegs = append(nsegs, *rseg)
		}
	}
	if inseg {
		nsegs = append(nsegs, seg)
	}
	rl.segs = nsegs
}

func (rl *RefLine) matches(subj string) bool {
	if len(rl.segs) == 0 {
		return rl.text == subj
	}
	return rl.match(0, subj, true)
}

func utf8Start(s string, runeStart int) (res int) {
	for runeStart > 0 {
		_, rsz := utf8.DecodeRuneInString(s)
		res += rsz
		runeStart--
	}
	return res
}

func (rl *RefLine) preSegPart(s int) string {
	switch {
	case s >= len(rl.segs):
		seg := &rl.segs[len(rl.segs)-1]
		return rl.text[seg.refEnd:]
	case s == 0:
		return rl.text[:rl.segs[0].refStart]
	}
	seg0 := &rl.segs[s-1]
	seg1 := &rl.segs[s]
	return rl.text[seg0.refEnd:seg1.refStart]
}

// backtracking because fix parts can be in many positions with variable segments
// TODO could sequence aligning be better (eliminate recurion to save stack?)
func (rl *RefLine) match(preSeg int, subj string, fix bool) bool {
	refstr := rl.preSegPart(preSeg)
	if preSeg >= len(rl.segs) {
		if fix {
			return refstr == subj
		}
		return strings.HasSuffix(subj, refstr)
	}
	seg := &rl.segs[preSeg]
	afterSeg := func(subj string) bool {
		switch seg.mode {
		case ArgSegExact:
			if sl := seg.len(); sl > len(subj) {
				return false
			} else {
				return rl.match(preSeg+1, subj[sl:], true)
			}
		case ArgSegOpt:
			return rl.match(preSeg+1, subj, false)
		case ArgSegVar:
			if subj == "" {
				return false
			}
			return rl.match(preSeg+1, subj[1:], false)
		}
		panic("unreachable code")
	}
	if refstr == "" {
		return afterSeg(subj)
	}
	if fix {
		if !strings.HasPrefix(subj, refstr) {
			return false
		}
		return afterSeg(subj[len(refstr):])
	}
	for {
		refpos := strings.Index(subj, refstr)
		if refpos < 0 {
			return false
		}
		if afterSeg(subj[refpos+len(refstr):]) {
			return true
		}
		subj = subj[refpos+1:]
		if len(subj) < len(refstr) {
			return false
		}
	}
}

func (rl *RefLine) read(rd *bufio.Reader, gsegs string, lno *int) error {
	rl.srcLn = *lno
	line, err := readLine(rd, lno)
	switch {
	case err == io.EOF:
		tag, igroup, tail := lineHead(line)
		switch {
		case tag != TagRefLine:
			return fmt.Errorf("syntax error: not a reference line: '%s'", line)
		case igroup == 0:
			return errors.New("syntax error: incomplete reference line")
		}
		rl.igroup = igroup
		rl.text = line[tail:]
		rl.segs = nil
		if len(gsegs) > 2 {
			rl.lineSegs(gsegs[2:], gsegs[1])
		}
		return nil
	case err != nil:
		return err
	}
	tag, igroup, tail := lineHead(line)
	switch {
	case tag != TagRefLine:
		return fmt.Errorf("syntax error: not a reference line: '%s'", line)
	case igroup == 0:
		return errors.New("syntax error: incomplete reference line")
	}
	rl.igroup = igroup
	rl.text = line[tail:]
	rl.segs = nil
	if len(gsegs) > 2 {
		rl.lineSegs(gsegs[2:], gsegs[1])
	}
	err = eachTagLine(rd, lno, tags(TagRefArgs), func(line string) error {
		if len(line) < 2 {
			return errors.New("syntax error: incomplete args line")
		}
		switch line[1] {
		case ArgSegExact, ArgSegOpt, ArgSegVar:
			err = rl.lineSegs(line[2:], line[1])
		default:
			err = fmt.Errorf("invalid args tag '%c'", line[1])
		}
		return err
	})
	return err
}

func (rl *RefLine) lineSegs(pattern string, mode byte) (err error) {
	if pattern == "" {
		return errors.New("empty line segments pattern")
	}
	var (
		cStart = -1
		cName  = rune(' ')
	)
	for i, r := range pattern {
		if r == ' ' {
			if cName != ' ' {
				rl.addSegment(cName, mode, cStart, i)
				cStart = -1
				cName = ' '
			}
		} else if r != cName {
			if cName != ' ' {
				rl.addSegment(cName, mode, cStart, i)
			}
			cStart = i
			cName = r
		}
	}
	if cName != ' ' {
		rl.addSegment(cName, mode, cStart, len(pattern))
	}
	return nil
}

func lineHead(line string) (tag byte, igroup rune, tailAt int) {
	switch len(line) {
	case 0:
		return 0, 0, 0
	case 1:
		return line[0], 0, 0
	}
	tag = line[0]
	igroup, tailAt = utf8.DecodeRuneInString(line[1:])
	return tag, igroup, tailAt + 1
}

// ListNext to implement intrusive singly linked list
func (rl *RefLine) ListNext() islist.Node { return rl.islsNext }

// SetListNext to implement intrusive singly linked list
func (rl *RefLine) SetListNext(n islist.Node) {
	if n == nil {
		rl.islsNext = nil
	} else {
		rl.islsNext = n.(*RefLine)
	}
}
