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

func (rl *RefLine) IGroup() rune { return rl.igName }
func (rl *RefLine) Text() string { return rl.text }

func (rl *RefLine) match(line []byte) (match []int) {
	match = rl.rgx.FindSubmatchIndex(line)
	return match
}

func (rl *RefLine) regexp() string {
	var sb strings.Builder
	sb.WriteRune('^')
	ln := []rune(rl.text)
	lidx := 0
	for _, seg := range rl.segs {
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
	segs    []*segment
}

func (rl *lineTemplate) SourceName() string { return rl.srcName }
func (rl *lineTemplate) SourceLine() int    { return rl.srcLine }

func (rl *lineTemplate) addSeg(s *segment) error {
	if s.empty() {
		return fmt.Errorf("segment %s is empty", s)
	}
	for _, es := range rl.segs {
		if _, ol := es.overlap(s); ol > 0 {
			return fmt.Errorf("segment %s overlaps %s", s, es)
		}
	}
	ins, _ := slices.BinarySearchFunc(rl.segs, s, segCmpr)
	rl.segs = slices.Insert(rl.segs, ins, s)
	return nil
}

type segType int32

const (
	segFix     segType = iota // .
	seg0OrMore                // *
	seg1OrMore                // +
	seg0UpTo                  // 0
	seg1UpTo                  // 1
	segAtLeast                // -
	segMatch                  // ~
	segClass                  // ?
)

func parseSegType(r rune) (segType, error) {
	st := strings.IndexRune(".*+01-~?", r)
	if st < 0 {
		return segType(-1), fmt.Errorf("illegal segemnt type '%c'", r)
	}
	return segType(st), nil
}

type segment struct {
	name       rune
	typ        segType
	start, len int
	match      string
	checks     []SegChecker
}

func (s *segment) empty() bool { return s.len == 0 }

func (s *segment) end() int { return s.start + s.len }

func (s *segment) overlap(with *segment) (start, len int) {
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

func (s *segment) writeRegexp(w io.Writer) {
	class := "."
	if s.match != "" {
		class = s.match
	}
	switch s.typ {
	case segFix:
		fmt.Fprintf(w, "(%s{%d})", class, s.len)
	case seg0OrMore:
		fmt.Fprintf(w, "(%s{0,})", class)
	case seg1OrMore:
		fmt.Fprintf(w, "(%s{1,})", class)
	case seg0UpTo:
		fmt.Fprintf(w, "(%s{0,%d})", class, s.len)
	case seg1UpTo:
		fmt.Fprintf(w, "(%s{1,%d})", class, s.len)
	case segAtLeast:
		fmt.Fprintf(w, "(%s{%d,})", class, s.len)
	case segMatch:
		fmt.Fprintf(w, "(%s)", s.match)
	default:
		panic(fmt.Sprintf("segment.writeRegexp(): illegal typ %d", s.typ))
	}
}

func (s *segment) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%c:%d+%d", s.name, s.start, s.len)
	if s.match != "" {
		fmt.Fprintf(&sb, ":%s", s.match)
	}
	sb.WriteByte(']')
	return sb.String()
}

func segCmpr(s, t *segment) int { return s.start - t.start }

type SegChecker interface {
	Check(seg []byte) error
}
