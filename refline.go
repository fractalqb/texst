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
	ArgMaskExact = '='
	// The matching part of the subject can be of any length, even zero.
	ArgMaskOpt = '*'
	// The matching part of the subject can be of any length greater than zero.
	ArgMaskVar = '+'
)

type matchMode byte

func parseMatchMode(c byte) (matchMode, error) {
	switch c {
	case ArgMaskExact, ArgMaskOpt, ArgMaskVar:
		return matchMode(c), nil
	}
	return 0, fmt.Errorf("invalid match mode '%c'", c)
}

func (mm matchMode) locate(subj, refseg string, start int) int {
	switch mm {
	case ArgMaskExact:
		if len(subj) < start {
			return -1
		}
		if strings.HasPrefix(subj[start:], refseg) {
			return 0
		}
		return -1
	case ArgMaskOpt:
		if len(subj) < start {
			return -1
		}
		pos := strings.Index(subj[start:], refseg)
		if pos < 0 {
			return -1
		}
		return pos
	case ArgMaskVar:
		start++
		if len(subj) < start {
			return -1
		}
		pos := strings.Index(subj[start:], refseg)
		if pos < 0 {
			return -1
		}
		return pos + 1
	}
	panic(fmt.Errorf("unknown match mode '%c'", mm))
}

type mask struct {
	name             rune
	start, end       int
	refStart, refEnd int
	mode             matchMode
}

func (s *mask) len() int { return s.end - s.start }

func (s *mask) empty() bool { return s.end <= s.start }

// sub removes the runes covered by mask m from the mask s. If m is in the mid
// of s, split will be the rightmost remaining part opf s.
func (s *mask) sub(m *mask) (split *mask) {
	if s.start < m.start {
		if s.end > m.end {
			split = &mask{
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

// RefLine represents a line of reference text with its arguments. API users
// will get current reference lines when using the MismatchFunc.
type RefLine struct {
	igroup   rune
	text     string
	masks    []mask
	srcLn    int
	islsNext *RefLine
}

func newRefLine() *RefLine {
	res := new(RefLine)
	return res
}

// Line returns the line number in the refrecence text specification.
func (rl *RefLine) Line() int { return rl.srcLn + 1 }

// IGroup returns the name of the line's interleaving group.
func (rl *RefLine) IGroup() rune { return rl.igroup }

// Text returns the verbatim reference text.
func (rl *RefLine) Text() string { return rl.text }

func (rl *RefLine) addMask(name rune, mode matchMode, start, end int) {
	newmask := mask{
		name:  name,
		mode:  mode,
		start: start, end: end,
		refStart: utf8Start(rl.text, start),
	}
	newmask.refEnd = newmask.refStart + utf8Start(rl.text[newmask.refStart:], end-start)
	if len(rl.masks) == 0 {
		rl.masks = []mask{newmask}
		return
	}
	var newmasks []mask
	insmask := true
	for i := range rl.masks {
		oldmask := &rl.masks[i]
		before := oldmask.start < newmask.start
		split := oldmask.sub(&newmask)
		if insmask {
			if before {
				if !oldmask.empty() {
					newmasks = append(newmasks, *oldmask)
				}
				if split != nil {
					newmasks = append(newmasks, newmask, *split)
					insmask = false
				}
			} else {
				newmasks = append(newmasks, newmask)
				insmask = false
				if !oldmask.empty() {
					newmasks = append(newmasks, *oldmask)
				}
			}
		} else if !oldmask.empty() {
			newmasks = append(newmasks, *oldmask)
		}
	}
	if insmask {
		newmasks = append(newmasks, newmask)
	}
	rl.masks = newmasks
}

func utf8Start(s string, toRune int) (res int) {
	for toRune > 0 {
		_, rsz := utf8.DecodeRuneInString(s)
		s = s[rsz:]
		res += rsz
		toRune--
	}
	return res
}

func (rl *RefLine) matches(sline string) (err error) {
	if len(rl.masks) == 0 {
		if rl.text == sline {
			return nil
		}
		return errors.New("verbatim line mismatch")
	}
	if sline, err = rl.matchPrefix(sline); err != nil {
		return err
	}
	type subjmatch struct{ start, end int }
	midx := 0
	smsegs := make([]subjmatch, len(rl.masks))
	smsegs[0] = subjmatch{start: 0, end: -1}
	midxmax := -1
	backtrack := func(msg, reftxt string, pos int) {
		if err == nil || midx > midxmax {
			midxmax = midx
			err = fmt.Errorf(msg, reftxt, pos)
		}
		midx--
	}
	for {
		sseg := &smsegs[midx]
		mask := &rl.masks[midx]
		if sseg.end < 0 { // not backtracking
			switch mask.mode {
			case ArgMaskExact:
				sseg.end = sseg.start + utf8Start(sline, mask.len())
				reftxt, final := rl.postMaskSeg(midx)
				if mask.mode.locate(sline, reftxt, sseg.end) >= 0 {
					if final {
						return nil
					} else {
						midx++
						smsegs[midx] = subjmatch{
							start: sseg.end + len(reftxt),
							end:   -1,
						}
					}
				} else {
					backtrack("cannot find ref text '%s' at %d", reftxt, sseg.end)
					if midx < 0 {
						return err
					}
				}
			case ArgMaskOpt, ArgMaskVar:
				reftxt, final := rl.postMaskSeg(midx)
				if pos := mask.mode.locate(sline, reftxt, sseg.start); pos >= 0 {
					sseg.end = sseg.start + pos
					if final {
						return nil
					} else {
						midx++
						smsegs[midx] = subjmatch{
							start: sseg.end + len(reftxt),
							end:   -1,
						}
					}
				} else {
					backtrack("cannot find ref text '%s' after %d", reftxt, sseg.start)
					if midx < 0 {
						return err
					}
				}
			default:
				panic(fmt.Errorf("unknown mask mode '%c", mask.mode))
			}
		} else { // backtracking
			switch mask.mode {
			case ArgMaskExact:
				reftxt, _ := rl.postMaskSeg(midx)
				return fmt.Errorf("match fails after reference text '%s'", reftxt)
			case ArgMaskOpt, ArgMaskVar:
				reftxt, final := rl.postMaskSeg(midx)
				pos := matchMode(ArgMaskVar).locate(sline, reftxt, sseg.end)
				if pos >= 0 {
					sseg.end += pos
					if final {
						return nil
					} else {
						midx++
						smsegs[midx] = subjmatch{
							start: sseg.end + len(reftxt),
							end:   -1,
						}
					}
				} else {
					backtrack("cannot find ref text '%s' after %d", reftxt, sseg.end)
					if midx < 0 {
						return err
					}
				}
			default:
				panic(fmt.Errorf("unknown mask mode '%c", mask.mode))
			}
		}
	}
}

func (rl *RefLine) matchPrefix(sline string) (string, error) {
	mask := &rl.masks[0]
	if mask.refStart == 0 {
		return sline, nil
	}
	if len(sline) < mask.refStart {
		return sline, errors.New(
			"subject line is shorter than initial reference segment",
		)
	}
	if sline[:mask.refStart] != rl.text[:mask.refStart] {
		return sline, errors.New("mismatch in initial reference segment")
	}
	return sline[mask.refStart:], nil
}

func (rl *RefLine) postMaskSeg(midx int) (seg string, final bool) {
	mask := &rl.masks[midx]
	if midx+1 < len(rl.masks) {
		next := &rl.masks[midx+1]
		return rl.text[mask.refEnd:next.refStart], false
	}
	return rl.text[mask.refEnd:], true
}

func (rl *RefLine) read(rd *bufio.Reader, gmasks string, lno *int) error {
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
		rl.masks = nil
		if len(gmasks) > 2 {
			mode, err := parseMatchMode(gmasks[1])
			if err != nil {
				return err
			}
			rl.masksPattern(gmasks[2:], mode)
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
	rl.masks = nil
	if len(gmasks) > 2 {
		mode, err := parseMatchMode(gmasks[1])
		if err != nil {
			return err
		}
		rl.masksPattern(gmasks[2:], mode)
	}
	err = eachTagLine(rd, lno, tags(TagRefArgs), func(line string) error {
		if len(line) < 2 {
			return errors.New("syntax error: incomplete args line")
		}
		if mode, err := parseMatchMode(line[1]); err != nil {
			return err
		} else {
			// currently there are no other args line tags
			return rl.masksPattern(line[2:], mode)
		}
		return nil
	})
	return err
}

func (rl *RefLine) masksPattern(pattern string, mode matchMode) (err error) {
	if pattern == "" {
		return errors.New("empty line masks pattern")
	}
	var (
		cStart = -1
		cName  = rune(' ')
	)
	for i, r := range pattern {
		if r == ' ' {
			if cName != ' ' {
				rl.addMask(cName, mode, cStart, i)
				cStart = -1
				cName = ' '
			}
		} else if r != cName {
			if cName != ' ' {
				rl.addMask(cName, mode, cStart, i)
			}
			cStart = i
			cName = r
		}
	}
	if cName != ' ' {
		rl.addMask(cName, mode, cStart, len(pattern))
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
