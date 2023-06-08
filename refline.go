package texst

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"git.fractalqb.de/fractalqb/icontainer"
)

// Types of argument lines
const (
	// The masked part of the subject must have the same length as the reference
	// line segment.
	ArgMaskExact = '='
	// The masked part of the subject can be of any length, even zero.
	ArgMaskOpt = '*'
	// The masked part of the subject can be of any length greater than zero.
	ArgMaskVar = '+'
	// Match masked parts against a regular expression. TODO syntax of the line…
	ArgRegexp = '~'
)

type matchMode byte

func parseMatchMode(c byte) (matchMode, error) {
	switch c {
	case ArgMaskExact, ArgMaskOpt, ArgMaskVar:
		return matchMode(c), nil
	}
	return 0, fmt.Errorf("invalid match mode '%c'", c)
}

func (mm matchMode) locate(subj, refseg string, startAt int) int {
	switch mm {
	case ArgMaskExact:
		if len(subj) < startAt {
			return -1
		}
		if strings.HasPrefix(subj[startAt:], refseg) {
			return 0
		}
		return -1
	case ArgMaskOpt:
		if len(subj) < startAt {
			return -1
		}
		pos := strings.Index(subj[startAt:], refseg)
		if pos < 0 {
			return -1
		}
		return pos
	case ArgMaskVar:
		startAt++
		if len(subj) < startAt {
			return -1
		}
		pos := strings.Index(subj[startAt:], refseg)
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
	checker          func(ref, subj string) error
}

func (m *mask) len() int { return m.end - m.start }

func (m *mask) empty() bool { return m.end <= m.start }

func (m *mask) check(rline, sseg string) error {
	if m.checker == nil {
		return nil
	}
	rseg := rline[m.refStart:m.refEnd]
	return m.checker(rseg, sseg)
}

// sub removes the runes covered by mask m from the mask s. If m is in the mid
// of s, split will be the rightmost remaining part opf s.
func (m *mask) sub(s *mask) (split *mask) {
	if m.start < s.start {
		if m.end > s.end {
			split = &mask{
				name:  m.name,
				start: s.end, end: m.end,
				refStart: s.refEnd, refEnd: m.refEnd,
				mode: m.mode,
			}
			m.end = s.start
			m.refEnd = s.refStart
			return split
		}
		if m.end > s.start {
			m.end = s.start
			m.refEnd = s.refStart
		}
	} else if m.end <= s.end {
		m.end = m.start
		m.refEnd = m.refStart
	} else if s.end > m.start {
		m.start = s.end
		m.refStart = s.refEnd
	}
	return nil
}

// RefLine represents a line of reference text with its arguments. API users
// will get current reference lines when using the MismatchFunc.
type RefLine struct {
	igroup rune
	text   string
	masks  []mask
	srcLn  int
	icontainer.SListNode[*RefLine]
}

func newRefLine() *RefLine { return new(RefLine) }

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

type subjmatch struct{ start, end int }

func (sm subjmatch) of(line string) (string, error) {
	switch {
	case sm.start >= len(line):
		return "", errors.New("submatch after line end")
	case sm.end > len(line):
		return line[sm.start:], errors.New("submatch exceeds line")
	}
	return line[sm.start:sm.end], nil
}

func (rl *RefLine) matches(sline string) (err error) {
	if len(rl.masks) == 0 {
		if rl.text == sline {
			return nil
		}
		return errors.New("verbatim line mismatch")
	}
	smsegs := make([]subjmatch, len(rl.masks))
	pflen, err := rl.matchPrefix(sline)
	if err != nil {
		return err
	}
	smsegs[0] = subjmatch{start: pflen, end: -1}
	midx := 0
	midxmax := -1
	backtrack := func(format string, args ...interface{}) {
		if err == nil || midx > midxmax {
			midxmax = midx
			err = fmt.Errorf(format, args...)
		}
		midx--
	}
	for {
		if midx < 0 {
			return err
		}
		sseg := &smsegs[midx]
		mask := &rl.masks[midx]
		if sseg.end < 0 { // not backtracking
			switch mask.mode {
			case ArgMaskExact:
				sseg.end = sseg.start + utf8Start(sline, mask.len())
				reftxt, final := rl.postMaskSeg(midx)
				if segstr, err := sseg.of(sline); err != nil {
					backtrack("masked segment '%s': %s", segstr, err)
					continue
				} else if err = mask.check(rl.text, segstr); err != nil {
					backtrack("masked segment '%s': %s", segstr, err)
					continue
				}
				if mask.mode.locate(sline, reftxt, sseg.end) >= 0 {
					if final {
						return nil
					}
					midx++
					smsegs[midx] = subjmatch{
						start: sseg.end + len(reftxt),
						end:   -1,
					}
				} else {
					backtrack("cannot find ref text '%s' at %d", reftxt, sseg.end)
				}
			case ArgMaskOpt, ArgMaskVar:
				reftxt, final := rl.postMaskSeg(midx)
				if pos := mask.mode.locate(sline, reftxt, sseg.start); pos >= 0 {
					sseg.end = sseg.start + pos
					if segstr, err := sseg.of(sline); err != nil {
						backtrack("masked segment '%s': %s", segstr, err)
						continue
					} else if err = mask.check(rl.text, segstr); err != nil {
						backtrack("masked segment '%s': %s", segstr, err)
						continue
					}
					if final {
						return nil
					}
					midx++
					smsegs[midx] = subjmatch{
						start: sseg.end + len(reftxt),
						end:   -1,
					}
				} else {
					backtrack("cannot find ref text '%s' after %d", reftxt, sseg.start)
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
					if segstr, err := sseg.of(sline); err != nil {
						backtrack("masked segment '%s': %s", segstr, err)
						continue
					} else if err = mask.check(rl.text, segstr); err != nil {
						backtrack("masked segment '%s': %s", segstr, err)
						continue
					}
					if final {
						return nil
					}
					midx++
					smsegs[midx] = subjmatch{
						start: sseg.end + len(reftxt),
						end:   -1,
					}
				} else {
					backtrack("cannot find ref text '%s' after %d", reftxt, sseg.end)
				}
			default:
				panic(fmt.Errorf("unknown mask mode '%c", mask.mode))
			}
		}
	}
}

func (rl *RefLine) matchPrefix(sline string) (int, error) {
	mask := &rl.masks[0]
	if mask.refStart == 0 {
		return 0, nil
	}
	if len(sline) < mask.refStart {
		return 0, errors.New(
			"subject line is shorter than initial reference segment",
		)
	}
	if sline[:mask.refStart] != rl.text[:mask.refStart] {
		return 0, errors.New("mismatch in initial reference segment")
	}
	return mask.refStart, nil
}

func (rl *RefLine) postMaskSeg(midx int) (seg string, final bool) {
	mask := &rl.masks[midx]
	if midx+1 < len(rl.masks) {
		next := &rl.masks[midx+1]
		return rl.text[mask.refEnd:next.refStart], false
	}
	return rl.text[mask.refEnd:], true
}

var argRegexp = regexp.MustCompile(`^(.)(\[\d+\])? (.+)$`)

func (rl *RefLine) read(rd *bufio.Reader, gmasks map[rune]*maskDefns, lno *int) error {
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
		if err = gmasks[0].applyTo(rl); err != nil {
			return err
		}
		err = gmasks[igroup].applyTo(rl)
		return err
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
	if err = gmasks[0].applyTo(rl); err != nil {
		return err
	}
	if err = gmasks[igroup].applyTo(rl); err != nil {
		return err
	}
	err = eachTagLine(rd, lno, tags(TagRefArgs), func(line string) error {
		if len(line) < 2 {
			return errors.New("syntax error: incomplete args line")
		}
		if mode, err := parseMatchMode(line[1]); err == nil {
			return rl.masksPattern(line[2:], mode)
		}
		switch line[1] {
		case ArgRegexp:
			return rl.regexpArg(line[2:])
		default:
			return fmt.Errorf("unknown argument line '%c'", line[1])
		}
	})
	return err
}

func (rl *RefLine) regexpArg(arg string) (err error) {
	match := argRegexp.FindStringSubmatch(arg)
	if match == nil {
		return errors.New("syntax error in regexp argument line")
	}
	name, _ := utf8.DecodeRuneInString(match[1])
	num := 0
	if match[2] != "" {
		num, err = strconv.Atoi(strings.Trim(match[2], "[]"))
		if err != nil {
			return err
		}
	}
	rgx, err := regexp.Compile(match[3])
	if err != nil {
		return err
	}
	nmno, app := 0, 0
	for i := range rl.masks {
		m := &rl.masks[i]
		if m.name == name {
			nmno++
			if num == 0 || num == nmno {
				m.checker = func(_, s string) error {
					if rgx.MatchString(s) {
						return nil
					}
					return fmt.Errorf("regexp '%s' mismatch", match[3])
				}
				app++
			}
		}
	}
	if app == 0 {
		return fmt.Errorf("no mask '%c' to apply regexp '%s' to", name, match[3])
	}
	return nil
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
