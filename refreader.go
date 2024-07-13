package texst

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RefReader struct {
	src    string
	rd     io.Reader
	scn    *bufio.Scanner
	ll     []byte
	lno    int
	ilgs   []rune
	globLT *lineTemplate

	rlPool *RefLine
}

func NewRefReader(name string, r io.Reader) (*RefReader, error) {
	if r == nil {
		return nil, errors.New("nil reader")
	}
	rr := &RefReader{
		src: name,
		rd:  r,
		scn: bufio.NewScanner(r),
	}
	if err := rr.preamble(); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, lineErrorf(rr, "no reference line after preamble")
		}
		return nil, lineError(rr, err)
	}
	if len(rr.ilgs) == 0 {
		rr.ilgs = []rune{' '}
	}
	return rr, nil
}

func NewRefString(name, texts string) (*RefReader, error) {
	return NewRefReader(name, strings.NewReader(texts))
}

func OpenRefFile(file string) (*RefReader, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return NewRefReader(file, r)
}

func (rr *RefReader) Close() error {
	rr.scn = nil
	if c, ok := rr.rd.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (rr *RefReader) Name() string { return rr.src }

func (rr *RefReader) Line() int { return rr.lno }

func (rr *RefReader) IGroups() []rune { return rr.ilgs }

func (rr *RefReader) NextLine() (*RefLine, error) {
	if rr.ll == nil {
		if err := rr.scan(); err != nil {
			return nil, lineError(rr, err)
		}
	}
	c0, c1, line, err := rr.tokenize()
	if err != nil {
		return nil, lineError(rr, err)
	}
	if c0 != TagRefLine {
		return nil, lineErrorf(rr,
			"expect reference line marker '%c', have '%c'",
			TagRefLine,
			c0,
		)
	}
	rr.ll = nil
	rl := rr.newLine(c1, string(line))
	if rr.globLT != nil {
		rl.masks = slices.Clone(rr.globLT.masks)
	}
	err = rr.argLines(&rl.lineTemplate)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if rl.rgx, err = regexp.Compile(rl.regexp()); err != nil {
		return nil, lineError(rr, err)
	}
	return rl, nil
}

func (rr *RefReader) FreeLine(rl *RefLine) {
	*rl = RefLine{}
	rl.lsNext = rr.rlPool
	rr.rlPool = rl
}

func (rr *RefReader) newLine(ig rune, txt string) (rl *RefLine) {
	if rr.rlPool == nil {
		rl = &RefLine{
			lineTemplate: lineTemplate{
				srcName: rr.Name(),
				srcLine: rr.Line(),
			},
			igName: ig,
			text:   txt,
		}
	} else {
		rl = rr.rlPool
		rr.rlPool = rl.lsNext
		rl.srcName = rr.Name()
		rl.srcLine = rr.Line()
		rl.igName = ig
		rl.text = txt
		rl.lsNext = nil
	}
	return rl
}

func (rr *RefReader) argLines(rl *lineTemplate) error {
	for {
		if err := rr.scan(); err != nil {
			return err
		}
		c0, c1, line, err := rr.tokenize()
		if err != nil {
			return err
		}
		if c0 != TagRefLineArg {
			break
		}
		rr.ll = nil
		segType, err := parseMaskType(c1)
		if err != nil {
			return fmt.Errorf("arg line: %w", err)
		}
		switch segType {
		case maskMatch:
			if err = rr.match(rl, line); err != nil {
				return err
			}
		case maskClass:
			if err = rr.class(rl, line); err != nil {
				return err
			}
		default:
			if err = rr.masks(rl, segType, line); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rr *RefReader) class(rl *lineTemplate, line []byte) error {
	nm, sz := utf8.DecodeRune(line)
	if nm == utf8.RuneError {
		return lineErrorf(rr, "rune error for mask name")
	}
	line = bytes.TrimSpace(line[sz:])
	for _, seg := range rl.masks {
		if seg.name != nm {
			continue
		}
		if seg.typ == maskMatch {
			return lineErrorf(rr,
				"must not set rune class on matching mask '%c'",
				nm,
			)
		}
		seg.match = string(line)
	}
	return nil
}

func (rr *RefReader) match(rl *lineTemplate, line []byte) error {
	nm, sz := utf8.DecodeRune(line)
	if nm == utf8.RuneError {
		return lineErrorf(rr, "rune error for mask name")
	}
	line = bytes.TrimSpace(line[sz:])
	for _, seg := range rl.masks {
		if seg.name != nm {
			continue
		}
		if seg.typ != maskFix {
			return lineErrorf(rr,
				"must not match segemnt '%c' with length constraint",
				nm,
			)
		}
		seg.typ = maskMatch
		seg.match = string(line)
	}
	return nil
}

func (rr *RefReader) masks(rl *lineTemplate, st maskType, l []byte) error {
	name := ' '
	start, length := 0, 0
	for i := 0; len(l) > 0; i++ {
		c, csz := utf8.DecodeRune(l)
		if c == utf8.RuneError {
			return fmt.Errorf("invalid rune at %d", i)
		}
		l = l[csz:]
		length++
		if c == name {
			continue
		}
		if !unicode.IsSpace(name) {
			s := &Mask{
				name:  name,
				typ:   st,
				start: start,
				len:   i - start,
			}
			if err := rl.addMask(s); err != nil {
				return err
			}
		}
		name = c
		start = i
	}
	if !unicode.IsSpace(name) {
		s := &Mask{
			name:  name,
			typ:   st,
			start: start,
			len:   length - start,
		}
		if err := rl.addMask(s); err != nil {
			return err
		}
	}
	return nil
}

var notIGroup = string([]byte{TagComment, TagIGroup, TagGlobalArg, TagRefLine})

func (rr *RefReader) preamble() error {
	for {
		if err := rr.scan(); err != nil {
			return err
		}
		c0, c1, line, err := rr.tokenize()
		if err != nil {
			return err
		}
		if c0 == TagRefLine {
			return nil
		}
		switch c0 {
		case TagGlobalArg:
			segType, err := parseMaskType(c1)
			if err != nil {
				return err
			}
			if rr.globLT == nil {
				rr.globLT = &lineTemplate{srcName: rr.Name(), srcLine: rr.Line()}
			}
			switch segType {
			case maskMatch:
				if err := rr.match(rr.globLT, line); err != nil {
					return err
				}
			case maskClass:
				if err = rr.class(rr.globLT, line); err != nil {
					return err
				}
			default:
				if err = rr.masks(rr.globLT, segType, line); err != nil {
					return err
				}
			}
		case TagIGroup:
			switch c1 {
			case TagIGroup:
				if rr.ilgs != nil {
					return lineErrorf(rr, "redefining interleafing groups")
				}
				if i := bytes.IndexAny(line, notIGroup); i >= 0 {
					nig, _ := utf8.DecodeRune(line[i:])
					return lineErrorf(rr,
						"illegal interleaving group name '%c' (not allowed: %s)",
						nig,
						notIGroup,
					)
				}
				rr.ilgs = []rune(string(line))
			default:
				return lineErrorf(rr, "invalid preamble line %c%c…", c0, c1)
			}
		default:
			return lineErrorf(rr, "invalid preamble line %c…", c0)
		}
		rr.ll = nil
	}
}

func (rr *RefReader) scan() error {
	for {
		if !rr.scn.Scan() {
			rr.ll = nil
			return io.EOF
		}
		l := rr.scn.Bytes()
		rr.lno++
		if trimmed := bytes.TrimSpace(l); len(trimmed) == 0 {
			rr.ll = nil
			return errors.New("empty reference line")
		}
		if l[0] != '#' {
			rr.ll = l
			break
		}
	}
	return nil
}

func (rr *RefReader) tokenize() (c0, c1 rune, rest []byte, err error) {
	line := rr.ll
	c0, csz := utf8.DecodeRune(line)
	if c0 == utf8.RuneError {
		if csz == 0 {
			return 0, 0, line, errors.New("empty reference line")
		}
		return 0, 0, line, errors.New("invalid UTF-8 encoding in column 0")
	}
	line = line[csz:]
	c1, csz = utf8.DecodeRune(line)
	if c1 == utf8.RuneError {
		if csz == 0 {
			return 0, 0, line, errors.New("incomplete reference line")
		}
		return 0, 0, line, errors.New("invalid UTF-8 encoding in column 1")
	}
	rest = line[csz:]
	return c0, c1, rest, nil
}
