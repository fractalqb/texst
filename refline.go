package texst

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

type RefLine struct {
	lineTemplate
	igName rune
	text   string
	rgx    *regexp.Regexp
	lsNext *RefLine
}

func (rl *RefLine) IGroup() rune   { return rl.igName }
func (rl *RefLine) Text() string   { return rl.text }
func (rl *RefLine) Regexp() string { return rl.rgx.String() }

func (rl *RefLine) match(line []byte) (match []int) {
	match = rl.rgx.FindSubmatchIndex(line)
	return match
}

func (rl *RefLine) regexp() string {
	var sb strings.Builder
	sb.WriteRune('^')
	ln := []rune(rl.text)
	lidx := 0
	for _, seg := range rl.masks {
		if lidx < seg.start {
			sb.WriteString(regexp.QuoteMeta(string(ln[lidx:seg.start])))
		}
		lidx = seg.end()
		seg.writeRegexp(&sb)
	}
	sb.WriteString(regexp.QuoteMeta(string(ln[lidx:])))
	sb.WriteRune('$')
	return sb.String()
}

type lineTemplate struct {
	srcName string
	srcLine int
	masks   []*Mask
}

func (rl *lineTemplate) SourceName() string { return rl.srcName }
func (rl *lineTemplate) SourceLine() int    { return rl.srcLine }
func (rl *lineTemplate) Masks() []*Mask     { return rl.masks } // TODO return []Mask?

func (rl *lineTemplate) addMask(s *Mask) error {
	if s.empty() {
		return fmt.Errorf("mask %s is empty", s)
	}
	for _, es := range rl.masks {
		if _, ol := es.overlap(s); ol > 0 {
			return fmt.Errorf("mask %s overlaps %s", s, es)
		}
	}
	ins, _ := slices.BinarySearchFunc(rl.masks, s, segCmpr)
	rl.masks = slices.Insert(rl.masks, ins, s)
	return nil
}

type maskType int32

const (
	maskFix     maskType = iota // .
	mask0OrMore                 // *
	mask1OrMore                 // +
	mask0UpTo                   // 0
	mask1UpTo                   // 1
	maskAtLeast                 // -
	maskClass                   // ?
	maskMatch                   // ~
)

func parseMaskType(r rune) (maskType, error) {
	st := strings.IndexRune(".*+01-?~", r)
	if st < 0 {
		return maskType(-1), fmt.Errorf("illegal mask type '%c'", r)
	}
	return maskType(st), nil
}

type Mask struct {
	name       rune
	typ        maskType
	start, len int
	match      string
	checks     []SegChecker
}

func (s *Mask) Start() int { return s.start }
func (s *Mask) Len() int   { return s.len }

func (s *Mask) empty() bool { return s.len == 0 }

func (s *Mask) end() int { return s.start + s.len }

func (s *Mask) overlap(with *Mask) (start, len int) {
	se, we := s.start+s.len, with.start+with.len
	if s.start <= with.start {
		start = with.start
		if se > with.start {
			len = se - with.start
		}
	} else {
		start = s.start
		if we > s.start {
			len = we - s.start
		}
	}
	return
}

func (s *Mask) writeRegexp(w io.Writer) {
	class := "."
	if s.match != "" {
		class = s.match
	}
	switch s.typ {
	case maskFix:
		fmt.Fprintf(w, "(%s{%d})", class, s.len)
	case mask0OrMore:
		fmt.Fprintf(w, "(%s{0,})", class)
	case mask1OrMore:
		fmt.Fprintf(w, "(%s{1,})", class)
	case mask0UpTo:
		fmt.Fprintf(w, "(%s{0,%d})", class, s.len)
	case mask1UpTo:
		fmt.Fprintf(w, "(%s{1,%d})", class, s.len)
	case maskAtLeast:
		fmt.Fprintf(w, "(%s{%d,})", class, s.len)
	case maskMatch:
		fmt.Fprintf(w, "(%s)", s.match)
	default:
		panic(fmt.Sprintf("Mask.writeRegexp(): illegal typ %d", s.typ))
	}
}

func (s *Mask) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%c:%d+%d", s.name, s.start, s.len)
	if s.match != "" {
		fmt.Fprintf(&sb, ":%s", s.match)
	}
	sb.WriteByte(']')
	return sb.String()
}

func segCmpr(s, t *Mask) int { return s.start - t.start }

type SegChecker interface {
	Check(seg []byte) error
}
